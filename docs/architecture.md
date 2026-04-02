# Architecture

## Component Overview

```
                    ┌──────────────────────────────────┐
                    │         rate-limiter pod         │
  Client ─ POST──>  │   HTTP: 8081   gRPC: 50051       │
                    │       │            │             │
                    │       └──────┬─────┘             │
                    │              │                   │
                    │   limiter.AllowRequest()         │
                    │              │                   │
                    │     store.AllowRequest()         │
                    │              │                   │
                    └──────────────┼───────────────────┘
                                   │  EVALSHA (Lua)
                                   ▼
                    ┌──────────────────────────────────┐
                    │           Redis                  │
                    │   sorted set per API key         │
                    │   sliding_window.lua (atomic)    │
                    └──────────────────────────────────┘
```

## Request Lifecycle

1. Client sends `POST \check` with `X-API-KEY: <key>` header.
2. HTTP handler calls `limiter.AllowRequest(ctx, apiKey)`.
3. Limiter looks up the `Policy` for the key (per-key override of default: 100 req/min).
4. Limiter calls `store.AllowRequest(ctx, key, policy)`.
5. Store issues `EVALSHA <sha> rl:<key> <now_ms> <window_ms> <limit>` to Redis.
6. Lua script atomically decides allow (1) or reject (0).
7. Store returns `(bool, error)` to limiter.
8. Metrics recorded: `ratelimiter_requests_total{status=allowed|rejected}
9. HTTP handler writes `200 OK` or `429 Too Many Requests` with JSON body.

## Kubernetes deployment

```
┌───────────────────────────────────────────────────┐
│  Kubernetes cluster                               │
│                                                   │
│  ┌─────────────────────────────┐                  │
│  │  rate-limiter Deployment    │                  │
│  │  replicas: 3 (HPA: 3–10)    │                  │
│  │                             │                  │
│  │  pod-0  pod-1  pod-2        │                  │
│  └──────────┬──────────────────┘                  │
│             │ ClusterIP Service :8080/:50051      │
│             ▼                                     │
│  ┌──────────────────────────┐                     │
│  │  Redis StatefulSet       │                     │
│  │  redis-0 (PVC: 1Gi)      │                     │
│  └──────────────────────────┘                     │
│                                                   │
│  Prometheus ─── scrapes /metrics on each pod      │
│  Grafana    ─── dashboard: deploy/grafana/        │
└───────────────────────────────────────────────────┘
```

## Package dependency graph

```
cmd/server
    ├── internal/config      (load env/yaml, policy map)
    ├── internal/limiter     (AllowRequest — core domain)
    │       └── internal/store (Store interface)
    │               └── Redis + scripts/lua/
    ├── internal/metrics     (Prometheus counters + histograms)
    ├── internal/grpc        (thin gRPC handler)
    └── internal/http        (thin HTTP handler)
```

No circular dependencies. `internal/limiter` defines the `Store` interface it depends on (dependency inversion) - this is why `store/store.go` imports `internal/limiter` rather than the other way around.

## Test Strategy

| Layer | Tool | Infrastructure |
|-------|------|----------------|
| Algorithm unit tests | `go test` | None (pure Go) |
| Limiter unit tests | `go test` with `FakeStore` | None |
| Store integration tests | `go test -tags=integration` | Live Redis |
| Concurrency correctness | `go test -tags=integration` | Live Redis + running service |
| Load / latency | k6, vegeta | Live Redis + running service |
| Failure scenarios | Manual (documented in failure_test.go) | k8s cluster |
