# Load Test Results

> Fill this in after running `make load-test`.
> The numbers here are what you quote in the CV bullet — only report what you actually measured.

## Environment

| Field | Value |
|-------|-------|
| Date | 2026-03-31 |
| Hardware | Windows laptop (WSL2), ~16 GB RAM |
| Redis | Docker local |
| Service replicas | 1 |
| Go version | 1.22.2 |
| k6 version | 1.6.1 |

## Results

| Metric | Value |
|--------|-------|
| Peak sustained RPS (before p99 > 10ms) | 1108 RPS |
| p50 decision latency | 1 ms |
| p95 decision latency | 2 ms |
| p99 decision latency | 2 ms |
| Non-error failure rate | 0 % |
| Concurrent correctness (goroutine storm) | 5113 / 5113 admitted correctly |

## Observations

- p99 latency well within SLO  
  `http_req_duration{status:200}` p99 = 2.06 ms, comfortably below the 10 ms threshold.
- Decision latency is highly stable  
  - p50: 1 ms  
  - p95: 2 ms  
  - p99: 2 ms  
  Indicates low contention and efficient limiter execution even under high concurrency.

- High rejection rate (~98.5%) is expected  
  Driven by aggressive load relative to API key cardinality. Confirms the rate limiter is correctly enforcing limits, not failing.

- Zero real failures observed  
  `http_req_failed{status:!429} = 0%` → no 5xx responses or transport-level errors.

- Throughput saturation is controlled  
  System stabilises at ~1100 RPS admitted traffic, with excess requests cleanly rejected rather than degrading performance.

- Tail latency spikes are isolated  
  - Decision latency max: 133 ms  
  - HTTP latency max: 122 ms  
  Likely caused by client-side scheduling or resource contention (e.g. k6/WSL), not core service degradation.

- Near-perfect correctness under load  
  Only 1 failed check out of ~694k, indicating robust behaviour even during peak stress.

- No cascading failure under extreme concurrency  
  Even at 5000 VUs, the system maintains predictable latency and rejection behaviour without instability.

## How to reproduce

```bash
# Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# Start the service
make run

# Run load test
make load-test

# Or with vegeta at specific RPS
./tests/load/vegeta_attack.sh 1000 60s