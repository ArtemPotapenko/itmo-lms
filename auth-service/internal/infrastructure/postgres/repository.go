package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"itmo-lms/auth-service/internal/domain"
	pg "itmo-lms/pkg/postgres"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		insert into users(id, phone, email, first_name, last_name, nick, password_hash, roles_json, status, created_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, user.ID, user.Phone, user.Email, user.FirstName, user.LastName, user.Nick, user.PasswordHash, pg.Marshal(user.Roles), user.Status, user.CreatedAt)
	return err
}

func (r *UserRepository) FindByPhone(ctx context.Context, phone string) (domain.User, bool, error) {
	return r.scanOne(ctx, `select id, phone, email, first_name, last_name, nick, password_hash, roles_json, status, created_at from users where phone=$1`, phone)
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, bool, error) {
	return r.scanOne(ctx, `select id, phone, email, first_name, last_name, nick, password_hash, roles_json, status, created_at from users where id=$1`, id)
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `select id, phone, email, first_name, last_name, nick, password_hash, roles_json, status, created_at from users order by created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		user, err := scanUser(rows.Scan)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *UserRepository) SeedAdmin(ctx context.Context, user domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		insert into users(id, phone, email, first_name, last_name, nick, password_hash, roles_json, status, created_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		on conflict (id) do nothing
	`, user.ID, user.Phone, user.Email, user.FirstName, user.LastName, user.Nick, user.PasswordHash, pg.Marshal(user.Roles), user.Status, user.CreatedAt)
	return err
}

func (r *UserRepository) scanOne(ctx context.Context, query string, arg string) (domain.User, bool, error) {
	row := r.db.QueryRowContext(ctx, query, arg)
	user, err := scanUser(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, false, nil
		}
		return domain.User{}, false, err
	}
	return user, true, nil
}

func scanUser(scan func(dest ...any) error) (domain.User, error) {
	var user domain.User
	var rolesRaw []byte
	if err := scan(&user.ID, &user.Phone, &user.Email, &user.FirstName, &user.LastName, &user.Nick, &user.PasswordHash, &rolesRaw, &user.Status, &user.CreatedAt); err != nil {
		return domain.User{}, err
	}
	user.Roles = pg.Unmarshal[[]string](rolesRaw)
	return user, nil
}

func IsUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}
