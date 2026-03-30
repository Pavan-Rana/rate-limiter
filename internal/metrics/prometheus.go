package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ratelimiter_requests_total",
		Help: "Total rate-limit decisions, labelled by api_key and status (allowed|rejected)",
	}, []string{"api_key", "status"})

	decisionLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ratelimiter_decision_latency_seconds",
		Help:    "E2E latency of AllowRequest and Redis round-trip",
		Buckets: []float64{0.001, 0.005, 0.010, 0.025, 0.050, 0.100},
	})

	redisCommandDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ratelimiter_redis_command_duration_seconds",
		Help:    "Latency of Redis EVALSHA commands",
		Buckets: []float64{0.0005, 0.001, 0.005, 0.010, 0.025},
	})

	redisErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ratelimiter_redis_errors_total",
		Help: "Total Redis errors encountered. Non-zero triggers fail-open behaviour.",
	})
)

// Register is a no-op; promauto registers automatically on package import.
// Call it in main to ensure this package is imported and metrics are registered.
func Register() {}

func RecordDecision(apiKey string, allowed bool) {
	status := "rejected"
	if allowed {
		status = "allowed"
	}
	requestsTotal.WithLabelValues(apiKey, status).Inc()
}

func RecordRedisLatency(d time.Duration) {
	redisCommandDuration.Observe(d.Seconds())
}

func RecordDecisionLatency(d time.Duration) {
	decisionLatency.Observe(d.Seconds())
}

func RecordRedisError() {
	redisErrors.Inc()
}
