package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LiStudioorg/boxli/internal/auth"
	"github.com/LiStudioorg/boxli/internal/config"
	"github.com/LiStudioorg/boxli/internal/db"
	"github.com/LiStudioorg/boxli/internal/hub"
	"github.com/LiStudioorg/boxli/internal/localauth"
	"github.com/LiStudioorg/boxli/internal/web"
)

// 构建信息，由 CI 通过 -ldflags 注入：
//
//	go build -ldflags "-X main.buildVersion=v1.0.0 -X main.buildCommit=abc1234"
//
// 本地 go build 时保持默认值。
var (
	buildVersion = "dev"
	buildCommit  = "unknown"
)

func main() {
	configPath := flag.String("config", config.DefaultPath,
		"TOML 配置文件路径")
	noSSR := flag.Bool("no-ssr", false,
		"不启动 SSR 前端（只提供内嵌静态资源，页面将无服务端渲染）")
	showVersion := flag.Bool("version", false,
		"打印版本信息后退出")
	flag.Parse()

	if *showVersion {
		fmt.Printf("boxli-hub %s (%s)\n", buildVersion, buildCommit)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "boxli-hub: %v\n", err)
		os.Exit(1)
	}
	log.Printf("boxli-hub %s (%s)", buildVersion, buildCommit)
	log.Printf("loaded %s", cfg)

	ctx := context.Background()

	pool, err := db.Pool(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	// 迁移在启动时自动执行（幂等，已执行的记录在 schema_migrations）。
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("db migrate: %v", err)
	}
	log.Println("database migrated")

	srv := hub.New(cfg, pool)

	// ---- 首次部署引导 ----
	// 若系统中还没有任何用户，生成一次性安装令牌并打印到**服务端日志**。
	// 引导接口 /api/v1/setup 要求携带该令牌，因此即使服务暴露在公网，
	// 也只有能看到服务器日志的人（即部署者）能完成初始化。
	setupToken, err := auth.NewSetupToken()
	if err != nil {
		log.Fatalf("生成安装令牌: %v", err)
	}
	userCount, err := localauth.NewStore(pool).UserCount(ctx)
	if err != nil {
		log.Fatalf("检查用户: %v", err)
	}
	if userCount == 0 {
		srv.EnableSetup(setupToken)
		log.Println("────────────────────────────────────────────────────────────")
		log.Println(" 首次部署引导已启用：系统中还没有任何用户")
		log.Printf(" 请在浏览器打开： http://<你的域名>/setup?token=%s", setupToken)
		log.Println(" 该令牌仅在本次启动且系统无用户时有效，初始化后立即作废。")
		log.Println("────────────────────────────────────────────────────────────")
	} else {
		log.Printf("已有 %d 个用户，跳过首次部署引导", userCount)
	}

	// SSR 前端：把内嵌的 bundle 释放到临时目录并托管 node 子进程。
	// 这一步让「一个二进制」成立——前端产物全部内嵌，无需外部文件。
	var ssr *web.SSR
	if !*noSSR {
		ssr, err = web.StartSSR(ctx, cfg.NodePath, cfg.AllowMissingNode)
		if err != nil {
			log.Fatalf("启动 SSR 前端失败: %v", err)
		}
		if ssr == nil {
			log.Println("提示：未找到 node，已跳过 SSR（仅静态资源模式）")
		}
	} else {
		log.Println("--no-ssr 已指定：跳过 SSR 启动")
	}
	if ssr != nil {
		defer ssr.Stop()
	}
	srv = srv.WithSSR(ssr)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("boxli hub listening on %s", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	// defer ssr.Stop() 会在此后执行，负责回收 node 子进程与临时目录。
	log.Println("boxli hub stopped")
}
