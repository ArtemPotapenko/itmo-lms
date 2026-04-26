package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	basekafka "itmo-lms/pkg/kafka"
	"itmo-lms/pkg/platform"
	pg "itmo-lms/pkg/postgres"
	"itmo-lms/pkg/rediscache"
	statisticv1 "itmo-lms/statistic-service/gen"
	"itmo-lms/statistic-service/internal/application"
	grpcclient "itmo-lms/statistic-service/internal/infrastructure/grpc"
	kafkainfra "itmo-lms/statistic-service/internal/infrastructure/kafka"
	infra "itmo-lms/statistic-service/internal/infrastructure/postgres"
	grpctransport "itmo-lms/statistic-service/internal/transport/grpc"
	httptransport "itmo-lms/statistic-service/internal/transport/http"
)

func main() {
	ctx := context.Background()
	port := platform.EnvInt("PORT", 8085)
	host := platform.Env("SERVICE_HOST", "127.0.0.1")
	platform.RegisterConsul(platform.Env("CONSUL_URL", ""), "statistic-service", platform.Env("SERVICE_ID", "statistic-service-1"), host, port)
	db, err := pg.Open(ctx, platform.Env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/itmo_lms_statistic?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := pg.RunMigrations(ctx, db, "statistic-service", infra.Migrations); err != nil {
		log.Fatal(err)
	}
	var metadata application.MetadataProvider
	if target := platform.Env("CONTENT_SERVICE_GRPC_ADDR", ""); target != "" {
		client, err := grpcclient.NewContentClient(target)
		if err != nil {
			log.Fatal(err)
		}
		metadata = client
	}
	var cache application.Cache
	if addr := platform.Env("REDIS_ADDR", ""); addr != "" {
		cache = rediscache.New(addr)
	}
	cacheTTL := 2 * time.Hour
	if raw := platform.Env("STAT_CACHE_TTL", "2h"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			cacheTTL = parsed
		}
	}
	service := application.NewService(infra.NewRepository(db), metadata, cache, cacheTTL)
	if brokers := basekafka.BrokersFromEnv(platform.Env("KAFKA_BROKERS", "")); len(brokers) > 0 {
		consumer := kafkainfra.NewAttemptConsumer(
			basekafka.NewConsumer(brokers, platform.Env("KAFKA_ATTEMPTS_TOPIC", "attempt-events"), platform.Env("KAFKA_ATTEMPTS_GROUP", "statistic-service")),
			service,
		)
		go func() {
			if err := consumer.Consume(context.Background()); err != nil {
				log.Printf("attempt consumer stopped: %v", err)
			}
		}()
	}
	if _, err := platform.StartGRPC(platform.Env("GRPC_ADDR", ":9085"), func(server *grpc.Server) {
		statisticv1.RegisterStatisticServiceServer(server, grpctransport.New(service))
	}); err != nil {
		log.Fatal(err)
	}
	handler := httptransport.New(service, platform.Env("JWT_SECRET", "dev-secret"))
	if err := platform.RunHTTP(platform.Env("ADDR", ":8085"), handler.Routes()); err != nil {
		log.Fatal(err)
	}
}
