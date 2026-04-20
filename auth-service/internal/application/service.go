package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"itmo-lms/auth-service/internal/domain"
	"itmo-lms/pkg/platform"
)

var ErrConflict = errors.New("user already exists")
var ErrUnauthorized = errors.New("invalid credentials")
var ErrNotFound = errors.New("user not found")

type Service struct {
	repo   domain.UserRepository
	secret string
}

type RegisterCommand struct {
	Phone     string
	Email     string
	FirstName string
	LastName  string
	Nick      string
	Password  string
	Roles     []string
}

func NewService(repo domain.UserRepository, secret string) *Service {
	return &Service{repo: repo, secret: secret}
}

func (s *Service) SeedAdmin(ctx context.Context) error {
	return s.repo.SeedAdmin(ctx, domain.User{
		ID:           "usr_admin",
		Phone:        "admin",
		FirstName:    "System",
		LastName:     "Admin",
		Nick:         "admin",
		PasswordHash: platform.Hash("admin:admin"),
		Roles:        []string{"admin", "teacher"},
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	})
}

func (s *Service) Register(ctx context.Context, cmd RegisterCommand) (domain.User, error) {
	cmd.Phone = strings.TrimSpace(cmd.Phone)
	if cmd.Phone == "" || cmd.Password == "" {
		return domain.User{}, errors.New("phone and password are required")
	}
	if len(cmd.Roles) == 0 {
		cmd.Roles = []string{"student"}
	}
	user := domain.User{
		ID:           platform.NewID("usr"),
		Phone:        cmd.Phone,
		Email:        strings.TrimSpace(cmd.Email),
		FirstName:    cmd.FirstName,
		LastName:     cmd.LastName,
		Nick:         cmd.Nick,
		PasswordHash: platform.Hash(cmd.Phone + ":" + cmd.Password),
		Roles:        cmd.Roles,
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, user); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return domain.User{}, ErrConflict
		}
		return domain.User{}, err
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, phone, password string) (string, domain.User, error) {
	user, ok, err := s.repo.FindByPhone(ctx, phone)
	if err != nil {
		return "", domain.User{}, err
	}
	if !ok || user.PasswordHash != platform.Hash(phone+":"+password) {
		return "", domain.User{}, ErrUnauthorized
	}
	token, err := platform.SignToken(s.secret, platform.Claims{Subject: user.ID, Roles: user.Roles, Expires: time.Now().Add(24 * time.Hour).Unix()})
	if err != nil {
		return "", domain.User{}, err
	}
	return token, user, nil
}

func (s *Service) Me(ctx context.Context, userID string) (domain.User, error) {
	user, ok, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	if !ok {
		return domain.User{}, ErrNotFound
	}
	return user, nil
}

func (s *Service) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.List(ctx)
}
