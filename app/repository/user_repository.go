package repository

import (
	"context"
	"errors"
	"latihan-sintaks/app/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.UserResponse, passwordHash string) error
	FindByUsername(ctx context.Context, username string) (*model.UserInternal, error)
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.UserResponse, passwordHash string) error {
	query := `INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, 'user') RETURNING id, created_at`
	err := r.db.QueryRow(ctx, query, user.Username, user.Email, passwordHash).Scan(&user.ID, &user.CreatedAt)
	return err
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.UserInternal, error) {
	query := `SELECT id, username, email, password, role, created_at FROM users WHERE username = $1`
	row := r.db.QueryRow(ctx, query, username)

	var user model.UserInternal
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}
	return &user, nil
}