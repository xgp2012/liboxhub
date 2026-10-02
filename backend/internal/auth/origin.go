package auth

import (
	"net/http"
	"net/url"
	"strings"
)

// OriginGuard 是 Cookie 会话方案下的 CSRF 防线。
//
// 背景：改用 httpOnly Cookie 承载会话后，浏览器会自动携带 Cookie，
// 因此「跨站发起的写请求」本身就能带上凭证——这正是 CSRF。
// 现有两道防线：
//
//  1. Cookie 的 SameSite=Lax —— 跨站 POST 不携带 Cookie（浏览器层）
//  2. 本中间件校验 Origin/Referer —— 必须属于白名单（应用层）
//
// 之所以两层都要，是因为 SameSite 的行为依赖浏览器版本与实现，
// 且对「同站不同源」（如子域）不设防；应用层校验不依赖浏览器实现。
//
// 判定规则（仅作用于非安全的写方法）：
//   - 取 Origin；缺失则回退 Referer
//   - 两者都缺失：放行（非浏览器客户端，如 curl/CLI/测试脚本，它们不携带
//     ambient 凭证，不存在 CSRF 风险；其凭证必须被显式写入请求头或参数）
//   - 存在则必须与白名单同源（scheme+host+port 完全一致）
func OriginGuard(allowed []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 安全方法（GET/HEAD/OPTIONS）不改变状态，无需校验。
			// OPTIONS 是 CORS 预检，必须放行，否则预检失败会让正常请求也挂掉。
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				origin = originFromReferer(r.Header.Get("Referer"))
			}
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// "null" 出现在沙箱 iframe、file:// 等场景，一律视为不可信。
			if origin == "null" || !OriginAllowed(origin, allowed) {
				writeJSONErr(w, http.StatusForbidden, "cross-site request blocked")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

func originFromReferer(ref string) string {
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// OriginAllowed 做精确同源比较（scheme + host + port），
// 避免用 strings.HasPrefix 导致 "http://localhost:3000.evil.com" 这类前缀绕过。
func OriginAllowed(origin string, allowed []string) bool {
	got, err := url.Parse(origin)
	if err != nil || got.Host == "" {
		return false
	}
	for _, a := range allowed {
		want, err := url.Parse(strings.TrimRight(strings.TrimSpace(a), "/"))
		if err != nil || want.Host == "" {
			continue
		}
		if strings.EqualFold(got.Scheme, want.Scheme) && strings.EqualFold(got.Host, want.Host) {
			return true
		}
	}
	return false
}
