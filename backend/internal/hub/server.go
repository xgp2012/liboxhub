package hub

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LiStudioorg/boxli/internal/auth"
	"github.com/LiStudioorg/boxli/internal/config"
)

type Server struct {
	cfg   config.Config
	store *store
	auth  *auth.Service
	oauth *auth.OAuthClient
	state *auth.StateStore
}

func New(cfg config.Config, pool *pgxpool.Pool) *Server {
	return &Server{
		cfg:   cfg,
		store: &store{pool: pool},
		auth:  auth.New(pool, cfg.JWTSecret, cfg.SessionTTLHours),
		oauth: auth.NewOAuthClient(cfg.GitHubClientID, cfg.GitHubSecret, cfg.GitHubRedirect),
		state: auth.NewStateStore(pool),
	}
}

type response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func writeJSON(w http.ResponseWriter, status int, resp response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeOK(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, response{Code: 0, Message: "ok", Data: data})
}

func writeErr(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, response{Code: status, Message: message})
}

// decodeJSON 解析请求体，限制大小。
func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return false
	}
	return true
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", s.handleHealth)

	// 认证
	mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("/api/v1/auth/callback", s.handleCallback)
	mux.HandleFunc("/api/v1/auth/me", s.handleMe)
	mux.HandleFunc("/api/v1/auth/logout", s.handleLogout)

	// 镜像读接口
	mux.HandleFunc("/api/v1/search", s.handleSearch)
	mux.HandleFunc("/api/v1/repos", s.handleRepos)
	mux.HandleFunc("/api/v1/repos/", s.handleRepoSub)

	// 中间件顺序：日志 → CORS → CSRF 来源校验 → 路由。
	// CSRF 校验必须在路由之前，确保所有写方法都被覆盖（含未来新增的写接口）。
	handler := withLogging(s.withCORS(auth.OriginGuard(s.cfg.AllowedOrigins())(mux)))
	return handler
}

// handleRepos 处理 /api/v1/repos 的 GET（列表）与 POST（创建）。
func (s *Server) handleRepos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listRepos(w, r)
	case http.MethodPost:
		s.auth.Middleware(http.HandlerFunc(s.createRepo)).ServeHTTP(w, r)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleRepoSub 分发 /api/v1/repos/{ns}/{repo}[/tags|/readme]
func (s *Server) handleRepoSub(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/repos/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	ns, repo := parts[0], parts[1]

	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.getRepo(w, r, ns, repo)
		case http.MethodPut:
			s.auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.updateRepo(w, r, ns, repo)
			})).ServeHTTP(w, r)
		case http.MethodDelete:
			s.auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.deleteRepo(w, r, ns, repo)
			})).ServeHTTP(w, r)
		default:
			writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if len(parts) == 3 && r.Method == http.MethodGet {
		switch parts[2] {
		case "tags":
			s.getRepoTags(w, r, ns, repo)
		case "readme":
			s.getRepoReadme(w, r, ns, repo)
		default:
			writeErr(w, http.StatusNotFound, "not found")
		}
		return
	}
	writeErr(w, http.StatusNotFound, "not found")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeOK(w, map[string]string{"status": "healthy"})
}

// ---- 中间件 ----

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// withCORS 处理跨源头。
//
// ⚠️ 关键：会话改用 httpOnly Cookie 后，**不能再用 `Access-Control-Allow-Origin: *`**——
// 带凭证的请求要求响应头必须是具体来源，且必须显式声明 Allow-Credentials，
// 通配符会被浏览器直接拒绝。
//
// 现在改为：请求的 Origin 在白名单内时，回显该 Origin 并允许凭证；
// 不在白名单内则不发 CORS 头（浏览器会拦截，服务端不受影响）。
// 生产环境由 Nginx 同源反代，届时根本不会产生跨源请求。
func (s *Server) withCORS(next http.Handler) http.Handler {
	allowed := s.cfg.AllowedOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && auth.OriginAllowed(origin, allowed) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
			// 响应随 Origin 变化，避免中间缓存把 A 站的响应喂给 B 站。
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
