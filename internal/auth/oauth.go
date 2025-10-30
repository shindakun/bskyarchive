package auth

import (
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

// HandleOAuthCallback completes the OAuth flow
func (om *OAuthManager) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Check for OAuth errors
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("OAuth error: %s - %s", errMsg, errDesc), http.StatusBadRequest)
		return
	}

	// Get authorization code, state, and issuer
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	issuer := r.URL.Query().Get("iss")

	// Complete OAuth flow
	ctx := r.Context()
	bskySession, err := om.client.CompleteAuthFlow(ctx, code, state, issuer)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to complete OAuth flow: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract handle from form or use DID as fallback
	// TODO: Fetch actual handle from Bluesky API in future iteration
	handle := r.FormValue("handle")
	if handle == "" {
		handle = bskySession.DID // Use DID as fallback
	}

	// Save authenticated session to our database
	err = om.sessionManager.SaveSession(
		w, r,
		bskySession.DID,
		handle,
		handle, // Use handle as display name initially
		bskySession.AccessToken,
		bskySession.RefreshToken,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save session: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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
