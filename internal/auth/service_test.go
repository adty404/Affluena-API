package auth

import "testing"

func TestHashPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password check to accept the original password")
	}

	if CheckPassword(hash, "wrong password") {
		t.Fatal("expected password check to reject a different password")
	}
}

func TestNormalizeEmail(t *testing.T) {
	got := NormalizeEmail("  USER@Example.COM ")
	if got != "user@example.com" {
		t.Fatalf("expected normalized email user@example.com, got %q", got)
	}
}
