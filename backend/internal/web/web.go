// Package web 把前端产物内嵌进后端二进制，并托管 Nuxt SSR 子进程。
//
// 设计目标：**一个二进制完成部署**。部署方只需拷贝 boxli-hub 与 hub.toml，
// 不再需要单独同步 frontend/.output 目录、也不需要单独的 systemd unit。
//
// 组成：
//
//	dist/public  客户端静态资源（.js/.css/字体等）——由 Go 直接伺服，不经过 node
//	dist/ssr.mjs Nuxt SSR 服务端 bundle（单文件，已内联全部 npm 依赖，约 15MB）
//
// 为什么 SSR 还需要 node：
//
//	Go 无法执行 JavaScript（实测纯 Go 引擎 goja 会在打包产物的第一个正则字面量上 panic）。
//	因此 SSR 必须交给 node 执行。但 node 只是**运行时依赖**（生产已装），
//	前端产物本身完全内嵌在二进制里，无需任何外部文件。
//
// 启动流程：
//
//	1. 把内嵌的 ssr.mjs 释放到临时目录（默认 os.TempDir 下的随机目录）
//	2. 挑一个空闲回环端口，以子进程方式启动 node
//	3. 健康探测等待其就绪
//	4. 把非 /api、非静态资源的请求反向代理给它
//	5. 退出时转发信号并清理临时目录
package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ⚠️ 必须用 all: 前缀。Go 的 go:embed 默认**忽略**以 "." 或 "_" 开头的目录与文件，
// 而 Nuxt 的产物目录恰好是 _nuxt —— 实测漏掉 all: 会导致静态资源全部 404，
// 且编译期不报任何错误（静默陷阱）。
//
//go:embed all:dist
var assets embed.FS

// PublicPrefix 是客户端资源的 URL 前缀（Nuxt 默认 /_nuxt/）。
const PublicPrefix = "/_nuxt/"

// Assets 返回客户端静态资源的文件系统（根为 public/ 的内容）。
func Assets() (fs.FS, error) {
	return fs.Sub(assets, "dist/public")
}

// SSRBundle 返回内嵌的 SSR bundle 字节。
func SSRBundle() ([]byte, error) {
	return assets.ReadFile("dist/ssr.mjs")
}

// SSR 是一个受管的 Nuxt SSR 子进程。
type SSR struct {
	cmd     *exec.Cmd
	port    int
	tmpDir  string
	proxy   *httputil.ReverseProxy
	mu      sync.Mutex
	stopped bool
}

// StartSSR 释放内嵌 bundle 并启动 node 子进程，返回可直接使用的反向代理。
//
// nodePath 为空时从 PATH 查找 node。若找不到 node，返回错误（除非
// allowMissing 为 true，此时返回 nil, nil —— 调用方据此降级为「仅静态资源」模式）。
func StartSSR(ctx context.Context, nodePath string, allowMissing bool) (*SSR, error) {
	bundle, err := SSRBundle()
	if err != nil {
		return nil, fmt.Errorf("读取内嵌 SSR bundle: %w", err)
	}

	if nodePath == "" {
		nodePath = "node"
	}
	resolved, err := exec.LookPath(nodePath)
	if err != nil {
		if allowMissing {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"未找到 node 可执行文件（%s）：SSR 需要 node 运行时。\n"+
				"请安装 node（如 apt install nodejs），或用 --no-ssr 仅提供静态资源", nodePath)
	}

	// 释放 bundle 到临时目录。用 MkdirTemp 保证多实例/多次启动互不干扰，
	// 且权限为 0700，避免同机其他用户读到。
	tmpDir, err := os.MkdirTemp("", "boxli-ssr-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录: %w", err)
	}

	bundlePath := filepath.Join(tmpDir, "ssr.mjs")
	if err := os.WriteFile(bundlePath, bundle, 0o600); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("释放 SSR bundle: %w", err)
	}

	port, err := freePort()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("分配端口: %w", err)
	}

	cmd := exec.Command(resolved, bundlePath)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("NITRO_PORT=%d", port),
		"NITRO_HOST=127.0.0.1",
		"NODE_ENV=production",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// 独立进程组：便于退出时连同其子进程一起清理（见 Stop 里的负号 kill）。
	//
	// ⚠️ 仅靠 SIGTERM 处理是不够的：若 Go 进程被 SIGKILL（或崩溃）杀掉，
	// defer 不会执行，node 子进程会变成 PPID=1 的孤儿并一直占着端口和内存。
	// Linux 的 Pdeathsig 让内核在父进程消亡时自动给子进程发信号，
	// 覆盖这种「来不及清理」的情况。
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}

	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("启动 node 子进程: %w", err)
	}

	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("ssr proxy error (%s): %v", r.URL.Path, err)
		http.Error(w, "SSR 服务不可用", http.StatusBadGateway)
	}

	s := &SSR{cmd: cmd, port: port, tmpDir: tmpDir, proxy: proxy}
	log.Printf("SSR 子进程已启动 (pid=%d, port=%d)", cmd.Process.Pid, port)

	// 等待就绪：SSR 首次启动需加载 15MB bundle，给足超时。
	if err := s.waitReady(ctx, 30*time.Second); err != nil {
		s.Stop()
		return nil, err
	}
	return s, nil
}

// waitReady 轮询直到 SSR 端口可连接，或超时/子进程退出。
func (s *SSR) waitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 子进程若已退出，不必等满超时
		if s.cmd.ProcessState != nil && s.cmd.ProcessState.Exited() {
			return errors.New("SSR 子进程启动后立即退出，请检查 node 版本与上方日志")
		}

		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			log.Printf("SSR 已就绪 (port=%d)", s.port)
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("等待 SSR 就绪超时（%s），port=%d", timeout, s.port)
}

// ServeHTTP 把请求反向代理给 SSR 子进程。
func (s *SSR) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

// Stop 终止子进程并清理临时目录。可安全重复调用。
func (s *SSR) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	if s.cmd != nil && s.cmd.Process != nil {
		// 负号 = 终止整个进程组，覆盖 node 可能派生的子进程。
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)

		done := make(chan struct{})
		go func() {
			_, _ = s.cmd.Process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
		}
	}
	if s.tmpDir != "" {
		_ = os.RemoveAll(s.tmpDir)
	}
	log.Println("SSR 子进程已停止，临时文件已清理")
}

// freePort 让内核分配一个空闲的回环端口。
// 注意存在「拿到端口到 node 真正监听」之间的极短竞争窗口，
// 因此端口仅绑定回环，不对外暴露。
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// IsAssetPath 判断请求是否应由 Go 直接伺服（静态资源），而非转给 SSR。
func IsAssetPath(p string) bool {
	return strings.HasPrefix(p, PublicPrefix)
}
