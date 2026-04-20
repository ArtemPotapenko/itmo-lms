package platform

import (
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

func StartGRPC(addr string, register func(*grpc.Server)) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server := grpc.NewServer()
	register(server)
	go func() {
		slog.Info("grpc server started", "addr", addr)
		if err := server.Serve(lis); err != nil {
			slog.Error("grpc server stopped", "addr", addr, "err", err)
		}
	}()
	return server, nil
}
