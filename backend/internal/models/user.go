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

// CreateUserRequest описывает тело запроса на создание пользователя.
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
	IsActive bool   `json:"is_active"`
}

// UpdateUserRequest описывает запрос на изменение роли и активности.
type UpdateUserRequest struct {
	Role     Role `json:"role"`
	IsActive bool `json:"is_active"`
}

// UpdatePasswordRequest описывает смену пароля пользователя администратором.
type UpdatePasswordRequest struct {
	Password string `json:"password"`
}

// UserFilter задает параметры фильтрации списка пользователей.
type UserFilter struct {
	Query    string
	Role     Role
	IsActive *bool
	Offset   int
	Limit    int
}

// UserListResponse возвращает список пользователей и общее количество для пагинации.
type UserListResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}
