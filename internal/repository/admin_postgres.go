package repository

import (
	"context"
	"efootball-bot/internal/models"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Типы ─────────────────────────────────────────────────────────────────────

type AdminCredential struct {
	UserID       int64
	PasswordHash string
}

type UserWithRole struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	Rating      int    `json:"rating"`
	Rank        string `json:"rank"`
	TeamPower   int    `json:"team_power"`
	HasTelegram bool   `json:"has_telegram"`
	AdminRole   string `json:"admin_role"` // "" | "admin" | "super_admin"
}

// ─── Интерфейс ────────────────────────────────────────────────────────────────

type AdminRepository interface {
	// Telegram-based checks (bot)
	GetRole(ctx context.Context, telegramID int64) (models.AdminRole, error)
	IsAdmin(ctx context.Context, telegramID int64) (bool, error)
	IsSuperAdmin(ctx context.Context, telegramID int64) (bool, error)

	// User-ID-based checks (web)
	IsAdminByUserID(ctx context.Context, userID int64) (bool, error)
	IsSuperAdminByUserID(ctx context.Context, userID int64) (bool, error)
	GetAdminRoleByUserID(ctx context.Context, userID int64) (models.AdminRole, error)

	// Management
	Add(ctx context.Context, userID int64, role models.AdminRole) error
	Remove(ctx context.Context, userID int64) error
	List(ctx context.Context) ([]*models.Admin, error)

	// Users with roles (for admin panel)
	GetUsersWithRoles(ctx context.Context, limit int) ([]*UserWithRole, error)

	// Credentials (username/password login)
	SuperAdminExists(ctx context.Context) (bool, error)
	SeedSuperAdmin(ctx context.Context, username, passwordHash, displayName string) error
	GetAdminCredential(ctx context.Context, username string) (*AdminCredential, error)
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
		return "", nil
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

func (r *adminRepo) IsAdminByUserID(ctx context.Context, userID int64) (bool, error) {
	role, err := r.GetAdminRoleByUserID(ctx, userID)
	return role != "", err
}

func (r *adminRepo) IsSuperAdminByUserID(ctx context.Context, userID int64) (bool, error) {
	role, err := r.GetAdminRoleByUserID(ctx, userID)
	return role == models.RoleSuperAdmin, err
}

// GetAdminRoleByUserID возвращает роль пользователя одним запросом.
func (r *adminRepo) GetAdminRoleByUserID(ctx context.Context, userID int64) (models.AdminRole, error) {
	var role models.AdminRole
	err := r.db.QueryRow(ctx, `SELECT role FROM admins WHERE user_id = $1`, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return role, err
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
		       COALESCE(u.telegram_id, 0), u.display_name, u.username
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

// SuperAdminExists — проверяет, создан ли уже супер-админ через сидер.
func (r *adminRepo) SuperAdminExists(ctx context.Context) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM admins a
			JOIN admin_credentials ac ON ac.user_id = a.user_id
			WHERE a.role = 'super_admin'
		)
	`).Scan(&exists)
	return exists, err
}

// SeedSuperAdmin — создаёт системного пользователя, добавляет в admins и admin_credentials.
// Выполняется в одной транзакции; вызывается только при первом запуске.
func (r *adminRepo) SeedSuperAdmin(ctx context.Context, username, passwordHash, displayName string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO users (display_name, rating)
		VALUES ($1, 1000)
		RETURNING id
	`, displayName).Scan(&userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO admins (user_id, role) VALUES ($1, 'super_admin')
	`, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO admin_credentials (user_id, username, password_hash)
		VALUES ($1, $2, $3)
	`, userID, username, passwordHash)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetUsersWithRoles — все пользователи с их ролью в одном JOIN-запросе.
func (r *adminRepo) GetUsersWithRoles(ctx context.Context, limit int) ([]*UserWithRole, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.display_name, u.rating, COALESCE(u.rank,''), u.team_power,
		       (COALESCE(u.telegram_id, 0) != 0) AS has_telegram,
		       COALESCE(a.role, '') AS admin_role
		FROM users u
		LEFT JOIN admins a ON a.user_id = u.id
		ORDER BY u.rating DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*UserWithRole
	for rows.Next() {
		u := &UserWithRole{}
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.Rating, &u.Rank, &u.TeamPower, &u.HasTelegram, &u.AdminRole); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// GetAdminCredential — возвращает учётные данные администратора по имени пользователя.
func (r *adminRepo) GetAdminCredential(ctx context.Context, username string) (*AdminCredential, error) {
	cred := &AdminCredential{}
	err := r.db.QueryRow(ctx, `
		SELECT ac.user_id, ac.password_hash
		FROM admin_credentials ac
		JOIN admins a ON a.user_id = ac.user_id
		WHERE ac.username = $1
	`, username).Scan(&cred.UserID, &cred.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return cred, err
}
