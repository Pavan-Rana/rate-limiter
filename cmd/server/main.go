package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pavan-Rana/rate-limiter/internal/config"
	grpcserver "github.com/Pavan-Rana/rate-limiter/internal/grpc"
	httpserver "github.com/Pavan-Rana/rate-limiter/internal/http"
	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
	"github.com/Pavan-Rana/rate-limiter/internal/metrics"
	"github.com/Pavan-Rana/rate-limiter/internal/store"

	pb "github.com/Pavan-Rana/rate-limiter/proto"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	redisStore, err := store.NewRedisStore(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("Failed to create Redis store: %v", err)
	}

	metrics.Register()

	lim := limiter.New(redisStore, cfg)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	pb.RegisterRateLimiterServer(grpcSrv, grpcserver.New(lim))

	go func() {
		log.Printf("gRPC listening on %s", cfg.GRPCAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	httpSrv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpserver.NewRouter(lim),
	}
	go func() {
		log.Printf("HTTP listening on %s", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("Shutting down...")
	grpcSrv.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}
