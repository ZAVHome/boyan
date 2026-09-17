package models

import (
	"time"
)

// Role определяет уровень привилегий пользователя.
type Role string

const (
	RoleAdmin      Role = "admin"
	RoleUser       Role = "user"
	RoleRestricted Role = "restricted"
)

// User представляет пользователя системы.
type User struct {
	ID           string    `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         Role      `db:"role" json:"role"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
