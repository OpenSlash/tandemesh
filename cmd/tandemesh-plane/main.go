// Command tandemesh-plane runs the Orca control plane.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/OpenSlash/tandemesh/internal/app"
	"github.com/OpenSlash/tandemesh/internal/config"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	application, err := app.New(ctx, cfg, log)
	if err != nil {
		log.Error("start", "err", err)
		os.Exit(1)
	}
	defer application.Close()

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           application.Handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("orca plane listening", "addr", cfg.Listen, "db", cfg.DBPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("orca plane stopped")
}
