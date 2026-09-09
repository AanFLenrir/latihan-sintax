package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepository interface {
	Save(ctx context.Context, userID int, token string) error
	FindByToken(ctx context.Context, token string) (int, error)
	Delete(ctx context.Context, token string) error
}

type tokenRepository struct {
	db *pgxpool.Pool
}

func NewTokenRepository(db *pgxpool.Pool) TokenRepository {
	// Hapus tabel lama kalau ada, lalu buat ulang dari nol supaya strukturnya bersih
	dropQuery := `DROP TABLE IF EXISTS refresh_tokens CASCADE;`
	_, _ = db.Exec(context.Background(), dropQuery)

	createQuery := `
	CREATE TABLE refresh_tokens (
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL,
		token TEXT NOT NULL UNIQUE,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	_, _ = db.Exec(context.Background(), createQuery)

	return &tokenRepository{db: db}
}

func (r *tokenRepository) Save(ctx context.Context, userID int, token string) error {
	query := `INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err := r.db.Exec(ctx, query, userID, token, expiresAt)
	return err
}

func (r *tokenRepository) FindByToken(ctx context.Context, token string) (int, error) {
	query := `SELECT user_id FROM refresh_tokens WHERE token = $1 AND expires_at > NOW()`
	var userID int
	err := r.db.QueryRow(ctx, query, token).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("refresh token tidak valid atau kadaluarsa")
		}
		return 0, err
	}
	return userID, nil
}

func (r *tokenRepository) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`
	_, err := r.db.Exec(ctx, query, token)
	return err
}