package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/api"
	"github.com/joaoGMPereira/autocut/server/internal/config"
	"github.com/joaoGMPereira/autocut/server/internal/database"
)

func main() {
	host := flag.String("host", "127.0.0.1", "bind host")
	port := flag.Int("port", 4070, "listen port")
	dir := flag.String("dir", "", "data directory")
	flag.Parse()

	if *dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get home dir: %v", err)
		}
		*dir = fmt.Sprintf("%s/.autocut", home)
	}

	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	cfg, err := config.Load(*dir)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.Open(*dir)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	router := api.NewRouter(cfg, db)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // disabled for SSE streaming
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("autocut server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}

	log.Println("server stopped")
}
