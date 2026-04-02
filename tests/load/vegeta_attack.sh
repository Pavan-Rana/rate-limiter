#!/usr/bin/env bash
# Vegeta load test — sustained constant-rate attack
#
# Usage:
#   chmod +x tests/load/vegeta_attack.sh
#   ./tests/load/vegeta_attack.sh [rate] [duration]
#
# Defaults: 1000 RPS for 60s
# Output:   tests/load/benchmarks/vegeta_<rate>rps_<timestamp>.txt
#
# Install vegeta: go install github.com/tsenart/vegeta@latest

set -euo pipefail

RATE="${1:-1000}"
DURATION="${2:-60s}"
SERVICE_URL="${SERVICE_URL:-http://localhost:8080}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
OUT="tests/load/benchmarks/vegeta_${RATE}rps_${TIMESTAMP}.txt"
JSON_OUT="${OUT%.txt}.json"

mkdir -p tests/load/benchmarks

echo "Running vegeta: ${RATE} RPS for ${DURATION} → ${OUT}"

# Generate target file with rotating API keys
generate_targets() {
  for i in $(seq 1 10); do
    echo "POST ${SERVICE_URL}/check"
    echo "X-API-Key: vegeta-key-${i}"
    echo ""
  done
}

# Run attack and save text report
generate_targets \
  | vegeta attack -rate="${RATE}" -duration="${DURATION}" \
  | vegeta report -type=text | tee "${OUT}"

# Run attack again and save JSON report
generate_targets \
  | vegeta attack -rate="${RATE}" -duration="${DURATION}" \
  | vegeta report -type=json | tee "${JSON_OUT}" | python3 -m json.tool

# Extract p99 latency from JSON
P99_DECISION_LATENCY=$(jq -r '.latencies["99th"]' "${JSON_OUT}")
echo ""
echo "p99 latency: ${P99_DECISION_LATENCY} ms"

TOTAL_FAILS=$(jq '.status_codes | to_entries | map(select(.key != "200")) | map(.value) | add' "${JSON_OUT}")
FAIL_419=$(jq -r '.status_codes["419"] // 0' "${JSON_OUT}")
FAILS_NO_419=$((TOTAL_FAILS - FAIL_419))

echo "Failures (excluding 419): ${FAILS_NO_419}"

echo ""
echo "Results written to ${OUT} and ${JSON_OUT}"
echo "Copy the p99 latency into tests/load/benchmarks/results.md"
