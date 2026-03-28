.PHONY: run test test-integration test-race load-test proto lint build docker k8s-apply

run:
	go run ./cmd/server/...

test:
	go test ./internal/... -v -race

test-integration:
	go test -tags=integration ./tests/integration/... -v

test-race:
	go test ./... -race

load-test:
	k6 run ./tests/load/k6_ramp.js --out json=tests/load/benchmarks/result.json

load-test-vegeta:
	./tests/load/vegeta_attack.sh 1000 60s

proto:
	protoc --go_out=. --go-grpc_out=. proto/ratelimiter.proto

lint:
	golangci-lint run ./...

build:
	go build -o bin/server ./cmd/server/...

docker:
	docker build -t rate-limiter:latest .

k8s-apply:
	kubectl apply -f deploy/k8s/redis.yaml
	kubectl apply -f deploy/k8s/configmap.yaml
	kubectl apply -f deploy/k8s/deployment.yaml
	kubectl apply -f deploy/k8s/service.yaml
	kubectl apply -f deploy/k8s/hpa.yaml