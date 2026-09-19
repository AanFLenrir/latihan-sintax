package repository

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleRepository interface {
	LoadPermissions(ctx context.Context) (map[string][]string, error)
}

type rolePostgresRepository struct {
	pool *pgxpool.Pool
}

func NewRoleRepository(pool *pgxpool.Pool) RoleRepository {
	return &rolePostgresRepository{pool: pool}
}

// LoadPermissions mengambil pemetaan role ke daftar permission langsung dari tabel role_permissions
func (r *rolePostgresRepository) LoadPermissions(ctx context.Context) (map[string][]string, error) {
	// Menyesuaikan dengan kolom di file 004_student_permissions.sql (role_name dan permission_name)
	query := `
		SELECT role_name, permission_name
		FROM role_permissions
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal query permissions: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var role, permission string
		if err := rows.Scan(&role, &permission); err != nil {
			return nil, fmt.Errorf("gagal scan row permission: %w", err)
		}
		result[role] = append(result[role], permission)
	}

	return result, nil
}