package hub

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// newSetupTestServer 构造一个仅用于引导相关测试的 Server。
//
// 这些测试**不连接数据库**：handleSetupPage 与令牌校验都不触碰 DB，
// 因此可以在无库环境下运行。真正写库的 CreateFirstAdmin 由集成测试覆盖。
func newSetupTestServer(token string) *Server {
	s := &Server{}
	return s.EnableSetup(token)
}

// 引导页必须是自包含 HTML：除了表单与内联脚本，不依赖任何外部资源。
// 原因：引导发生在「前端可能尚未正确配置」的时刻，不能依赖 SPA 产物。
func TestSetupPageIsSelfContained(t *testing.T) {
	s := newSetupTestServer("token-abc")

	req := httptest.NewRequest(http.MethodGet, "/setup?token=token-abc", nil)
	rec := httptest.NewRecorder()
	s.handleSetupPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"<form", "username", "password", "/api/v1/setup",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("引导页缺少 %q", want)
		}
	}

	// 不应引用外部资源（否则引导可能因前端未就绪而失败）
	for _, bad := range []string{"<link rel=\"stylesheet\"", "<script src="} {
		if strings.Contains(body, bad) {
			t.Errorf("引导页不应引用外部资源，发现 %q", bad)
		}
	}

	// 不应缓存引导页
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Errorf("Cache-Control = %q，期望包含 no-store", cc)
	}
}

func TestSetupPageRejectsNonGET(t *testing.T) {
	s := newSetupTestServer("token-abc")
	req := httptest.NewRequest(http.MethodPost, "/setup", nil)
	rec := httptest.NewRecorder()
	s.handleSetupPage(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /setup 状态码 = %d，期望 405", rec.Code)
	}
}

// 令牌校验是引导接口的核心防线。
func TestSetupTokenValidation(t *testing.T) {
	const good = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	cases := []struct {
		name  string
		token string // 服务端持有的令牌
		got   string // 请求携带的令牌
		want  bool
	}{
		{"正确令牌", good, good, true},
		{"错误令牌", good, "wrong", false},
		{"空令牌请求", good, "", false},
		{"令牌已作废（服务端为空）", "", good, false},
		{"前缀匹配不算通过", good, good[:32], false},
		{"后缀匹配不算通过", good, good[32:], false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newSetupTestServer(c.token)
			if got := s.setupTokenOK(c.got); got != c.want {
				t.Errorf("setupTokenOK(%q) = %v，期望 %v", c.got, got, c.want)
			}
		})
	}
}

// 初始化完成后必须作废令牌，避免被重放。
func TestInvalidateSetupToken(t *testing.T) {
	const good = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	s := newSetupTestServer(good)

	if !s.SetupEnabled() {
		t.Fatal("设置令牌后 SetupEnabled 应为 true")
	}
	if !s.setupTokenOK(good) {
		t.Fatal("正确令牌应通过")
	}

	s.invalidateSetupToken()

	if s.SetupEnabled() {
		t.Error("作废后 SetupEnabled 应为 false")
	}
	if s.setupTokenOK(good) {
		t.Error("作废后原令牌不应再通过")
	}
}

// 未启用引导时，接口必须拒绝，不能因为「没有令牌」而放行。
func TestSetupRejectsWhenTokenEmpty(t *testing.T) {
	s := &Server{} // 未调用 EnableSetup
	if s.SetupEnabled() {
		t.Error("未启用引导时 SetupEnabled 应为 false")
	}
	if s.setupTokenOK("anything") {
		t.Error("未启用引导时任何令牌都不应通过")
	}
}

// handleSetup 必须拒绝非 POST。
func TestSetupEndpointRejectsGET(t *testing.T) {
	s := newSetupTestServer("tok")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup", nil)
	rec := httptest.NewRecorder()
	s.handleSetup(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/v1/setup 状态码 = %d，期望 405", rec.Code)
	}
}

// 确保 jwt 依赖被显式使用（避免 import 被误删导致签名校验逻辑失去保障）。
var _ = jwt.SigningMethodHS256
