# Distributed Rate Limiter

A horizontally scalable rate-limiting service enforcing per-client quotas with strong
correctness guarantees under concurrent load.

## What this demonstrates

| Signal | How |
|--------|-----|
| Distributed systems | Atomic decisions across stateless replicas via Redis Lua scripts |
| Concurrency & correctness | Goroutine-storm integration test; correctness argument in `docs/tradeoffs.md` |
| Performance validation | k6 load test with committed p99 latency results |
| Production observability | Prometheus counters + latency histograms + Grafana dashboard |
| Deployment & reliability | Kubernetes with rolling updates, HPA, health probes, Redis StatefulSet |
| Engineering tradeoffs | Algorithm choice, atomicity, failure behaviour in `docs/tradeoffs.md` |

## Quick start

```bash
# Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# Run the service
make run

# Test a request
curl -X POST http://localhost:8080/check -H "X-API-Key: my-key"

# Unit tests
make test

# Integration tests (requires running service + Redis)
make test-integration

# Load test (requires k6)
make load-test
```

## API

POST /check with X-API-Key header -> 200 (allowed) or 429 (rejected)
GET /metrics -> Prometheus metrics
gRPC AllowRequest on :50051 (see proto/ratelimiter.proto)

## Project structure

| Path | Purpose |
|------|---------|
| cmd/server/ | Entry point |
| internal/algorithm/ | Pure Go sliding window + token bucket |
| internal/limiter/ | Core domain — AllowRequest() |
| internal/store/ | Redis abstraction + Lua script loading |
| internal/metrics/ | Prometheus instrumentation |
| internal/grpc/ | gRPC handler |
| internal/http/ | HTTP handler |
| scripts/lua/ | Atomic Redis Lua scripts |
| tests/integration/ | Correctness, concurrency, failure tests |
| tests/load/ | k6 and vegeta load tests + benchmark results |
| deploy/k8s/ | Kubernetes manifests |
| deploy/grafana/ | Grafana dashboard |
| docs/ | Architecture and tradeoff docs |

## Documentation

- docs/architecture.md — component diagram, request lifecycle, test strategy
- docs/tradeoffs.md — algorithm choice, atomicity, failure behaviour, memory model
- tests/load/benchmarks/results.md — measured latency under load