package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	authv1 "itmo-lms/auth-service/gen"
	"itmo-lms/auth-service/internal/application"
	infra "itmo-lms/auth-service/internal/infrastructure/postgres"
	grpctransport "itmo-lms/auth-service/internal/transport/grpc"
	httptransport "itmo-lms/auth-service/internal/transport/http"
	"itmo-lms/pkg/platform"
	pg "itmo-lms/pkg/postgres"
)

func main() {
	ctx := context.Background()
	port := platform.EnvInt("PORT", 8081)
	host := platform.Env("SERVICE_HOST", "127.0.0.1")
	platform.RegisterConsul(platform.Env("CONSUL_URL", ""), "auth-service", platform.Env("SERVICE_ID", "auth-service-1"), host, port)
	db, err := pg.Open(ctx, platform.Env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/itmo_lms_auth?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := pg.RunMigrations(ctx, db, "auth-service", infra.Migrations); err != nil {
		log.Fatal(err)
	}
	repo := infra.NewUserRepository(db)
	service := application.NewService(repo, platform.Env("JWT_SECRET", "dev-secret"))
	if err := service.SeedAdmin(ctx); err != nil {
		log.Fatal(err)
	}
	grpcServer := grpctransport.New(service, platform.Env("JWT_SECRET", "dev-secret"))
	if _, err := platform.StartGRPC(platform.Env("GRPC_ADDR", ":9081"), func(server *grpc.Server) {
		authv1.RegisterAuthServiceServer(server, grpcServer)
	}); err != nil {
		log.Fatal(err)
	}
	handler := httptransport.New(service, platform.Env("JWT_SECRET", "dev-secret"))
	if err := platform.RunHTTP(platform.Env("ADDR", ":8081"), handler.Routes()); err != nil {
		log.Fatal(err)
	}
}
