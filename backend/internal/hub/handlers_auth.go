package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LiStudioorg/boxli/internal/auth"
)

// errIdentityTaken 表示该 GitHub 账号的用户名/邮箱已被另一个账号占用。
var errIdentityTaken = errors.New("identity taken")

// safeRedirectPath 把前端传来的「登录后回跳地址」限制为站内相对路径，
// 防止开放重定向（open redirect）：只接受以单个 "/" 开头、非协议相对的路径。
//
// 例："/submit" ✓；"//evil.com" ✗；"https://evil.com" ✗；"/submit?a=1" ✓
func safeRedirectPath(p string) string {
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") {
		return ""
	}
	if strings.ContainsAny(p, "\\\r\n") {
		return ""
	}
	if u, err := url.Parse(p); err != nil || u.IsAbs() || u.Host != "" {
		return ""
	}
	return p
}

// frontendRedirect 拼出前端站点的绝对地址，用于 302（path 需为安全相对路径）。
func (s *Server) frontendRedirect(path string) string {
	base := s.cfg.FrontendURL
	if base == "" {
		base = "http://localhost:3000"
	}
	if p := safeRedirectPath(path); p != "" {
		return base + p
	}
	return base + "/dashboard"
}

// redirectWithError 带错误码 302 回前端登录页，由前端展示提示。
// 不回显 GitHub 返回的原始错误文本，避免把内部细节带到 URL 上。
func (s *Server) redirectWithError(w http.ResponseWriter, r *http.Request, code string) {
	if r.Method != http.MethodGet {
		http.Redirect(w, r, "/login?error="+url.QueryEscape(code), http.StatusFound)
		return
	}
	http.Redirect(w, r, s.frontendRedirect("/login?error="+url.QueryEscape(code)), http.StatusFound)
}

