package app

import (
	"context"
	"encoding/json"
	"fmt"
	log "log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kirill-shtrykov/minimon/internal/conf"
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

func uPlotResponse(readings []monitor.Reading) ([]byte, error) {
	timestampsMap := make(map[int64]struct{})
	seriesMap := make(map[string]map[int64]any)

	for _, r := range readings {
		ts := r.Date.Unix()
		timestampsMap[ts] = struct{}{}

		if seriesMap[r.Key] == nil {
			seriesMap[r.Key] = make(map[int64]any)
		}

		seriesMap[r.Key][ts] = r.Value
	}

	var timestamps []int64
	for ts := range timestampsMap {
		timestamps = append(timestamps, ts)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })

	result := [][]any{}
	xRow := make([]any, len(timestamps))
	for i, ts := range timestamps {
		xRow[i] = ts
	}
	result = append(result, xRow)

	keys := make([]string, 0, len(seriesMap))
	for k := range seriesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		row := make([]any, len(timestamps))
		for i, ts := range timestamps {
			if v, ok := seriesMap[k][ts]; ok {
				row[i] = v
			} else {
				row[i] = nil
			}
		}
		result = append(result, row)
	}

	b, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
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

func boolFromString(raw string) bool {
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}

	return b
}

type Widget struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Strict bool   `json:"strict,omitempty"`
}

type Server struct {
	svc       *monitor.Service
	dashboard []Widget
}

func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	log.DebugContext(r.Context(), "request", "method", r.Method, "URI", r.RequestURI)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(s.dashboard); err != nil {
		log.ErrorContext(r.Context(), "failed to marshal dashboard", log.Any("error", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
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
	strict := boolFromString(r.URL.Query().Get("strict"))

	readings, err := s.svc.Metric(r.Context(), metric, minDate, maxDate, strict)
	if err != nil {
		log.ErrorContext(r.Context(), "failed to get readings", log.Any("error", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	b, err := uPlotResponse(readings)
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
	mux.Handle("/", http.FileServer(http.Dir("../../static")))
	mux.HandleFunc("/dashboard", s.dashboardHandler)
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

func NewHTTPServer(svc *monitor.Service, cfg []conf.Widget) *Server {
	dashboard := make([]Widget, len(cfg))

	for i, w := range cfg {
		dashboard[i] = Widget{
			Key:    w.Key,
			Title:  w.Title,
			Width:  w.Width,
			Height: w.Height,
			Strict: w.Strict,
		}
	}

	return &Server{svc: svc, dashboard: dashboard}
}
