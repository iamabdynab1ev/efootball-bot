package models

import "time"

// AdminRole — admin роли
type AdminRole string

const (
	RoleSuperAdmin AdminRole = "super_admin" 
	RoleAdmin      AdminRole = "admin"       
)

// Admin — adminlar жадвали
type Admin struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Role      AdminRole `db:"role"`
	CreatedAt time.Time `db:"created_at"`
	User      *User     `db:"-"`
}
