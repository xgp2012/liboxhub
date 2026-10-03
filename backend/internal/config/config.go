// Package config 从 TOML 配置文件加载 Boxli Hub 的全部配置。
//
// 设计约定：
//   - 配置文件是**唯一的**配置来源，本包不读取任何环境变量。
//   - 未在文件中出现的字段取默认值（见 defaults）；出现但非法的值直接报错，
//     不做静默降级（例如拼错的键会因 DisallowUnknownFields 等价行为被拒绝）。
//   - 不打印任何 secret（GitHubSecret）；DebugString 只输出脱敏摘要。
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// DefaultPath 是未显式指定 --config 时查找的文件名。
const DefaultPath = "hub.toml"

// ---------------------------------------------------------------------------
// TOML 文件结构
// ---------------------------------------------------------------------------

// File 与 hub.toml 一一对应。指针字段用于区分「未写」与「显式写了零值」：
// 未写取默认值，写了就用写的那个（哪怕它是 false / 0）。
type File struct {
	Server  ServerSection  `toml:"server"`
	DB      DBSection      `toml:"db"`
	Session SessionSection `toml:"session"`
	GitHub  GitHubSection  `toml:"github"`
	Site    SiteSection    `toml:"site"`
	Dev     DevSection     `toml:"dev"`
}

type ServerSection struct {
	// Addr 是监听地址，形如 127.0.0.1:3727。默认只监听回环，
	// 生产由 Nginx 同源反代；直接监听 0.0.0.0 会绕开 HTTPS 与 CSRF 保护。
	Addr string `toml:"addr"`
}

type DBSection struct {
	URL string `toml:"url"`
}

type SessionSection struct {
	// JWTSecret 是会话签名密钥，必填，至少 16 字符。
	JWTSecret string `toml:"jwt_secret"`
	// TTLHours 是会话有效期（小时），默认 720 = 30 天。
	TTLHours *int `toml:"ttl_hours"`
	// CookieSecure 控制会话 Cookie 是否带 Secure 属性：
	// 生产 HTTPS 必须 true，本地 http:// 必须 false（否则浏览器丢弃 Cookie）。
	CookieSecure *bool `toml:"cookie_secure"`
}

type GitHubSection struct {
	// ClientID / Secret 同时配置才启用真实 GitHub OAuth；任一为空则：
	// dev.enabled=true 时降级为模拟登录，否则认证接口返回 503。
	ClientID string `toml:"client_id"`
	Secret   string `toml:"secret"`
	// Redirect 必须与 GitHub 上登记的 Authorization callback URL 完全一致，
	// 否则换 token 时返回 redirect_uri_mismatch。
	Redirect string `toml:"redirect"`
}

type SiteSection struct {
	// FrontendURL 身兼两职：OAuth 成功后 302 的回跳落点，以及
	// 写接口 CSRF 校验的来源白名单。必须是前端实际访问的站点地址。
	FrontendURL string `toml:"frontend_url"`
	// ExtraOrigins 是额外允许的跨站来源，仅多域名/CI 场景需要。
	ExtraOrigins []string `toml:"extra_origins"`
}

type DevSection struct {
	// Enabled 开启「模拟登录」后门：POST /auth/login 传 github_user 即可登录为
	// 任意账号。生产环境必须保持 false——设为 true 时 normalize 会直接拒绝启动。
	Enabled bool `toml:"enabled"`
}

// ---------------------------------------------------------------------------
// 运行时配置
// ---------------------------------------------------------------------------

type Config struct {
	Addr    string
	DBURL   string
	DataDir string

	JWTSecret       string
	SessionTTLHours int

	GitHubClientID string
	GitHubSecret   string
	GitHubRedirect string

	FrontendURL  string
	CookieSecure bool
	ExtraOrigins []string

	DevLogin bool

	// Path 是本次加载的配置文件绝对路径，用于日志与报错定位。
	Path string
}

// AllowedOrigins 返回写接口允许的跨站来源集合（CSRF 防护用）。
func (c Config) AllowedOrigins() []string {
	out := make([]string, 0, len(c.ExtraOrigins)+1)
	if c.FrontendURL != "" {
		out = append(out, c.FrontendURL)
	}
	return append(out, c.ExtraOrigins...)
}

