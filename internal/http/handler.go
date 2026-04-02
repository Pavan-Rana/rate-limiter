package http

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"time"

	"github.com/Pavan-Rana/rate-limiter/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Limiter interface {
	AllowRequest(ctx context.Context, apiKey string) (bool, error)
}

func NewRouter(lim Limiter) stdhttp.Handler {
	mux := stdhttp.NewServeMux()
	sem := make(chan struct{}, 1000) // Limit to 1000 concurrent requests

	mux.HandleFunc("/check", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		start := time.Now()
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
		default:
			stdhttp.Error(w, "server overloaded", stdhttp.StatusServiceUnavailable)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 50*time.Millisecond)
		defer cancel()

		// Read from header, matching what k6 sends
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			stdhttp.Error(w, "Missing X-API-Key", stdhttp.StatusBadRequest)
			return
		}

		allowed, err := lim.AllowRequest(ctx, apiKey)
		metrics.RecordDecision(apiKey, allowed)
		metrics.RecordDecisionLatency(time.Since(start))
		if err != nil {
			metrics.RecordRedisError()
			stdhttp.Error(w, err.Error(), stdhttp.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if allowed {
			w.WriteHeader(stdhttp.StatusOK)
		} else {
			w.WriteHeader(stdhttp.StatusTooManyRequests)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"allowed": allowed,
			"api_key": apiKey,
		})
	})

	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func checkHandler(lim Limiter) stdhttp.HandlerFunc {
	return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Method != stdhttp.MethodPost {
			stdhttp.Error(w, "Method not allowed", stdhttp.StatusMethodNotAllowed)
			return
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			stdhttp.Error(w, "Missing X-API-Key", stdhttp.StatusBadRequest)
			return
		}

		allowed, _ := lim.AllowRequest(r.Context(), apiKey)

		status := stdhttp.StatusOK
		if !allowed {
			status = stdhttp.StatusTooManyRequests
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"allowed": allowed,
			"api_key": apiKey,
		})
	}
}
