package auth

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/shindakun/bskyarchive/internal/models"
)

const (
	sessionName      = "bskyarchive-session"
	sessionKeyUserID = "user_id"
	sessionKeyDID    = "did"
)

// SessionManager handles session operations
type SessionManager struct {
	store *sessions.CookieStore
	db    *sql.DB
}

// InitSessions creates a new session manager with 7-day expiration and HTTP-only cookies
func InitSessions(secret string, maxAge int, secure bool, sameSite http.SameSite, db *sql.DB) *SessionManager {
	store := sessions.NewCookieStore([]byte(secret))

	// Configure session options
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true, // Prevent JavaScript access
		Secure:   secure,
		SameSite: sameSite,
	}

	return &SessionManager{
		store: store,
		db:    db,
	}
}

// SaveSession stores a new session in the database and cookie
// accessToken parameter now stores the bskyoauth session ID
// refreshToken parameter is ignored (kept for compatibility)
func (sm *SessionManager) SaveSession(w http.ResponseWriter, r *http.Request, did, handle, displayName, bskyoauthSessionID, _ string) error {
	// Create session model
	session := &models.Session{
		ID:           uuid.New().String(),
		DID:          did,
		Handle:       handle,
		DisplayName:  displayName,
		AccessToken:  bskyoauthSessionID, // Store bskyoauth session ID
		RefreshToken: "",                  // Not used anymore
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour), // 30 days like example
		CreatedAt:    time.Now(),
	}

	// Validate session (skip AccessToken check since it's now a session ID)
	if session.ID == "" {
		return fmt.Errorf("session id is required")
	}
	if session.DID == "" {
		return fmt.Errorf("did is required")
	}

	// Save to database
	query := `
		INSERT INTO sessions (id, did, handle, display_name, access_token, refresh_token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(did) DO UPDATE SET
			handle = excluded.handle,
			display_name = excluded.display_name,
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			expires_at = excluded.expires_at
	`

	_, err := sm.db.Exec(query,
		session.ID,
		session.DID,
		session.Handle,
		session.DisplayName,
		session.AccessToken, // bskyoauth session ID
		session.RefreshToken,
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save session to database: %w", err)
	}

	// Save to cookie
	cookieSession, err := sm.store.Get(r, sessionName)
	if err != nil {
		return fmt.Errorf("failed to get cookie session: %w", err)
	}

	cookieSession.Values[sessionKeyUserID] = session.ID
	cookieSession.Values[sessionKeyDID] = session.DID

	if err := cookieSession.Save(r, w); err != nil {
		return fmt.Errorf("failed to save cookie session: %w", err)
	}

	return nil
}

// GetSession retrieves session data from cookie and database
func (sm *SessionManager) GetSession(r *http.Request) (*models.Session, error) {
	// Get session from cookie
	cookieSession, err := sm.store.Get(r, sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookie session: %w", err)
	}

	// Check if session exists in cookie
	userID, ok := cookieSession.Values[sessionKeyUserID].(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("no session found in cookie")
	}

	// Retrieve from database
	var session models.Session
	query := `
		SELECT id, did, handle, display_name, access_token, refresh_token, expires_at, created_at
		FROM sessions
		WHERE id = ?
	`

	err = sm.db.QueryRow(query, userID).Scan(
		&session.ID,
		&session.DID,
		&session.Handle,
		&session.DisplayName,
		&session.AccessToken,
		&session.RefreshToken,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found in database")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session from database: %w", err)
	}

	// Check if session is expired
	if session.IsExpired() {
		// Clean up expired session
		sm.ClearSession(nil, r)
		return nil, fmt.Errorf("session has expired")
	}

	return &session, nil
}

// ClearSession removes session from cookie and database (logout)
func (sm *SessionManager) ClearSession(w http.ResponseWriter, r *http.Request) error {
	// Get session from cookie
	cookieSession, err := sm.store.Get(r, sessionName)
	if err != nil {
		// If we can't get the session, it might already be cleared
		return nil
	}

	// Get user ID from cookie
	userID, ok := cookieSession.Values[sessionKeyUserID].(string)
	if ok && userID != "" {
		// Delete from database
		query := `DELETE FROM sessions WHERE id = ?`
		_, err := sm.db.Exec(query, userID)
		if err != nil {
			return fmt.Errorf("failed to delete session from database: %w", err)
		}
	}

	// Clear cookie session if writer is provided
	if w != nil {
		cookieSession.Options.MaxAge = -1
		if err := cookieSession.Save(r, w); err != nil {
			return fmt.Errorf("failed to clear cookie session: %w", err)
		}
	}

	return nil
}

// GetSessionFromContext retrieves session from request context
func GetSessionFromContext(ctx context.Context) (*models.Session, bool) {
	session, ok := ctx.Value("session").(*models.Session)
	return session, ok
}

// SetSessionInContext stores session in request context
func SetSessionInContext(ctx context.Context, session *models.Session) context.Context {
	return context.WithValue(ctx, "session", session)
}

// UpdateAccessToken updates the access and refresh tokens for a session
func (sm *SessionManager) UpdateAccessToken(did, accessToken, refreshToken string) error {
	query := `
		UPDATE sessions
		SET access_token = ?, refresh_token = ?, expires_at = ?
		WHERE did = ?
	`

	// Extend expiration by 7 days when refreshing token
	newExpiry := time.Now().Add(7 * 24 * time.Hour)

	result, err := sm.db.Exec(query, accessToken, refreshToken, newExpiry, did)
	if err != nil {
		return fmt.Errorf("failed to update session tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no session found for DID: %s", did)
	}

	return nil
}
