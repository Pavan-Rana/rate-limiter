/**
 * k6 load test — ramping VUs from 100 to 5000 RPS
 *
 * Usage:
 *   k6 run tests/load/k6_ramp.js
 *   k6 run tests/load/k6_ramp.js --out json=tests/load/benchmarks/results.json
 *
 * Environment variables:
 *   SERVICE_URL — base URL of the rate-limiter service (default: http://localhost:8080)
 *   API_KEY_COUNT — number of distinct API keys to rotate across (default: 10)
 *
 * After the run, copy the headline numbers into tests/load/benchmarks/results.md.
 * Those numbers become your CV bullet metrics — only report what you measured.
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

const rejectionRate    = new Rate('rejection_rate');
const decisionLatency  = new Trend('decision_latency_ms', true);
const admittedRequests = new Counter('admitted_requests');
const rejectedRequests = new Counter('rejected_requests');

export const options = {
  stages: [
    { duration: '30s', target: 50   },  // warm-up
    { duration: '60s', target: 200  },  // ramp
    { duration: '60s', target: 500  },  // sustained mid
    { duration: '60s', target: 1000 },  // sustained target
    { duration: '60s', target: 2000 },  // push beyond
    { duration: '30s', target: 5000 },  // peak stress
    { duration: '30s', target: 0    },  // cool down
  ],
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'], 
  thresholds: {
    // p99 decision latency must stay under 10ms at sustained load
    'http_req_duration{status:200}': ['p(99)<10'],
    // Non-429 errors should be negligible
    'http_req_failed{status:!429}': ['rate<0.005'],
  },
};

const BASE_URL      = __ENV.SERVICE_URL   || 'http://localhost:8080';
const API_KEY_COUNT = parseInt(__ENV.API_KEY_COUNT || '10', 10);

export default function () {
  const apiKey = `load-test-key-${Math.floor(Math.random() * API_KEY_COUNT)}`;
  const start  = Date.now();

  const res = http.post(`${BASE_URL}/check`, null, {
    headers: { 'X-API-Key': apiKey },
    tags:    { api_key: apiKey },
  });

  decisionLatency.add(Math.max(0, Date.now() - start));

  const isRejected = res.status === 429;
  rejectionRate.add(isRejected);

  if (isRejected) {
    rejectedRequests.add(1);
  } else {
    admittedRequests.add(1);
  }

  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'response has body':     (r) => r.body && r.body.length > 0,
  });
  sleep(Math.random() * 2);
}

export function handleSummary(data) {
  return {
    'tests/load/benchmarks/summary.json': JSON.stringify(data, null, 2),
  };
}
