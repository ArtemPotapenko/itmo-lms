package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	documentv1 "itmo-lms/document-service/gen"
	"itmo-lms/document-service/internal/application"
	infra "itmo-lms/document-service/internal/infrastructure/postgres"
	grpctransport "itmo-lms/document-service/internal/transport/grpc"
	httptransport "itmo-lms/document-service/internal/transport/http"
	"itmo-lms/pkg/platform"
	pg "itmo-lms/pkg/postgres"
)

func main() {
	ctx := context.Background()
	port := platform.EnvInt("PORT", 8084)
	host := platform.Env("SERVICE_HOST", "127.0.0.1")
	platform.RegisterConsul(platform.Env("CONSUL_URL", ""), "document-service", platform.Env("SERVICE_ID", "document-service-1"), host, port)
	db, err := pg.Open(ctx, platform.Env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/itmo_lms_document?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := pg.RunMigrations(ctx, db, "document-service", infra.Migrations); err != nil {
		log.Fatal(err)
	}
	service := application.NewService(infra.NewRepository(db), platform.Env("DATA_DIR", "data/documents"))
	if _, err := platform.StartGRPC(platform.Env("GRPC_ADDR", ":9084"), func(server *grpc.Server) {
		documentv1.RegisterDocumentServiceServer(server, grpctransport.New(service))
	}); err != nil {
		log.Fatal(err)
	}
	handler := httptransport.New(service, platform.Env("JWT_SECRET", "dev-secret"))
	if err := platform.RunHTTP(platform.Env("ADDR", ":8084"), handler.Routes()); err != nil {
		log.Fatal(err)
	}
}
