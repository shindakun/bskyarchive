package archiver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/shindakun/bskyoauth"
)

// ATProtoClient wraps the indigo XRPC client with DPoP authentication
type ATProtoClient struct {
	session *bskyoauth.Session
	client  *xrpc.Client
}

// NewATProtoClientFromSession creates a new AT Protocol client from a bskyoauth session
// This properly sets up DPoP transport for secure token usage
func NewATProtoClientFromSession(ctx context.Context, session *bskyoauth.Session) (*ATProtoClient, error) {
	// Resolve PDS endpoint for the user
	dir := identity.DefaultDirectory()
	atid, err := syntax.ParseAtIdentifier(session.DID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DID: %w", err)
	}

	ident, err := dir.Lookup(ctx, *atid)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup identity: %w", err)
	}

	pdsHost := ident.PDSEndpoint()

	// Create DPoP transport - this is critical for DPoP-bound tokens!
	transport := bskyoauth.NewDPoPTransport(
		http.DefaultTransport,
		session.DPoPKey,      // Private key for signing requests
		session.AccessToken,
		session.DPoPNonce,    // Nonce from server
	)

	httpClient := &http.Client{
		Transport: transport,
	}

	xrpcClient := &xrpc.Client{
		Host:   pdsHost,
		Client: httpClient,
	}

	return &ATProtoClient{
		session: session,
		client:  xrpcClient,
	}, nil
}

// GetClient returns the underlying XRPC client for direct use
func (c *ATProtoClient) GetClient() *xrpc.Client {
	return c.client
}

// GetSession returns the bskyoauth session
func (c *ATProtoClient) GetSession() *bskyoauth.Session {
	return c.session
}

// UpdateSession updates the session (e.g., after token refresh) and recreates the client
func (c *ATProtoClient) UpdateSession(ctx context.Context, newSession *bskyoauth.Session) error {
	// Recreate client with new session
	newClient, err := NewATProtoClientFromSession(ctx, newSession)
	if err != nil {
		return err
	}

	c.session = newClient.session
	c.client = newClient.client
	return nil
}

// Ping checks if the client can reach the server
func (c *ATProtoClient) Ping(ctx context.Context) error {
	// Simple health check
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s/xrpc/_health", c.client.Host), nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.client.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to ping server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned error: %d", resp.StatusCode)
	}

	return nil
}
