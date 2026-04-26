package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	contentv1 "itmo-lms/content-service/gen"
	"itmo-lms/content-service/internal/application"
	grpcclient "itmo-lms/content-service/internal/infrastructure/grpc"
	httpclient "itmo-lms/content-service/internal/infrastructure/http"
	kafkainfra "itmo-lms/content-service/internal/infrastructure/kafka"
	infra "itmo-lms/content-service/internal/infrastructure/postgres"
	grpctransport "itmo-lms/content-service/internal/transport/grpc"
	httptransport "itmo-lms/content-service/internal/transport/http"
	basekafka "itmo-lms/pkg/kafka"
	"itmo-lms/pkg/platform"
	pg "itmo-lms/pkg/postgres"
)

func main() {
	ctx := context.Background()
	port := platform.EnvInt("PORT", 8082)
	host := platform.Env("SERVICE_HOST", "127.0.0.1")
	platform.RegisterConsul(platform.Env("CONSUL_URL", ""), "content-service", platform.Env("SERVICE_ID", "content-service-1"), host, port)
	db, err := pg.Open(ctx, platform.Env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/itmo_lms_content?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := pg.RunMigrations(ctx, db, "content-service", infra.Migrations); err != nil {
		log.Fatal(err)
	}
	var compiler application.DocumentCompiler
	if target := platform.Env("DOCUMENT_SERVICE_GRPC_ADDR", ""); target != "" {
		client, err := grpcclient.NewDocumentClient(target)
		if err != nil {
			log.Fatal(err)
		}
		compiler = client
	}
	var evaluator application.TaskEvaluator
	if target := platform.Env("AI_EVALUATOR_URL", ""); target != "" {
		evaluator = httpclient.NewAIEvaluatorClient(target)
	}
	service := application.NewService(infra.NewRepository(db), compiler, evaluator)
	if _, err := platform.StartGRPC(platform.Env("GRPC_ADDR", ":9082"), func(server *grpc.Server) {
		contentv1.RegisterContentServiceServer(server, grpctransport.New(service))
	}); err != nil {
		log.Fatal(err)
	}
	var attemptPublisher *kafkainfra.AttemptPublisher
	if brokers := basekafka.BrokersFromEnv(platform.Env("KAFKA_BROKERS", "")); len(brokers) > 0 {
		attemptPublisher = kafkainfra.NewAttemptPublisher(basekafka.NewPublisher(brokers, platform.Env("KAFKA_ATTEMPTS_TOPIC", "attempt-events")))
	}
	handler := httptransport.New(service, platform.Env("JWT_SECRET", "dev-secret"), attemptPublisher)
	if err := platform.RunHTTP(platform.Env("ADDR", ":8082"), handler.Routes()); err != nil {
		log.Fatal(err)
	}
}
