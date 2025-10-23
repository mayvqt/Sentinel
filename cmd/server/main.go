// Package main starts the Sentinel HTTP server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mayvqt/Sentinel/internal/auth"
	"github.com/mayvqt/Sentinel/internal/config"
	"github.com/mayvqt/Sentinel/internal/handlers"
	"github.com/mayvqt/Sentinel/internal/server"
	"github.com/mayvqt/Sentinel/internal/store"
)

func main() {
	cfg, _ := config.Load()
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	s := store.NewMemStore()
	a := auth.New(cfg)
	h := handlers.New(s, a)
	srv := server.New(":"+port, s, h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := srv.Start(ctx); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			cancel()
		}
	}()

	// Wait until context is canceled (e.g., via signal or server error).
	<-ctx.Done()
	_ = srv.Close()
	// Allow graceful shutdown time
	time.Sleep(100 * time.Millisecond)
}
