package hub

import (
	"errors"
	"log"
	"net/http"

	"github.com/LiStudioorg/boxli/internal/localauth"
)

// handlePasswordLogin POST /api/v1/auth/password
//
// 本地用户名 + 密码登录，与 GitHub OAuth 并存。
// 成功后签发与其他登录方式**完全一致**的会话（httpOnly Cookie + JWT + sessions 表），
// 因此下游鉴权逻辑无需区分登录来源。
//
// 请求体：{"username": "...", "password": "..."}
func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.local == nil {
		writeErr(w, http.StatusServiceUnavailable, "本地认证未启用")
		return
	}
	if !s.auth.Enabled() {
		writeErr(w, http.StatusServiceUnavailable,
			"认证未配置：请在 hub.toml 设置 session.jwt_secret")
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	userID, err := s.local.Authenticate(r.Context(), body.Username, body.Password)
	if err != nil {
		if errors.Is(err, localauth.ErrInvalidCredentials) {
			// 统一错误信息，不区分「用户不存在」与「密码错误」，
			// 避免被用来枚举系统内已存在的用户名。
			// 记录日志时不回显密码。
			log.Printf("password login failed for %q", body.Username)
			writeErr(w, http.StatusUnauthorized, "用户名或密码不正确")
			return
		}
		log.Printf("password login error: %v", err)
		writeErr(w, http.StatusInternalServerError, "登录失败")
		return
	}

	s.issueSession(w, r.Context(), userID)
}
