package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"boyan/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository управляет учетными записями пользователей.
type UserRepository struct {
	pool *DBPool
}

func NewUserRepository(pool *DBPool) *UserRepository {
	return &UserRepository{pool: pool}
}

// GetByUsername ищет пользователя по логину.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.pool.Reader.GetContext(ctx, &user, `SELECT * FROM users WHERE username = ?`, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &user, nil
}

// CreateUser создает нового пользователя с захэшированным паролем.
func (r *UserRepository) CreateUser(ctx context.Context, username, password string, role models.Role) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &models.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}

	_, err = r.pool.Writer.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash, role, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, user.ID, user.Username, user.PasswordHash, user.Role, user.IsActive, user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

// VerifyPassword проверяет правильность пароля пользователя.
func (r *UserRepository) VerifyPassword(user *models.User, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil
}

// EnsureAdminUser создает учетную запись администратора по умолчанию, если пользователей еще нет.
func (r *UserRepository) EnsureAdminUser(ctx context.Context, username, password string) error {
	var count int
	err := r.pool.Reader.GetContext(ctx, &count, `SELECT COUNT(*) FROM users WHERE role = 'admin'`)
	if err != nil {
		return fmt.Errorf("check admin count: %w", err)
	}

	if count == 0 {
		slog.Info("No admin users found. Creating default admin user...", "username", username)
		_, err := r.CreateUser(ctx, username, password, models.RoleAdmin)
		if err != nil {
			return fmt.Errorf("create default admin: %w", err)
		}
		slog.Info("Default admin user created successfully", "username", username)
	}
	return nil
}