// OAuthEnabled 表示是否配置了完整的 GitHub OAuth 凭据。
func (c Config) OAuthEnabled() bool {
	return c.GitHubClientID != "" && c.GitHubSecret != ""
}

// String 输出不含任何密钥的配置摘要，可安全写入日志。
func (c Config) String() string {
	return fmt.Sprintf(
		"config{path=%s addr=%s db=%s frontend=%s cookie_secure=%t ttl=%dh oauth=%t dev_login=%t}",
		c.Path, c.Addr, redactDBURL(c.DBURL), c.FrontendURL, c.CookieSecure,
		c.SessionTTLHours, c.OAuthEnabled(), c.DevLogin,
	)
}

// ---------------------------------------------------------------------------
// 加载
// ---------------------------------------------------------------------------

// defaults 返回全部字段带默认值的配置。
func defaults() Config {
	return Config{
		Addr:            "127.0.0.1:3727",
		DBURL:           "postgres://boxli:boxli@127.0.0.1:5432/boxli_hub?sslmode=disable",
		DataDir:         "./data",
		SessionTTLHours: 720,
		FrontendURL:     "http://localhost:3011",
	}
}

// Load 读取 path 指向的 TOML 配置文件并校验。
// path 为空时使用 DefaultPath。
func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	cfg := defaults()
	cfg.Path = abs

	var f File
	meta, err := toml.DecodeFile(abs, &f)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf(
				"配置文件不存在：%s\n请复制 hub.toml.example 为 hub.toml 后按需修改", abs)
		}
		return Config{}, fmt.Errorf("解析配置文件 %s 失败：%w", abs, err)
	}

	// 拒绝拼错的键，避免「写了但没生效」这类静默故障。
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, 0, len(undecoded))
		for _, k := range undecoded {
			keys = append(keys, k.String())
		}
		return Config{}, fmt.Errorf(
			"配置文件 %s 含无法识别的配置项：%s（请检查拼写）",
			abs, strings.Join(keys, ", "))
	}

	cfg.apply(f)

	if err := cfg.normalize(); err != nil {
		return Config{}, fmt.Errorf("配置文件 %s 校验失败：%w", abs, err)
	}
	return cfg, nil
}

// apply 把文件内容覆盖到默认值之上。
// 规则：非零值直接覆盖；指针字段只要非 nil 就覆盖（这样才能表达「显式写了 false/0」）。
func (c *Config) apply(f File) {
	if v := strings.TrimSpace(f.Server.Addr); v != "" {
		c.Addr = v
	}
	if v := strings.TrimSpace(f.DB.URL); v != "" {
		c.DBURL = v
	}

	if v := strings.TrimSpace(f.Session.JWTSecret); v != "" {
		c.JWTSecret = v
	}
	if f.Session.TTLHours != nil {
		c.SessionTTLHours = *f.Session.TTLHours
	}
	if f.Session.CookieSecure != nil {
		c.CookieSecure = *f.Session.CookieSecure
	}

	c.GitHubClientID = strings.TrimSpace(f.GitHub.ClientID)
	c.GitHubSecret = strings.TrimSpace(f.GitHub.Secret)
	c.GitHubRedirect = strings.TrimSpace(f.GitHub.Redirect)

	if v := strings.TrimSpace(f.Site.FrontendURL); v != "" {
		c.FrontendURL = v
	}
	c.ExtraOrigins = f.Site.ExtraOrigins

	c.DevLogin = f.Dev.Enabled
}

