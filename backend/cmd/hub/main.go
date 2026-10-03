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

	"github.com/LiStudioorg/boxli/internal/config"
	"github.com/LiStudioorg/boxli/internal/db"
	"github.com/LiStudioorg/boxli/internal/hub"
)

func main() {
	configPath := flag.String("config", config.DefaultPath,
		"TOML 配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "boxli-hub: %v\n", err)
		os.Exit(1)
	}
	log.Printf("loaded %s", cfg)

	pool, err := db.Pool(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(context.Background(), pool); err != nil {
		log.Fatalf("db migrate: %v", err)
	}
	log.Println("database migrated")

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           hub.New(cfg, pool).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("boxli hub listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("boxli hub stopped")
}
