package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"agent-control-plane/apps/api/internal/config"
	httpx "agent-control-plane/apps/api/internal/http"
	"agent-control-plane/apps/api/internal/repo"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer pool.Close()

	httpx.SetStore(repo.NewStore(pool))

	addr := ":8080"
	if p := os.Getenv("API_PORT"); p != "" {
		addr = ":" + p
	}

	r := httpx.NewRouter()
	log.Printf("api listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
