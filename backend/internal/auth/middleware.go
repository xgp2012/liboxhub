package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey int

const (
	userKey ctxKey = 1
	sidKey  ctxKey = 2
)

// Middleware 校验会话（httpOnly Cookie 优先，其次 Authorization: Bearer），
// 成功则把 *User 与 session id 注入 context。
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := SessionTokenFrom(r)
		if token == "" {
			writeJSONErr(w, http.StatusUnauthorized, "missing session cookie or bearer token")
			return
		}
		u, sid, err := s.Verify(r.Context(), token)
		if err != nil {
			writeJSONErr(w, http.StatusUnauthorized, err.Error())
			return
		}
		ctx := context.WithValue(r.Context(), userKey, u)
		ctx = context.WithValue(ctx, sidKey, sid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFrom 从 context 取出已认证用户，未认证返回 nil。
func UserFrom(ctx context.Context) *User {
	u, _ := ctx.Value(userKey).(*User)
	return u
}

// SessionIDFrom 从 context 取出当前会话 id。
func SessionIDFrom(ctx context.Context) string {
	sid, _ := ctx.Value(sidKey).(string)
	return sid
}

// BearerToken 从请求头提取 Bearer token。
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const p = "Bearer "
	if len(h) > len(p) && strings.EqualFold(h[:len(p)], p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}

func writeJSONErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(`{"code":` + itoa(status) + `,"message":"` + msg + `","data":null}`))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
