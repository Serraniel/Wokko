package auth

import (
	"testing"
	"time"
)

func TestPassword(t *testing.T) {
	h, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := CheckPassword(h, "correct horse"); err != nil || !ok {
		t.Fatalf("correct password rejected: %v", err)
	}
	if ok, err := CheckPassword(h, "wrong"); err != nil || ok {
		t.Fatalf("wrong password accepted: %v", err)
	}
	if _, err := CheckPassword("bcrypt$nope", "x"); err == nil {
		t.Fatal("unknown format must be an error")
	}
}

func TestLimiter(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewLimiter(2, time.Minute)
	l.now = func() time.Time { return now }
	if !l.Allow("a") || !l.Allow("a") {
		t.Fatal("first attempts must pass")
	}
	if l.Allow("a") {
		t.Fatal("third attempt must be blocked")
	}
	if !l.Allow("b") {
		t.Fatal("other keys are independent")
	}
	now = now.Add(2 * time.Minute)
	if !l.Allow("a") {
		t.Fatal("window must reset")
	}
}
