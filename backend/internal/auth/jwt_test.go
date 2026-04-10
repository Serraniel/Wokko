package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/serraniel/wokko/backend/internal/auth"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	secret := "test-secret-key"
	userID := uuid.New()
	username := "testuser"
	role := "user"

	token, err := auth.GenerateAccessToken(secret, userID, username, role)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := auth.ValidateAccessToken(secret, token)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, claims.UserID)
	}
	if claims.Username != username {
		t.Errorf("expected Username %v, got %v", username, claims.Username)
	}
	if claims.Role != role {
		t.Errorf("expected Role %v, got %v", role, claims.Role)
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	token, err := auth.GenerateAccessToken("correct-secret", uuid.New(), "user", "user")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = auth.ValidateAccessToken("wrong-secret", token)
	if err == nil {
		t.Fatal("expected error with wrong secret, got nil")
	}
}

func TestValidateAccessToken_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	// Build an already-expired token manually
	claims := &auth.Claims{
		UserID:   userID,
		Username: "expireduser",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = auth.ValidateAccessToken(secret, tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateAccessToken_Malformed(t *testing.T) {
	_, err := auth.ValidateAccessToken("secret", "not.a.valid.token")
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}
