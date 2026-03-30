package http

import (
	"encoding/json"
	"net/http"

	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(lim *limiter.Limiter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/check", checkHandler(lim))
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func checkHandler(lim *limiter.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, "Missing X-API-Key", http.StatusBadRequest)
			return
		}

		allowed, _ := lim.AllowRequest(r.Context(), apiKey)

		status := http.StatusOK
		if !allowed {
			status = http.StatusTooManyRequests
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"allowed": allowed,
			"api_key": apiKey,
		})
	}
}
