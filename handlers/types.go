package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"sync"
	"time"
)

const (
	userKey contextKey = "user"
)

// User represents a user in the system
type User struct {
	ID         int
	Name       string
	CardNumber string
	Role       string
	Balance    float64
	Email      string
	CreatedAt  time.Time
}

type csrfToken struct {
	token     string
	expiresAt time.Time
}

var (
	csrfTokens = make(map[string]csrfToken)
	csrfMutex  sync.RWMutex
)

// generateCSRFToken generates a new CSRF token
func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)

	csrfMutex.Lock()
	defer csrfMutex.Unlock()

	// Clean up expired tokens
	now := time.Now()
	for k, v := range csrfTokens {
		if now.After(v.expiresAt) {
			delete(csrfTokens, k)
		}
	}

	// Store new token with 1 hour expiration
	csrfTokens[token] = csrfToken{
		token:     token,
		expiresAt: now.Add(1 * time.Hour),
	}

	return token
}

// verifyCSRFToken verifies a CSRF token
func verifyCSRFToken(token string) bool {
	if token == "" {
		return false
	}

	csrfMutex.RLock()
	storedToken, exists := csrfTokens[token]
	csrfMutex.RUnlock()

	if !exists || time.Now().After(storedToken.expiresAt) {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(token), []byte(storedToken.token)) == 1
}
