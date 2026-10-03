package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/LiStudioorg/boxli/internal/config"
	"github.com/LiStudioorg/boxli/internal/db"
	"github.com/LiStudioorg/boxli/internal/seed"
)

func main() {
	configPath := flag.String("config", config.DefaultPath,
		"TOML 配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		os.Exit(1)
	}
	log.Printf("loaded %s", cfg)

	ctx := context.Background()
	pool, err := db.Pool(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	if err := seed.Run(ctx, pool); err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Println("seed done")
}
