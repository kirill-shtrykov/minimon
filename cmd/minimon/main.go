package main

import (
	"context"
	log "log/slog"
	"os"
	"os/signal"

	// Register sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"

	"github.com/kirill-shtrykov/minimon/internal/app"
	"github.com/kirill-shtrykov/minimon/internal/conf"
	"github.com/kirill-shtrykov/minimon/internal/db"
	"github.com/kirill-shtrykov/minimon/internal/monitor"
	"github.com/kirill-shtrykov/minimon/pkg/flags"
)

func setupLogging(ctx context.Context, debug bool) {
	log.InfoContext(ctx, "MiniMon - lightweight monitoring utility")

	if debug {
		log.SetLogLoggerLevel(log.LevelDebug)
		log.DebugContext(ctx, "debug mode on")
	}
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	f := flags.Parse()
	setupLogging(ctx, f.Debug)

	cfg, err := conf.LoadConfig(f.Conf)
	if err != nil {
		log.ErrorContext(ctx, "Failed to read config", log.Any("error", err))

		return 1
	}

	const chans = 2

	errCh := make(chan error, chans)

	r, err := db.New(cfg.DB)
	if err != nil {
		log.ErrorContext(ctx, "database connection failed", log.Any("error", err))

		return 1
	}

	svc, err := monitor.New(r, cfg.Metrics)
	if err != nil {
		log.ErrorContext(ctx, "failed to create service", log.Any("error", err))

		return 1
	}

	srv := app.NewHTTPServer(svc)

	go func() {
		if err := srv.Run(ctx, f.Addr); err != nil {
			errCh <- err
		}
	}()

	mon := app.NewMonitor(cfg, svc)

	go func() {
		if err := mon.Run(ctx); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.InfoContext(ctx, "Shutting down MiniMon")
	case err := <-errCh:
		log.ErrorContext(ctx, "Application shutdown unexpectedly", log.Any("error", err))

		return 1
	}

	return 0
}

func main() {
	os.Exit(run())
}
