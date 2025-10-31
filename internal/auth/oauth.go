package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/shindakun/bskyoauth"
)

// OAuthManager handles OAuth operations
type OAuthManager struct {
	client         *bskyoauth.Client
	sessionManager *SessionManager
}

// InitOAuth creates a new OAuth manager with baseURL and scopes
func InitOAuth(baseURL string, scopes []string, sessionManager *SessionManager) *OAuthManager {
	opts := bskyoauth.ClientOptions{
		BaseURL:         baseURL,
		ClientName:      "Bluesky Personal Archive Tool",
		ApplicationType: bskyoauth.ApplicationTypeWeb,
		Scopes:          scopes,
	}

	client := bskyoauth.NewClientWithOptions(opts)

	return &OAuthManager{
		client:         client,
		sessionManager: sessionManager,
	}
}

// HandleOAuthLogin initiates the OAuth flow by prompting for handle
func (om *OAuthManager) HandleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	// For now, use a simple form to get the handle
	// This will be replaced with a proper template in later tasks
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head><title>Login</title></head>
<body>
	<h1>Login with Bluesky</h1>
	<form method="POST">
		<label>Handle: <input type="text" name="handle" placeholder="user.bsky.social" required></label>
		<button type="submit">Login</button>
	</form>
</body>
</html>
		`))
		return
	}

	// POST: Start OAuth flow with handle
	handle := r.FormValue("handle")
	if handle == "" {
		http.Error(w, "Handle is required", http.StatusBadRequest)
		return
	}

	// Start OAuth flow
	ctx := r.Context()
	flowState, err := om.client.StartAuthFlow(ctx, handle)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start OAuth flow: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to authorization URL
	http.Redirect(w, r, flowState.AuthURL, http.StatusSeeOther)
}

// HandleOAuthCallback completes the OAuth flow using bskyoauth's built-in handler
func (om *OAuthManager) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Use bskyoauth's callback handler which stores the session with DPoP key/nonce
	handler := om.client.CallbackHandler(func(w http.ResponseWriter, r *http.Request, sessionID string) {
		// bskyoauth has stored the full session (including DPoP key/nonce)
		// Now we just need to link it to our app session

		// Get the bskyoauth session to extract user info
		bskySession, err := om.client.GetSession(sessionID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get session: %v", err), http.StatusInternalServerError)
			return
		}

		// Save session ID and user info to our database
		err = om.sessionManager.SaveSession(
			w, r,
			bskySession.DID,
			bskySession.DID, // Use DID as handle for now
			bskySession.DID, // Use DID as display name for now
			sessionID,        // Store the bskyoauth session ID
			"",              // No longer store tokens directly
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save session: %v", err), http.StatusInternalServerError)
			return
		}

		// Redirect to dashboard
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})

	handler(w, r)
}

// HandleLogout clears session and redirects to landing page
func (om *OAuthManager) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear session from database and cookie
	if err := om.sessionManager.ClearSession(w, r); err != nil {
		// Log error but continue with logout
		fmt.Printf("Error clearing session: %v\n", err)
	}

	// Redirect to landing page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ClientMetadataHandler returns the handler for OAuth client metadata
func (om *OAuthManager) ClientMetadataHandler() http.HandlerFunc {
	return om.client.ClientMetadataHandler()
}

// RefreshAccessToken refreshes an expired access token using the refresh token
func (om *OAuthManager) RefreshAccessToken(did, refreshToken string) (string, string, error) {
	ctx := context.Background()

	// Create a session object for the refresh call
	// Note: We only need DID and RefreshToken for the refresh operation
	session := &bskyoauth.Session{
		DID:          did,
		RefreshToken: refreshToken,
	}

	// Call the bskyoauth RefreshToken method
	newSession, err := om.client.RefreshToken(ctx, session)
	if err != nil {
		return "", "", fmt.Errorf("failed to refresh token: %w", err)
	}

	// Return the new tokens
	return newSession.AccessToken, newSession.RefreshToken, nil
}

// GetBskySession retrieves the bskyoauth session by session ID
func (om *OAuthManager) GetBskySession(sessionID string) (*bskyoauth.Session, error) {
	return om.client.GetSession(sessionID)
}
