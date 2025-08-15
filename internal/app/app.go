package app

import (
	"context"
	log "log/slog"
	"time"

	"github.com/kirill-shtrykov/minimon/internal/conf"
	"github.com/kirill-shtrykov/minimon/internal/monitor"
)

const defaultTick = 5

type Monitor struct {
	cfg *conf.Config
	svc *monitor.Service
}

func (m *Monitor) Run(ctx context.Context) error {
	ticker := time.NewTicker(defaultTick * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.InfoContext(ctx, "Monitoring stopped.")

			return nil
		case <-ticker.C:
			log.DebugContext(ctx, "Collecting metrics...")
			m.svc.CollectAndStore(ctx)
		}
	}
}

func NewMonitor(cfg *conf.Config, svc *monitor.Service) *Monitor {
	return &Monitor{cfg: cfg, svc: svc}
}
