# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go modules files and download deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN go build -o rate-limiter ./cmd/server

# Stage 2: Runtime
FROM alpine:3.18

WORKDIR /app

# Copy built binary
COPY --from=builder /app/rate-limiter .

# Expose port
EXPOSE 8080

# Run the service
CMD ["./rate-limiter"]