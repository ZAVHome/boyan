package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"boyan/internal/models"
)

func TestUserRepository_CRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "user_test_*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	ctx := context.Background()
	pool, err := NewSQLitePool(ctx, dbPath, 5000, 10000)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer pool.Close()

	if err := RunMigrations(ctx, pool.Writer); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	repo := NewUserRepository(pool)


	// 1. CreateUser
	user, err := repo.CreateUser(ctx, "testadmin", "secret123", models.RoleAdmin)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == "" || user.Username != "testadmin" || user.Role != models.RoleAdmin {
		t.Errorf("unexpected user: %+v", user)
	}

	// 2. VerifyPassword
	if !repo.VerifyPassword(user, "secret123") {
		t.Errorf("expected password match")
	}
	if repo.VerifyPassword(user, "wrongpass") {
		t.Errorf("expected password mismatch")
	}

	// 3. GetByID & GetByUsername
	byID, err := repo.GetByID(ctx, user.ID)
	if err != nil || byID == nil || byID.Username != "testadmin" {
		t.Fatalf("get by id: %v, user: %+v", err, byID)
	}

	byUsername, err := repo.GetByUsername(ctx, "testadmin")
	if err != nil || byUsername == nil || byUsername.ID != user.ID {
		t.Fatalf("get by username: %v, user: %+v", err, byUsername)
	}

	// 4. UpdateUser
	err = repo.UpdateUser(ctx, user.ID, models.RoleUser, false)
	if err != nil {
		t.Fatalf("update user: %v", err)
	}

	updated, err := repo.GetByID(ctx, user.ID)
	if err != nil || updated.Role != models.RoleUser || updated.IsActive != false {
		t.Fatalf("unexpected updated user: %+v", updated)
	}

	// 5. UpdatePassword
	err = repo.UpdatePassword(ctx, user.ID, "newsecret456")
	if err != nil {
		t.Fatalf("update password: %v", err)
	}

	updated, _ = repo.GetByID(ctx, user.ID)
	if !repo.VerifyPassword(updated, "newsecret456") {
		t.Errorf("expected new password to match")
	}

	// 6. ListUsers
	_, _ = repo.CreateUser(ctx, "alice", "alicepass", models.RoleUser)
	listResp, err := repo.ListUsers(ctx, models.UserFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if listResp.Total < 2 || len(listResp.Users) < 2 {
		t.Errorf("expected at least 2 users, got total=%d, count=%d", listResp.Total, len(listResp.Users))
	}

	// 7. DeleteUser
	err = repo.DeleteUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("delete user: %v", err)
	}

	deleted, err := repo.GetByID(ctx, user.ID)
	if err != nil || deleted != nil {
		t.Errorf("expected user to be deleted, got: %+v", deleted)
	}
}
