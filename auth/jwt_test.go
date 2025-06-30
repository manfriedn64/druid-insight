package auth

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateAndExtractJWT(t *testing.T) {
	secret := "test_secret"
	username := "alice"
	isAdmin := true
	expiration := 10

	// Generate a JWT
	token, err := GenerateJWT(secret, username, isAdmin, expiration)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	// Create a request with the token in Authorization header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Extract user and admin from JWT
	gotUser, gotAdmin, err := ExtractUserAndAdminFromJWT(req, secret)
	if err != nil {
		t.Fatalf("ExtractUserAndAdminFromJWT failed: %v", err)
	}
	if gotUser != username {
		t.Errorf("Expected username %q, got %q", username, gotUser)
	}
	if gotAdmin != isAdmin {
		t.Errorf("Expected isAdmin %v, got %v", isAdmin, gotAdmin)
	}
}

func TestExtractUserAndAdminFromJWT_InvalidToken(t *testing.T) {
	secret := "test_secret"
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken")

	_, _, err := ExtractUserAndAdminFromJWT(req, secret)
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}

func TestExtractUserAndAdminFromJWT_NoHeader(t *testing.T) {
	secret := "test_secret"
	req := httptest.NewRequest("GET", "/", nil)

	_, _, err := ExtractUserAndAdminFromJWT(req, secret)
	if err == nil {
		t.Error("Expected error for missing Authorization header, got nil")
	}
}

func TestGenerateJWT_Expiration(t *testing.T) {
	secret := "test_secret"
	username := "bob"
	isAdmin := false
	expiration := 0 // expires immediately

	token, err := GenerateJWT(secret, username, isAdmin, expiration)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Wait to ensure token is expired
	time.Sleep(2 * time.Second)

	_, _, err = ExtractUserAndAdminFromJWT(req, secret)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}
