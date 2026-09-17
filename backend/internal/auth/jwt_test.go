package auth

import (
	"context"
	"testing"
	"time"

	"boyan/internal/models"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	secret := "test-secret-key-1234567890-test"
	user := &models.User{
		ID:       "user-123",
		Username: "librarian",
		Role:     models.RoleAdmin,
		IsActive: true,
	}

	token, expiresAt, err := GenerateToken(user, secret, 2)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.Before(time.Now().UTC()) {
		t.Fatal("expected expiresAt in the future")
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("unexpected error validating token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected UserID %s, got %s", user.ID, claims.UserID)
	}
	if claims.Username != user.Username {
		t.Errorf("expected Username %s, got %s", user.Username, claims.Username)
	}
	if claims.Role != models.RoleAdmin {
		t.Errorf("expected Role %s, got %s", models.RoleAdmin, claims.Role)
	}

	// Проверка с неверным секретом
	_, err = ValidateToken(token, "wrong-secret-key")
	if err == nil {
		t.Fatal("expected error with wrong secret, got nil")
	}
}

func TestContextHelpers(t *testing.T) {
	claims := &UserClaims{
		UserID:   "u-456",
		Username: "reader",
		Role:     models.RoleUser,
	}

	ctx := WithUser(context.Background(), claims)
	retrieved, ok := UserFromContext(ctx)
	if !ok || retrieved == nil {
		t.Fatal("expected claims in context")
	}
	if retrieved.UserID != "u-456" {
		t.Errorf("expected UserID u-456, got %s", retrieved.UserID)
	}

	// Пустой контекст
	emptyClaims, ok := UserFromContext(context.Background())
	if ok || emptyClaims != nil {
		t.Fatal("expected no claims in empty context")
	}
}