// normalize 做 trim、补默认值与合法性校验。
func (c *Config) normalize() error {
	c.Addr = strings.TrimSpace(c.Addr)
	c.DBURL = strings.TrimSpace(c.DBURL)
	c.JWTSecret = strings.TrimSpace(c.JWTSecret)
	c.GitHubClientID = strings.TrimSpace(c.GitHubClientID)
	c.GitHubSecret = strings.TrimSpace(c.GitHubSecret)
	c.GitHubRedirect = strings.TrimSpace(c.GitHubRedirect)

	if c.Addr == "" {
		c.Addr = defaults().Addr
	}
	if c.DBURL == "" {
		c.DBURL = defaults().DBURL
	}
	if c.SessionTTLHours <= 0 {
		c.SessionTTLHours = defaults().SessionTTLHours
	}
	if c.DataDir == "" {
		c.DataDir = defaults().DataDir
	}

	// ---- server.addr ----
	host, _, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return fmt.Errorf("server.addr 必须是 host:port 形式，当前为 %q", c.Addr)
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return fmt.Errorf(
			"server.addr 不允许监听 %q：后端只应监听回环地址，"+
				"对外访问请由 Nginx 反向代理（另见 docs 部署说明）", host)
	}

	// ---- session.jwt_secret ----
	// 留空是允许的（未配置认证），但一旦要走登录就必须够长。
	if c.JWTSecret != "" && len(c.JWTSecret) < 16 {
		return errors.New("session.jwt_secret 至少需要 16 个字符（建议 openssl rand -hex 32 生成）")
	}

	// ---- github ----
	if (c.GitHubClientID == "") != (c.GitHubSecret == "") {
		return errors.New(
			"github.client_id 与 github.secret 必须同时配置或同时留空：" +
				"只配其中一个会在登录时静默失败")
	}
	if c.OAuthEnabled() && c.GitHubRedirect == "" {
		return errors.New("配置了 GitHub OAuth 凭据后，github.redirect 必填且须与 GitHub 上登记的回调地址完全一致")
	}

	// ---- site.frontend_url ----
	c.FrontendURL = strings.TrimRight(strings.TrimSpace(c.FrontendURL), "/")
	if c.FrontendURL == "" {
		return errors.New("site.frontend_url 不能为空：它同时是 OAuth 回跳落点与写接口 CSRF 来源白名单")
	}
	if err := validateOrigin("site.frontend_url", c.FrontendURL); err != nil {
		return err
	}

	// ---- site.extra_origins ----
	origins := make([]string, 0, len(c.ExtraOrigins))
	for _, o := range c.ExtraOrigins {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o == "" {
			continue
		}
		if err := validateOrigin("site.extra_origins", o); err != nil {
			return err
		}
		origins = append(origins, o)
	}
	c.ExtraOrigins = origins

	// ---- 互斥检查 ----
	// CookieSecure 与协议不匹配是「登录看似成功却始终未登录」的经典成因，
	// 在启动期拦下比线上排查便宜得多。
	if c.CookieSecure && strings.HasPrefix(c.FrontendURL, "http://") {
		return fmt.Errorf(
			"session.cookie_secure=true 但 site.frontend_url 是 http://（%s）："+
				"浏览器会直接丢弃 Secure Cookie，表现为登录后仍显示未登录。"+
				"本地开发请改为 false，生产 HTTPS 保持 true", c.FrontendURL)
	}
	if !c.CookieSecure && strings.HasPrefix(c.FrontendURL, "https://") && !isLoopback(c.FrontendURL) {
		return fmt.Errorf(
			"site.frontend_url 是 https://（%s）时 session.cookie_secure 应为 true，"+
				"否则会话 Cookie 会在明文信道传输", c.FrontendURL)
	}
	if c.DevLogin {
		return errors.New(
			"dev.enabled=true 会开启模拟登录后门（可冒充任意账号）。" +
				"请仅在本地调试时临时开启，确认无误后改回 false")
	}
	return nil
}

// validateOrigin 校验白名单来源必须是「scheme://host[:port]」，
// 不能带路径/查询，因为 OriginGuard 是按 scheme+host+port 精确比对。
func validateOrigin(field, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s 不是合法 URL：%q", field, raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s 必须以 http:// 或 https:// 开头，当前为 %q", field, raw)
	}
	if u.Host == "" {
		return fmt.Errorf("%s 缺少主机名：%q", field, raw)
	}
	if p := strings.Trim(u.Path, "/"); p != "" {
		return fmt.Errorf("%s 不能包含路径（%q）：它只用于比对 Origin 的 scheme://host:port", field, raw)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%s 不能包含查询串或片段：%q", field, raw)
	}
	return nil
}

func isLoopback(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// redactDBURL 隐去连接串里的密码，供日志使用。
// postgres://user:secret@host/db → postgres://user:***@host/db
func redactDBURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	if _, hasPassword := u.User.Password(); !hasPassword {
		return raw
	}
	return strings.Replace(raw, u.User.String(), u.User.Username()+":***", 1)
}
