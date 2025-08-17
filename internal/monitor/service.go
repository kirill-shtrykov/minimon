package monitor

import (
	"context"
	"fmt"
	log "log/slog"
	"strings"
	"time"

	"github.com/kirill-shtrykov/minimon/internal/conf"
	"github.com/kirill-shtrykov/minimon/internal/db"
	"github.com/kirill-shtrykov/minimon/internal/db/generated"
)

type Reading struct {
	Key   string
	Value any
	Date  time.Time
}

func fromBytes(data []byte, to string) (any, error) {
	switch to {
	case "string":
		return string(data), nil
	case "int":
		return BytesToInt(data)
	case "float":
		return BytesToFloat64(data)
	}

	return nil, fmt.Errorf("%w: %s", ErrUnknownValueType, to)
}

type Metric struct {
	Key         string
	Type        string
	LastValue   [][]byte
	LastCheck   time.Time
	Interval    time.Duration
	HandlerFunc func(metric *Metric) error
}

func (m *Metric) Handler() error {
	if err := m.HandlerFunc(m); err != nil {
		return fmt.Errorf("failed to make check: %w", err)
	}

	m.LastCheck = time.Now()

	return nil
}

type Service struct {
	repo    *db.Repo
	metrics []Metric
}

func (s *Service) CollectAndStore(ctx context.Context) {
	for _, m := range s.metrics {
		if time.Since(m.LastCheck) >= m.Interval {
			log.DebugContext(ctx, "collect", log.String("key", m.Key))

			if err := s.collectMetric(ctx, m); err != nil {
				log.ErrorContext(
					ctx,
					"failed to collect metric",
					log.String("metric", m.Key),
					log.String("type", m.Type),
					log.Any("error", err),
				)
			}
		}
	}
}

func (s *Service) collectMetric(ctx context.Context, metric Metric) error {
	err := metric.Handler()
	if err != nil {
		return fmt.Errorf("failed to run handler: %w", err)
	}

	if len(metric.LastValue) == 1 {
		if err := s.store(ctx, metric.Key, metric.LastValue[0], metric.Type); err != nil {
			return fmt.Errorf("failed to store metric: %w", err)
		}
	} else {
		for i, v := range metric.LastValue {
			if err := s.store(ctx, fmt.Sprintf("%s.%d", metric.Key, i), v, metric.Type); err != nil {
				return fmt.Errorf("failed to store metric: %w", err)
			}
		}
	}

	return nil
}

func (s *Service) store(ctx context.Context, key string, value []byte, t string) error {
	err := s.repo.AddValue(ctx, generated.AddValueParams{Key: key, Type: t, Value: value})
	if err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}

	return nil
}

func CollectInternal(m *Metric) error {
	switch k := strings.Split(m.Key, "."); k[0] {
	case "cpu":
		v, err := CPUByKey(m.Key)
		if err != nil {
			return err
		}

		m.LastValue = v
	default:
		return fmt.Errorf("%w: %s", ErrUnknownKeyError, m.Key)
	}

	return nil
}

func (s *Service) Metric(ctx context.Context, key string, minDate time.Time, maxDate time.Time) ([]Reading, error) {
	var metrics []generated.Metric
	var err error

	if key == "" {
		metrics, err = s.repo.MetricsByDate(ctx, minDate, maxDate)
	} else {
		metrics, err = s.repo.Metric(ctx, key, minDate, maxDate)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get metric: %w", err)
	}

	readings := make([]Reading, len(metrics))

	for i, m := range metrics {
		value, err := fromBytes(m.Value, m.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get metric: %w", err)
		}

		readings[i] = Reading{Key: m.Key, Value: value, Date: m.Date}
	}

	return readings, nil
}

func New(repo *db.Repo, cfg []conf.Metric) (*Service, error) {
	metrics := make([]Metric, len(cfg))

	for i, m := range cfg {
		var h func(metric *Metric) error

		switch m.Method {
		case "internal":
			h = CollectInternal
		default:
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedMetricMethodError, m.Method)
		}

		metrics[i] = Metric{
			Key:         m.Key,
			Type:        m.Type,
			LastValue:   nil,
			LastCheck:   time.Time{},
			Interval:    time.Duration(m.Interval) * time.Second,
			HandlerFunc: h,
		}
	}

	return &Service{repo: repo, metrics: metrics}, nil
}
