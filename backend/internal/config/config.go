package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr    string
	DBURL   string
	DataDir string

	// Auth (阶段 2 使用；阶段 1 仅读取)
	JWTSecret       string
	GitHubClientID  string
	GitHubSecret    string
	GitHubRedirect  string
	SessionTTLHours int

	// ---- 阶段 4：OAuth 回调落点与 token 交付方式 ----

	// FrontendURL 是回调成功后 302 的目标站点（如 http://localhost:3000）。
	// 前端 OAuth 必须经它同源代理发起，否则回调地址与 GitHub 登记值不符。
	FrontendURL string

	// CookieSecure 控制会话 Cookie 是否带 Secure 属性。
	// 本地 http://localhost 必须为 false，生产 HTTPS 下必须为 true。
	CookieSecure bool

	// ExtraOrigins 是除 FrontendURL 外额外允许的跨站来源（逗号分隔）。
	ExtraOrigins []string

	// DevLogin 显式开启「模拟登录」后门（POST /auth/login 传 github_user 直接发 token）。
	// 默认关闭：生产环境绝不能开启，否则任何人可冒充任意账号。
	DevLogin bool
}

func Load() Config {
	return Config{
		Addr:            getenv("BOXLI_ADDR", "127.0.0.1:3727"),
		DBURL:           getenv("BOXLI_DB", "postgres://boxli:boxli@127.0.0.1:5432/boxli_hub?sslmode=disable"),
		DataDir:         getenv("BOXLI_DATA_DIR", "./data"),
		JWTSecret:       getenv("BOXLI_JWT_SECRET", ""),
		GitHubClientID:  getenv("BOXLI_GITHUB_CLIENT_ID", ""),
		GitHubSecret:    getenv("BOXLI_GITHUB_SECRET", ""),
		GitHubRedirect:  getenv("BOXLI_GITHUB_REDIRECT", ""),
		SessionTTLHours: getenvInt("BOXLI_SESSION_TTL_HOURS", 720),
		FrontendURL:     strings.TrimRight(getenv("BOXLI_FRONTEND_URL", "http://localhost:3000"), "/"),
		CookieSecure:    getenvBool("BOXLI_COOKIE_SECURE", false),
		ExtraOrigins:    splitList(getenv("BOXLI_EXTRA_ORIGINS", "")),
		DevLogin:        getenvBool("BOXLI_DEV_LOGIN", false),
	}
}

// AllowedOrigins 返回写接口允许的跨站来源集合（CSRF 防护用）。
func (c Config) AllowedOrigins() []string {
	out := make([]string, 0, len(c.ExtraOrigins)+1)
	if c.FrontendURL != "" {
		out = append(out, c.FrontendURL)
	}
	return append(out, c.ExtraOrigins...)
}

func splitList(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