// handleLogin POST /api/v1/auth/login
// 有 OAuth 凭证时返回 GitHub 授权地址；否则（且开启 DevLogin 时）用 body 的 github_user 模拟登录。
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.auth.Enabled() {
		writeErr(w, http.StatusServiceUnavailable, "auth not configured: set BOXLI_JWT_SECRET")
		return
	}

	if s.oauth.Enabled() {
		// 记下用户从哪来，登录成功后原路送回（仅限站内相对路径）。
		var body struct {
			Redirect string `json:"redirect"`
		}
		// body 可选：前端可能只发空请求，解析失败不阻断登录。
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body)
		redirect := safeRedirectPath(body.Redirect)

		state, expiresAt, err := s.state.Create(r.Context(), redirect)
		if err != nil {
			log.Printf("create oauth state: %v", err)
			writeErr(w, http.StatusInternalServerError, "state generation failed")
			return
		}
		writeOK(w, map[string]interface{}{
			"authorize_url":    s.oauth.AuthorizeURL(state),
			"state":            state,
			"state_expires_at": expiresAt,
		})
		return
	}

	// dev 模拟登录：必须显式开启 BOXLI_DEV_LOGIN=1，默认关闭（生产禁止）。
	if !s.cfg.DevLogin {
		writeErr(w, http.StatusServiceUnavailable,
			"oauth not configured; dev login is disabled (set BOXLI_DEV_LOGIN=1 for local development)")
		return
	}

	// dev 模拟登录：{ "github_user": "alice" }
	var body struct {
		GitHubUser  string `json:"github_user"`
		GitHubID    int64  `json:"github_id"`
		AccessToken string `json:"access_token"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.GitHubUser == "" {
		writeErr(w, http.StatusBadRequest, "github_user required when OAuth is not configured")
		return
	}
	if body.GitHubID == 0 {
		body.GitHubID = -time.Now().UnixNano()
	}
	u, err := s.upsertUser(r.Context(), body.GitHubID, body.GitHubUser, "", "")
	if err != nil {
		if errors.Is(err, errIdentityTaken) {
			writeErr(w, http.StatusConflict, "username already taken by another account")
			return
		}
		log.Printf("login upsert: %v", err)
		writeErr(w, http.StatusInternalServerError, "upsert user failed")
		return
	}
	s.issueSession(w, r.Context(), u.ID)
}

// handleCallback GET /api/v1/auth/callback?code=...&state=...
//
// 先校验 state（一次性，防 CSRF / 授权码重放），再换取 token，
// 最后 302 回前端并**用 httpOnly Cookie 交付会话**。
//
// 注意：token 绝不放在重定向 URL 的 query 里——那会进入浏览器历史、
// Referer 头与服务器访问日志。
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.oauth.Enabled() {
		writeErr(w, http.StatusServiceUnavailable, "oauth not configured")
		return
	}

	ctx := r.Context()

	// state 校验必须最先做且总是消费：无论后续成功、被用户拒绝还是出错，
	// state 都是一次性的，避免留下可重放的授权凭证。
	state := r.URL.Query().Get("state")
	redirectPath, err := s.state.Consume(ctx, state)
	if err != nil {
		if errors.Is(err, auth.ErrBadState) {
			s.redirectWithError(w, r, "invalid_state")
			return
		}
		log.Printf("consume oauth state: %v", err)
		s.redirectWithError(w, r, "state_error")
		return
	}

	// 用户在 GitHub 上点了「取消」：GitHub 回调 error/error_description，无 code。
	if e := r.URL.Query().Get("error"); e != "" {
		log.Printf("oauth denied by user: %s %s", e, r.URL.Query().Get("error_description"))
		s.redirectWithError(w, r, "access_denied")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		s.redirectWithError(w, r, "missing_code")
		return
	}

	accessToken, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		log.Printf("oauth exchange: %v", err)
		s.redirectWithError(w, r, "exchange_failed")
		return
	}
	gh, err := s.oauth.FetchUser(ctx, accessToken)
	if err != nil {
		log.Printf("oauth fetch user: %v", err)
		s.redirectWithError(w, r, "github_failed")
		return
	}

	u, err := s.upsertUser(ctx, gh.ID, gh.Login, gh.Email, gh.AvatarURL)
	if err != nil {
		if errors.Is(err, errIdentityTaken) {
			log.Printf("oauth upsert user: %v", err)
			s.redirectWithError(w, r, "identity_taken")
			return
		}
		log.Printf("oauth upsert user: %v", err)
		s.redirectWithError(w, r, "user_failed")
		return
	}

	token, expiresAt, err := s.auth.Issue(ctx, u.ID)
	if err != nil {
		log.Printf("oauth issue token: %v", err)
		s.redirectWithError(w, r, "issue_failed")
		return
	}
	s.setSessionCookie(w, token, expiresAt)

	// 登录成功：回跳来源页，无则进用户中心。
	http.Redirect(w, r, s.frontendRedirect(redirectPath), http.StatusFound)
}

// handleMe GET /api/v1/auth/me
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, auth.UserFrom(r.Context()))
	})).ServeHTTP(w, r)
}

// handleLogout POST /api/v1/auth/logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sid := auth.SessionIDFrom(r.Context()); sid != "" {
			if err := s.auth.Revoke(r.Context(), sid); err != nil {
				writeErr(w, http.StatusInternalServerError, "logout failed")
				return
			}
		}
		// 无论走 Cookie 还是 Bearer，都清一次 Cookie，保证浏览器端彻底登出。
		auth.ClearSessionCookie(w, auth.CookieOptions{Secure: s.cfg.CookieSecure})
		writeOK(w, map[string]string{"status": "logged out"})
	})).ServeHTTP(w, r)
}

// issueSession 签发会话并下发 httpOnly Cookie（浏览器主路径）。
// 同时返回 token 字段，便于 curl/CLI 联调使用 Bearer。
func (s *Server) issueSession(w http.ResponseWriter, ctx context.Context, userID int64) {
	token, expiresAt, err := s.auth.Issue(ctx, userID)
	if err != nil {
		if errors.Is(err, auth.ErrNoSecret) {
			writeErr(w, http.StatusServiceUnavailable, "auth not configured")
			return
		}
		log.Printf("issue token: %v", err)
		writeErr(w, http.StatusInternalServerError, "issue token failed")
		return
	}
	s.setSessionCookie(w, token, expiresAt)
	writeOK(w, map[string]interface{}{
		"token":      token,
		"expires_at": expiresAt,
		"token_type": "Bearer",
	})
}

// setSessionCookie 按配置下发会话 Cookie，TTL 与会话过期时间对齐。
func (s *Server) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge <= 0 {
		maxAge = s.cfg.SessionTTLHours * 3600
	}
	auth.SetSessionCookie(w, token, auth.CookieOptions{
		Secure: s.cfg.CookieSecure,
		MaxAge: maxAge,
	})
}

// upsertUser 按 github_id 查/建用户并同步资料，返回用户。
//
// 注意：唯一冲突（username 或 email 已被「另一个 github_id」占用）时**不能**退化为
// 按 username 查找——那会把当前 GitHub 账号错误地登录成他人账号（账号接管）。
// 此处改为返回明确错误，由调用方以 409 告知用户改用户名/邮箱后重试。
func (s *Server) upsertUser(ctx context.Context, githubID int64, username, email, avatar string) (*auth.User, error) {
	var (
		id    int64
		final string
	)
	err := s.store.pool.QueryRow(ctx, `
		INSERT INTO users (username, email, avatar_url, github_id)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4)
		ON CONFLICT (github_id) DO UPDATE
		SET username = EXCLUDED.username,
		    avatar_url = COALESCE(EXCLUDED.avatar_url, users.avatar_url),
		    email = COALESCE(EXCLUDED.email, users.email),
		    updated_at = NOW()
		RETURNING id, username`,
		username, email, avatar, githubID).Scan(&id, &final)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: username %q or email already taken by another account",
				errIdentityTaken, username)
		}
		return nil, err
	}
	return &auth.User{ID: id, Username: final, Email: email, AvatarURL: avatar}, nil
}
