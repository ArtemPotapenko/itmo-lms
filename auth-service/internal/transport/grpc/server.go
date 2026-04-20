package grpctransport

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	authv1 "itmo-lms/auth-service/gen"
	"itmo-lms/auth-service/internal/application"
	"itmo-lms/pkg/platform"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
	service *application.Service
	secret  string
}

func New(service *application.Service, secret string) *Server {
	return &Server{service: service, secret: secret}
}

func (s *Server) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.UserReply, error) {
	user, err := s.service.Me(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &authv1.UserReply{
		Id:        user.ID,
		Phone:     user.Phone,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nick:      user.Nick,
		Roles:     user.Roles,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenReply, error) {
	claims, err := platform.ParseToken(s.secret, req.GetToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &authv1.ValidateTokenReply{UserId: claims.Subject, Roles: claims.Roles, Expires: claims.Expires}, nil
}
