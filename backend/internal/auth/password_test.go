package auth_test

import (
	"testing"

	"github.com/serraniel/wokko/backend/internal/auth"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "super-secret-password"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == password {
		t.Fatal("hash must not equal plain password")
	}

	if !auth.CheckPassword(hash, password) {
		t.Fatal("CheckPassword returned false for correct password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if auth.CheckPassword(hash, "wrong-password") {
		t.Fatal("CheckPassword returned true for wrong password")
	}
}

func TestHashPassword_DifferentHashEachTime(t *testing.T) {
	password := "same-password"
	hash1, err1 := auth.HashPassword(password)
	hash2, err2 := auth.HashPassword(password)
	if err1 != nil || err2 != nil {
		t.Fatal("HashPassword failed")
	}
	if hash1 == hash2 {
		t.Fatal("expected different hashes due to bcrypt salt, got same")
	}
	// Both should still verify correctly
	if !auth.CheckPassword(hash1, password) || !auth.CheckPassword(hash2, password) {
		t.Fatal("CheckPassword failed for one of the hashes")
	}
}
