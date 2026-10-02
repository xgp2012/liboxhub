package auth

import "net/http"

// 会话 Cookie 名。前端不读取它（httpOnly），只靠它自动随请求发送。
const SessionCookieName = "boxli_session"

// CookieOptions 控制会话 Cookie 的属性。
type CookieOptions struct {
	// Secure 在生产 HTTPS 下必须为 true；本地 http://localhost 必须为 false，
	// 否则浏览器会直接丢弃该 Cookie，表现为「登录后依旧未登录」。
	Secure bool
	// MaxAge 秒；<=0 表示会话级 Cookie（浏览器关闭即失效）。
	MaxAge int
}

// SetSessionCookie 下发 httpOnly 会话 Cookie，承载 JWT。
//
// 关键安全属性：
//   - HttpOnly：JS 读不到，XSS 无法窃取 token（这是选 Cookie 方案的核心原因）
//   - SameSite=Lax：跨站 POST 不带 Cookie，配合 Origin 校验构成 CSRF 双保险；
//     用 Lax 而非 Strict，是为了让从 GitHub 跳回本站的顶层导航能带上 Cookie
//   - Path=/：前端与 /api 同源代理下，全站请求均可见
func SetSessionCookie(w http.ResponseWriter, token string, opts CookieOptions) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   opts.MaxAge,
	})
}

// ClearSessionCookie 删除会话 Cookie（登出）。
// 必须与下发时同 Path/Secure/SameSite，否则浏览器不会认定为同一个 Cookie。
func ClearSessionCookie(w http.ResponseWriter, opts CookieOptions) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// SessionTokenFrom 按「Cookie 优先，其次 Authorization: Bearer」提取会话 token。
//
// 两个来源都支持是有意为之：
//   - 浏览器走 Cookie（httpOnly，防 XSS）
//   - curl / CLI / 测试脚本走 Bearer，无需处理 Cookie
func SessionTokenFrom(r *http.Request) string {
	if c, err := r.Cookie(SessionCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	return BearerToken(r)
}
