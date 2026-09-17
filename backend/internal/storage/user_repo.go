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

// GetByID ищет пользователя по его идентификатору.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.pool.Reader.GetContext(ctx, &user, `SELECT * FROM users WHERE id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

// ListUsers возвращает список пользователей с фильтрацией и пагинацией.
func (r *UserRepository) ListUsers(ctx context.Context, filter models.UserFilter) (*models.UserListResponse, error) {
	var whereConditions []string
	var args []any

	if filter.Query != "" {
		whereConditions = append(whereConditions, "username LIKE ?")
		args = append(args, "%"+filter.Query+"%")
	}
	if filter.Role != "" {
		whereConditions = append(whereConditions, "role = ?")
		args = append(args, string(filter.Role))
	}
	if filter.IsActive != nil {
		whereConditions = append(whereConditions, "is_active = ?")
		args = append(args, *filter.IsActive)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE "
		for i, cond := range whereConditions {
			if i > 0 {
				whereClause += " AND "
			}
			whereClause += cond
		}
	}

	// Подсчет общего количества
	countQuery := "SELECT COUNT(*) FROM users " + whereClause
	var total int
	err := r.pool.Reader.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	// Выборка пользователей
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	selectQuery := "SELECT * FROM users " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)

	var users []models.User
	err = r.pool.Reader.SelectContext(ctx, &users, selectQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}

	if users == nil {
		users = []models.User{}
	}

	return &models.UserListResponse{
		Users: users,
		Total: total,
	}, nil
}

// UpdateUser обновляет роль и статус активности пользователя.
func (r *UserRepository) UpdateUser(ctx context.Context, id string, role models.Role, isActive bool) error {
	res, err := r.pool.Writer.ExecContext(ctx, `
		UPDATE users SET role = ?, is_active = ? WHERE id = ?
	`, role, isActive, id)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdatePassword изменяет пароль пользователя на новый.
func (r *UserRepository) UpdatePassword(ctx context.Context, id string, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	res, err := r.pool.Writer.ExecContext(ctx, `
		UPDATE users SET password_hash = ? WHERE id = ?
	`, string(hash), id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteUser удаляет пользователя по его идентификатору.
func (r *UserRepository) DeleteUser(ctx context.Context, id string) error {
	res, err := r.pool.Writer.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

