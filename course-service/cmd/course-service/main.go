package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	coursev1 "itmo-lms/course-service/gen"
	"itmo-lms/course-service/internal/application"
	infra "itmo-lms/course-service/internal/infrastructure/postgres"
	grpctransport "itmo-lms/course-service/internal/transport/grpc"
	httptransport "itmo-lms/course-service/internal/transport/http"
	"itmo-lms/pkg/platform"
	pg "itmo-lms/pkg/postgres"
)

func main() {
	ctx := context.Background()
	port := platform.EnvInt("PORT", 8083)
	host := platform.Env("SERVICE_HOST", "127.0.0.1")
	platform.RegisterConsul(platform.Env("CONSUL_URL", ""), "course-service", platform.Env("SERVICE_ID", "course-service-1"), host, port)
	db, err := pg.Open(ctx, platform.Env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/itmo_lms_course?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := pg.RunMigrations(ctx, db, "course-service", infra.Migrations); err != nil {
		log.Fatal(err)
	}
	service := application.NewService(infra.NewRepository(db))
	if _, err := platform.StartGRPC(platform.Env("GRPC_ADDR", ":9083"), func(server *grpc.Server) {
		coursev1.RegisterCourseServiceServer(server, grpctransport.New(service))
	}); err != nil {
		log.Fatal(err)
	}
	handler := httptransport.New(service, platform.Env("JWT_SECRET", "dev-secret"))
	if err := platform.RunHTTP(platform.Env("ADDR", ":8083"), handler.Routes()); err != nil {
		log.Fatal(err)
	}
}
