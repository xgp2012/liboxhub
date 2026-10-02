package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginAllowed(t *testing.T) {
	allowed := []string{"http://localhost:3000", "https://boxli.dev"}

	cases := []struct {
		origin string
		want   bool
		why    string
	}{
		{"http://localhost:3000", true, "白名单精确匹配"},
		{"https://boxli.dev", true, "白名单精确匹配"},
		{"https://boxli.dev/", true, "尾部斜杠应被容忍"},
		{"http://localhost:3001", false, "端口不同即跨源"},
		{"https://boxli.dev.evil.com", false, "前缀绕过必须被拒"},
		{"http://localhost:3000.evil.com", false, "前缀绕过必须被拒"},
		{"https://evil.com", false, "非白名单"},
		{"http://boxli.dev", false, "scheme 不同即跨源（降级攻击）"},
		{"null", false, "沙箱/opaque origin 不可信"},
		{"", false, "空来源不可信"},
		{"not-a-url", false, "非法 URL"},
	}
	for _, c := range cases {
		if got := OriginAllowed(c.origin, allowed); got != c.want {
			t.Errorf("OriginAllowed(%q) = %v, want %v（%s）", c.origin, got, c.want, c.why)
		}
	}
}

// TestOriginGuard 验证 CSRF 中间件对写方法的判定。
func TestOriginGuard(t *testing.T) {
	guard := OriginGuard([]string{"http://localhost:3000"})
	reachable := false
	h := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reachable = true
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name    string
		method  string
		origin  string
		referer string
		want    int
	}{
		{"同源 POST 放行", http.MethodPost, "http://localhost:3000", "", http.StatusOK},
		{"跨站 POST 拦截", http.MethodPost, "https://evil.com", "", http.StatusForbidden},
		{"伪造前缀 POST 拦截", http.MethodPost, "http://localhost:3000.evil.com", "", http.StatusForbidden},
		{"null origin POST 拦截", http.MethodPost, "null", "", http.StatusForbidden},
		{"无 Origin 的非浏览器客户端放行", http.MethodPost, "", "", http.StatusOK},
		{"Origin 缺失时回退 Referer（同源）", http.MethodPost, "", "http://localhost:3000/submit", http.StatusOK},
		{"Origin 缺失时回退 Referer（跨站）", http.MethodPost, "", "https://evil.com/x", http.StatusForbidden},
		{"DELETE 跨站拦截", http.MethodDelete, "https://evil.com", "", http.StatusForbidden},
		{"GET 不校验（读方法不改状态）", http.MethodGet, "https://evil.com", "", http.StatusOK},
		{"OPTIONS 预检放行", http.MethodOptions, "https://evil.com", "", http.StatusOK},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reachable = false
			req := httptest.NewRequest(c.method, "/api/v1/repos", nil)
			if c.origin != "" {
				req.Header.Set("Origin", c.origin)
			}
			if c.referer != "" {
				req.Header.Set("Referer", c.referer)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Errorf("%s: status = %d, want %d", c.name, rec.Code, c.want)
			}
			// 被拦截时绝不能到达业务处理器。
			if c.want == http.StatusForbidden && reachable {
				t.Errorf("%s: 请求被拦截但仍到达了处理器", c.name)
			}
		})
	}
}

func TestSessionTokenFrom(t *testing.T) {
	t.Run("Cookie 优先于 Bearer", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "from-cookie"})
		r.Header.Set("Authorization", "Bearer from-header")
		if got := SessionTokenFrom(r); got != "from-cookie" {
			t.Errorf("got %q, want %q", got, "from-cookie")
		}
	})
	t.Run("无 Cookie 时回退 Bearer", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer from-header")
		if got := SessionTokenFrom(r); got != "from-header" {
			t.Errorf("got %q, want %q", got, "from-header")
		}
	})
	t.Run("都没有则为空", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if got := SessionTokenFrom(r); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestSetAndClearSessionCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	SetSessionCookie(rec, "tok", CookieOptions{Secure: true, MaxAge: 3600})

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if !c.HttpOnly {
		t.Error("会话 Cookie 必须是 HttpOnly（否则 XSS 可窃取 token）")
	}
	if !c.Secure {
		t.Error("Secure 选项未生效")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want /", c.Path)
	}

	// 登出必须能真正删除 Cookie：MaxAge < 0。
	rec2 := httptest.NewRecorder()
	ClearSessionCookie(rec2, CookieOptions{Secure: true})
	cleared := rec2.Result().Cookies()[0]
	if cleared.MaxAge >= 0 {
		t.Errorf("登出 Cookie MaxAge = %d, want < 0", cleared.MaxAge)
	}
	if cleared.Value != "" {
		t.Errorf("登出 Cookie 值应为空, got %q", cleared.Value)
	}
}
