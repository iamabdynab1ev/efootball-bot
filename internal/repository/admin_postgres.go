// admin_postgres.go - Postgres'да админларни бошқариш учун репозиторий
package repository

import (
	"context"
	"efootball-bot/internal/models"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Интерфейс ────────────────────────────────────────────────────────────────

type AdminRepository interface {
	// Текшириш
	GetRole(ctx context.Context, telegramID int64) (models.AdminRole, error) // "" = admin эмас
	IsAdmin(ctx context.Context, telegramID int64) (bool, error)
	IsSuperAdmin(ctx context.Context, telegramID int64) (bool, error)

	// Бошқариш (фақат super_admin)
	Add(ctx context.Context, userID int64, role models.AdminRole) error
	Remove(ctx context.Context, userID int64) error
	List(ctx context.Context) ([]*models.Admin, error)
}

// ─── Реализация ───────────────────────────────────────────────────────────────

type adminRepo struct {
	db *pgxpool.Pool
}

func NewAdminRepository(db *pgxpool.Pool) AdminRepository {
	return &adminRepo{db: db}
}

func (r *adminRepo) GetRole(ctx context.Context, telegramID int64) (models.AdminRole, error) {
	var role models.AdminRole
	err := r.db.QueryRow(ctx, `
		SELECT a.role FROM admins a
		JOIN users u ON u.id = a.user_id
		WHERE u.telegram_id = $1
	`, telegramID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil // admin эмас
	}
	return role, err
}

func (r *adminRepo) IsAdmin(ctx context.Context, telegramID int64) (bool, error) {
	role, err := r.GetRole(ctx, telegramID)
	return role != "", err
}

func (r *adminRepo) IsSuperAdmin(ctx context.Context, telegramID int64) (bool, error) {
	role, err := r.GetRole(ctx, telegramID)
	return role == models.RoleSuperAdmin, err
}

func (r *adminRepo) Add(ctx context.Context, userID int64, role models.AdminRole) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO admins (user_id, role)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET role = EXCLUDED.role
	`, userID, role)
	return err
}

func (r *adminRepo) Remove(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM admins WHERE user_id = $1`, userID)
	return err
}

func (r *adminRepo) List(ctx context.Context) ([]*models.Admin, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.user_id, a.role, a.created_at,
		       u.telegram_id, u.display_name, u.username
		FROM admins a
		JOIN users u ON u.id = a.user_id
		ORDER BY a.role, a.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Admin
	for rows.Next() {
		a := &models.Admin{User: &models.User{}}
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Role, &a.CreatedAt,
			&a.User.TelegramID, &a.User.DisplayName, &a.User.Username,
		); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
