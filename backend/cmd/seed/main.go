package main

import (
	"context"
	"log"

	"github.com/LiStudioorg/boxli/internal/config"
	"github.com/LiStudioorg/boxli/internal/db"
	"github.com/LiStudioorg/boxli/internal/seed"
)

func main() {
	cfg := config.Load()

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
