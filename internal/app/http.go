package app

import (
	"context"
	"encoding/json"
	"fmt"
	log "log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kirill-shtrykov/minimon/internal/monitor"
)

const (
	defaultTimeout      = 5 * time.Second
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 15 * time.Second
	defaultIdleTimeout  = 60 * time.Second
	defaultMinTime      = -15 * time.Minute
)

type Reading struct {
	Value any       `json:"value"`
	Time  time.Time `json:"time"`
}

type Metric struct {
	Name     string    `json:"name"`
	Readings []Reading `json:"readings"`
}

type ResponseBody struct {
	Metrics []Metric `json:"metrics"`
}

func newResponseBody(readings []monitor.Reading) ([]byte, error) {
	metrics := make(map[string]any)

	for _, r := range readings {
		parts := strings.Split(r.Key, ".")
		current := metrics

		for i := 0; i < len(parts)-1; i++ {
			p := parts[i]
			if _, ok := current[p]; !ok {
				current[p] = make(map[string]any)
			}
			current = current[p].(map[string]any)
		}

		last := parts[len(parts)-1]
		_, err := strconv.Atoi(last)
		if err != nil {
			current[last] = map[string]any{"value": r.Value, "time": r.Date}
			continue
		}

		if _, ok := current[last]; !ok {
			current[last] = []any{}
		}

		arr := current[last].([]any)

		current[last] = append(arr, map[string]any{
			"value": r.Value,
			"time":  r.Date,
		})
	}

	out := map[string]any{"metrics": metrics}

	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return b, nil
}

func dateFromString(date string, def time.Time) time.Time {
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return def
	}

	return t
}

type Server struct {
	svc *monitor.Service
}

func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	log.DebugContext(r.Context(), "request", "method", r.Method, "URI", r.RequestURI)

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("OK")); err != nil {
		log.ErrorContext(r.Context(), "failed to write response body", log.Any("error", err))
	}
}

func (s *Server) apiHandler(w http.ResponseWriter, r *http.Request) {
	log.DebugContext(r.Context(), "request", "method", r.Method, "URI", r.RequestURI)

	if r.Method != http.MethodGet {
		log.WarnContext(r.Context(), "unknown method", "method", r.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)

		return
	}

	metric := r.PathValue("metric")
	minDate := dateFromString(r.URL.Query().Get("min"), time.Now().Add(defaultMinTime))
	maxDate := dateFromString(r.URL.Query().Get("max"), time.Now())

	readings, err := s.svc.Metric(r.Context(), metric, minDate, maxDate)
	if err != nil {
		log.ErrorContext(r.Context(), "failed to get readings", log.Any("error", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	b, err := newResponseBody(readings)
	if err != nil {
		log.ErrorContext(r.Context(), "failed to create response body", log.Any("error", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(b); err != nil {
		log.ErrorContext(r.Context(), "failed to write response body", log.Any("error", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) Run(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.rootHandler)
	mux.HandleFunc("/api/v1/metrics", s.apiHandler)
	mux.HandleFunc("/api/v1/metrics/{metric}", s.apiHandler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ReadHeaderTimeout: defaultTimeout,
	}

	go func() {
		<-ctx.Done()

		log.InfoContext(ctx, "HTTP server shutting down...")

		shCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
		defer cancel()

		if err := srv.Shutdown(shCtx); err != nil {
			log.ErrorContext(ctx, "failed to shutdown HTTP server", log.Any("error", err))
		}
	}()

	log.InfoContext(ctx, "start HTTP server", log.String("address", addr))

	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

func NewHTTPServer(svc *monitor.Service) *Server {
	return &Server{svc: svc}
}
