package manager

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lewis-england/halyard/api"
	"golang.org/x/crypto/bcrypt"
)

// HasUsers checks if there are any users in the database
func (d *DB) HasUsers() (bool, error) {
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateUser creates the first user in the database
func (d *DB) CreateUser(username, realName, password string) (*api.User, error) {
	// Validate non-empty fields
	username = strings.TrimSpace(username)
	realName = strings.TrimSpace(realName)
	if username == "" || realName == "" || password == "" {
		return nil, errors.New("username, real name, and password cannot be empty")
	}

	// Enforce password requirements (e.g. min 8 chars)
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters long")
	}

	// Double-check if a user already exists (zero-trust enforcement)
	hasUsers, err := d.HasUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to check existing users: %w", err)
	}
	if hasUsers {
		return nil, errors.New("registration is disabled: a user already exists in this cluster")
	}

	// Securely hash the password using bcrypt with standard/default cost
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	id := uuid.New().String()
	now := time.Now()

	_, err = d.Exec(`INSERT INTO users (id, username, real_name, password_hash, created_at) 
		VALUES (?, ?, ?, ?, ?)`, id, username, realName, string(hashBytes), now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return &api.User{
		ID:        id,
		Username:  username,
		RealName:  realName,
		CreatedAt: now,
	}, nil
}

// AuthenticateUser verifies a user's credentials and returns the user details if valid
func (d *DB) AuthenticateUser(username, password string) (*api.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, errors.New("invalid username or password")
	}

	var user api.User
	var passwordHash string

	err := d.QueryRow("SELECT id, username, real_name, password_hash, created_at FROM users WHERE username = ?", username).
		Scan(&user.ID, &user.Username, &user.RealName, &passwordHash, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("invalid username or password")
	} else if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Verify the hash (timing-attack resilient bcrypt compare)
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	return &user, nil
}

// GenerateToken creates a signed stateless session token for a username
func GenerateToken(username string, secret []byte) (string, error) {
	// Base64 encode username to prevent separator conflicts
	usernameB64 := base64.RawURLEncoding.EncodeToString([]byte(username))
	
	// Expiration time set to 24 hours from now
	expiryStr := strconv.FormatInt(time.Now().Add(24*time.Hour).Unix(), 10)
	
	payload := usernameB64 + "." + expiryStr
	
	// Generate signature
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	
	return payload + "." + signature, nil
}

// ValidateToken verifies token signature and expiration, returning the username if valid
func ValidateToken(tokenStr string, secret []byte) (string, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token format")
	}
	
	usernameB64 := parts[0]
	expiryStr := parts[1]
	tokenSig := parts[2]
	
	payload := usernameB64 + "." + expiryStr
	
	// Calculate expected signature
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	
	// Timing attack safe comparison
	if !hmac.Equal([]byte(tokenSig), []byte(expectedSig)) {
		return "", errors.New("invalid signature")
	}
	
	// Check expiration
	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return "", errors.New("invalid expiration time format")
	}
	
	if time.Now().Unix() > expiry {
		return "", errors.New("token has expired")
	}
	
	// Decode username
	usernameBytes, err := base64.RawURLEncoding.DecodeString(usernameB64)
	if err != nil {
		return "", errors.New("invalid encoded username")
	}
	
	return string(usernameBytes), nil
}
