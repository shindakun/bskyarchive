package models

import (
	"fmt"
	"strings"
	"time"
)

// Session represents an authenticated user's session with OAuth tokens and identity information
type Session struct {
	ID           string    `json:"id"`
	DID          string    `json:"did"`           // Decentralized Identifier
	Handle       string    `json:"handle"`        // Bluesky handle (e.g., "user.bsky.social")
	DisplayName  string    `json:"display_name"`  // Optional display name
	AccessToken  string    `json:"-"`             // Never serialize to JSON
	RefreshToken string    `json:"-"`             // Never serialize to JSON
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// SessionState represents the current state of a session
type SessionState string

const (
	SessionStateActive  SessionState = "active"
	SessionStateExpired SessionState = "expired"
	SessionStateRevoked SessionState = "revoked"
)

// Validate checks if the session fields are valid
func (s *Session) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("session id is required")
	}

	if s.DID == "" {
		return fmt.Errorf("did is required")
	}

	if !strings.HasPrefix(s.DID, "did:plc:") && !strings.HasPrefix(s.DID, "did:web:") {
		return fmt.Errorf("did must start with 'did:plc:' or 'did:web:'")
	}

	if s.Handle == "" {
		return fmt.Errorf("handle is required")
	}

	if s.AccessToken == "" {
		return fmt.Errorf("access_token is required")
	}

	if s.ExpiresAt.IsZero() {
		return fmt.Errorf("expires_at is required")
	}

	if s.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("expires_at must be a future timestamp")
	}

	return nil
}

// State returns the current state of the session
func (s *Session) State() SessionState {
	if time.Now().After(s.ExpiresAt) || time.Now().Equal(s.ExpiresAt) {
		return SessionStateExpired
	}
	return SessionStateActive
}

// IsActive returns true if the session is currently active
func (s *Session) IsActive() bool {
	return s.State() == SessionStateActive
}

// IsExpired returns true if the session has expired
func (s *Session) IsExpired() bool {
	return s.State() == SessionStateExpired
}
