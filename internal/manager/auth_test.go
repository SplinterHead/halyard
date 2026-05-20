package manager

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTokenLifecycle(t *testing.T) {
	// Generate random secret
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatalf("failed to generate random secret: %v", err)
	}

	username := "lewis.england"

	// 1. Generate token
	token, err := GenerateToken(username, secret)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// 2. Validate token
	validatedUser, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken failed on valid token: %v", err)
	}
	if validatedUser != username {
		t.Errorf("expected username %q, got %q", username, validatedUser)
	}

	// 3. Test invalid signature
	corruptedToken := token + "corrupted"
	_, err = ValidateToken(corruptedToken, secret)
	if err == nil {
		t.Error("expected error for corrupted token, got nil")
	}

	// 4. Test expired token
	t.Run("ExpiredToken", func(t *testing.T) {
		usernameB64 := base64.RawURLEncoding.EncodeToString([]byte(username))
		pastExpiryStr := strconv.FormatInt(time.Now().Add(-10*time.Second).Unix(), 10)
		payload := usernameB64 + "." + pastExpiryStr
		
		// Sign payload manually
		mac := hmac.New(sha256.New, secret)
		mac.Write([]byte(payload))
		sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
		manuallyExpiredToken := payload + "." + sig
		
		_, err = ValidateToken(manuallyExpiredToken, secret)
		if err == nil {
			t.Error("expected error for expired token, got nil")
		} else if !strings.Contains(err.Error(), "expired") {
			t.Errorf("expected expiry error, got: %v", err)
		}
	})

	// 5. Test invalid secret
	wrongSecret := make([]byte, 32)
	wrongSecret[0] = 0xFF // make it different
	_, err = ValidateToken(token, wrongSecret)
	if err == nil {
		t.Error("expected error when validating with wrong secret, got nil")
	}
}
