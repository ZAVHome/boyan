package auth

import (
	"context"
)

type contextKey string

const userContextKey contextKey = "boyan_user_claims"

// WithUser сохраняет UserClaims в контексте запроса.
func WithUser(ctx context.Context, claims *UserClaims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

// UserFromContext извлекает UserClaims из контекста запроса.
func UserFromContext(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(userContextKey).(*UserClaims)
	return claims, ok && claims != nil
}
