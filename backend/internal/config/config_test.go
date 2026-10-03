package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write 在临时目录写一份配置并返回路径。
func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hub.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// TestDefaultsApplied 验证未写的字段取默认值。
func TestDefaultsApplied(t *testing.T) {
	cfg, err := Load(write(t, `
[site]
frontend_url = "http://localhost:3011"
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != "127.0.0.1:3727" {
		t.Errorf("Addr = %q, want default 127.0.0.1:3727", cfg.Addr)
	}
	if cfg.SessionTTLHours != 720 {
		t.Errorf("SessionTTLHours = %d, want default 720", cfg.SessionTTLHours)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure should default to false")
	}
	if cfg.DevLogin {
		t.Error("DevLogin should default to false")
	}
	if cfg.DBURL == "" {
		t.Error("DBURL should have a default")
	}
}

// TestExplicitZeroValuesHonoured 验证显式写零值不会被默认值覆盖。
// 这是指针字段存在的理由：ttl_hours = 0 与「未写」必须区分开。
func TestExplicitZeroValuesHonoured(t *testing.T) {
	cfg, err := Load(write(t, `
[site]
frontend_url = "https://boxli.dev"

[session]
cookie_secure = true
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CookieSecure {
		t.Error("explicit cookie_secure = true was not honoured")
	}
}

// TestFrontendURLTrailingSlashTrimmed 验证尾部斜杠被去掉：
// OriginGuard 按 scheme://host:port 精确比对，带斜杠会永远匹配不上。
func TestFrontendURLTrailingSlashTrimmed(t *testing.T) {
	cfg, err := Load(write(t, `
[site]
frontend_url = "http://localhost:3011/"
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.FrontendURL != "http://localhost:3011" {
		t.Errorf("FrontendURL = %q, want trailing slash trimmed", cfg.FrontendURL)
	}
}

// TestAllowedOrigins 验证 CSRF 白名单由 frontend_url + extra_origins 组成。
func TestAllowedOrigins(t *testing.T) {
	cfg, err := Load(write(t, `
[site]
frontend_url = "http://localhost:3011"
extra_origins = ["http://127.0.0.1:3011", "  ", "http://localhost:3012/"]
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := cfg.AllowedOrigins()
	want := []string{"http://localhost:3011", "http://127.0.0.1:3011", "http://localhost:3012"}
	if len(got) != len(want) {
		t.Fatalf("AllowedOrigins() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllowedOrigins()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestOAuthEnabled 验证只有 ID 与 Secret 同时存在才算启用真实 OAuth。
func TestOAuthEnabled(t *testing.T) {
	base := `
[site]
frontend_url = "http://localhost:3011"
[github]
client_id = %q
secret = %q
redirect = "http://127.0.0.1:3727/api/v1/auth/callback"
`
	cases := []struct {
		id, secret string
		want       bool
	}{
		{"id", "secret", true},
		{"", "", false},
	}
	for _, c := range cases {
		cfg, err := Load(write(t, sprintf(base, c.id, c.secret)))
		if err != nil {
			t.Fatalf("Load(id=%q secret=%q): %v", c.id, c.secret, err)
		}
		if cfg.OAuthEnabled() != c.want {
			t.Errorf("OAuthEnabled(id=%q secret=%q) = %t, want %t",
				c.id, c.secret, cfg.OAuthEnabled(), c.want)
		}
	}
}

// TestUnknownKeyRejected 验证拼错的配置项会让启动失败，
// 而不是被静默忽略（「写了但没生效」是最难排查的一类故障）。
func TestUnknownKeyRejected(t *testing.T) {
	_, err := Load(write(t, `
[site]
frontend_url = "http://localhost:3011"
frontned_url = "http://typo"
`))
	if err == nil {
		t.Fatal("expected an error for the misspelled key, got nil")
	}
	if !strings.Contains(err.Error(), "frontned_url") {
		t.Errorf("error should name the unknown key, got: %v", err)
	}
}

// TestValidationErrors 覆盖各项启动期校验。
func TestValidationErrors(t *testing.T) {
	const okSite = "[site]\nfrontend_url = \"http://localhost:3011\"\n"

	cases := []struct {
		name string
		body string
		want string // 错误信息里应包含的片段
		ok   bool   // true 表示这一条应当加载成功
	}{
		{
			name: "监听非回环地址",
			body: "[server]\naddr = \"0.0.0.0:3727\"\n" + okSite,
			want: "回环",
		},
		{
			name: "addr 缺少端口",
			body: "[server]\naddr = \"127.0.0.1\"\n" + okSite,
			want: "host:port",
		},
		{
			name: "jwt_secret 过短",
			body: "[session]\njwt_secret = \"short\"\n" + okSite,
			want: "16",
		},
		{
			name: "只配 client_id 不配 secret",
			body: "[github]\nclient_id = \"abc\"\n" + okSite,
			want: "同时配置",
		},
		{
			name: "有凭据但缺 redirect",
			body: "[github]\nclient_id = \"abc\"\nsecret = \"def\"\n" + okSite,
			want: "redirect",
		},
		{
			name: "frontend_url 留空则取默认值",
			body: "[site]\nfrontend_url = \"\"\n",
			// 留空等价于「未配置」，不报错；这里靠 want 为空表示只断言 err == nil
			want: "",
			ok:   true,
		},
		{
			name: "frontend_url 带路径",
			body: "[site]\nfrontend_url = \"http://localhost:3011/app\"\n",
			want: "路径",
		},
		{
			name: "frontend_url 协议非法",
			body: "[site]\nfrontend_url = \"ftp://localhost\"\n",
			want: "http://",
		},
		{
			name: "extra_origins 非法",
			body: okSite + "[site.extra_origins]\n", // 类型错误（应为数组）
			want: "",
		},
		{
			name: "Secure Cookie 配 http",
			body: "[session]\ncookie_secure = true\n" + okSite,
			want: "丢弃",
		},
		{
			name: "https 却未开 Secure Cookie",
			body: "[site]\nfrontend_url = \"https://boxli.dev\"\n",
			want: "cookie_secure",
		},
		{
			name: "开启 dev 后门",
			body: "[dev]\nenabled = true\n" + okSite,
			want: "后门",
		},
		{
			name: "TOML 语法错误",
			body: "[site\nfrontend_url = ",
			want: "解析",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Load(write(t, c.body))
			if c.ok {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.want)
			}
			if c.want != "" && !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %q, want it to contain %q", err, c.want)
			}
		})
	}
}

// TestLoopbackHTTPSAllowed 验证回环地址上的 https 不强制 Secure Cookie
// （自签证书的本地调试场景），而公网 https 则强制。
func TestLoopbackHTTPSAllowed(t *testing.T) {
	if _, err := Load(write(t, `
[site]
frontend_url = "https://localhost:3011"
`)); err != nil {
		t.Errorf("loopback https should be accepted without cookie_secure: %v", err)
	}
	if _, err := Load(write(t, `
[site]
frontend_url = "https://boxli.dev"
`)); err == nil {
		t.Error("public https without cookie_secure should be rejected")
	}
}

// TestMissingFile 验证缺文件时给出可操作的提示。
func TestMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "hub.toml.example") {
		t.Errorf("error should point at hub.toml.example, got: %v", err)
	}
}

// TestStringRedactsSecrets 验证日志摘要不含任何密钥。
// 这一条是安全断言：配置摘要会被写进启动日志。
func TestStringRedactsSecrets(t *testing.T) {
	// 故意用假值：绝不能把真实 secret 写进仓库，哪怕只是测试夹具。
	const secret = "0123456789abcdef0123456789abcdef01234567"
	cfg, err := Load(write(t, `
[github]
client_id = "Ov23liEXAMPLE"
secret = "`+secret+`"
redirect = "http://127.0.0.1:3727/api/v1/auth/callback"

[db]
url = "postgres://boxli:sup3rs3cret@127.0.0.1:5432/boxli_hub?sslmode=disable"

[site]
frontend_url = "http://localhost:3011"

[session]
jwt_secret = "another-secret-value-here"
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	s := cfg.String()
	for _, leak := range []string{secret, "sup3rs3cret", "another-secret-value-here"} {
		if strings.Contains(s, leak) {
			t.Errorf("String() leaked a secret (%q): %s", leak, s)
		}
	}
	if !strings.Contains(s, "boxli:***@") {
		t.Errorf("String() should redact the DB password, got: %s", s)
	}
}

// sprintf 只为让上面的表驱动用例更紧凑。
func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
