# Bluesky Personal Archive Tool

A local-first archival solution for Bluesky. Archive and search your Bluesky content on your own machine.

## Features

- **Privacy-first**: All data stored locally on your machine
- **Full-text search**: Find any post instantly with SQLite FTS5
- **Complete archive**: Posts, media, profiles, and engagement metrics
- **Fast & efficient**: Incremental updates and rate-limited operations

## Requirements

- Go 1.21 or higher
- SQLite (included via modernc.org/sqlite)

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/shindakun/bskyarchive.git
cd bskyarchive
```

### 2. Set up environment variables

You only need to set ONE environment variable:

```bash
# Generate a secure random session secret (must be at least 32 characters)
export SESSION_SECRET="your-secure-random-string-at-least-32-chars-long"

# Example: Generate a random secret
export SESSION_SECRET=$(openssl rand -base64 32)
```

**Important**: The `SESSION_SECRET` is used to encrypt session cookies. Keep it secret and don't commit it to version control.

### 3. Build and run

```bash
go build -o bskyarchive ./cmd/bskyarchive
./bskyarchive
```

The server will start on `http://localhost:8080`

### 4. First-time setup

1. Visit `http://localhost:8080`
2. Click "Sign in with Bluesky"
3. Enter your Bluesky handle (e.g., `user.bsky.social`)
4. Complete the OAuth flow
5. You'll be redirected to your dashboard

## Configuration

The application uses `config.yaml` for configuration. Default settings:

```yaml
server:
  port: 8080
  host: "localhost"
  # base_url: "https://your-domain.com"  # Required for OAuth to work with Bluesky

archive:
  db_path: "./data/archive.db"
  media_path: "./data/media"

oauth:
  scopes:
    - "atproto"
    - "transition:generic"
  session_max_age: 604800  # 7 days
```

### Setting the Base URL

**Important**: For OAuth to work with Bluesky, you need to set a publicly accessible `base_url`:

```yaml
server:
  base_url: "https://your-domain.com"
```

**For local development with a tunnel** (e.g., ngrok, cloudflared):

1. Start a tunnel: `ngrok http 8080`
2. Copy the public URL (e.g., `https://abc123.ngrok.io`)
3. Set it in `config.yaml`:
   ```yaml
   server:
     base_url: "https://abc123.ngrok.io"
   ```
4. Start the application

**Note**: Bluesky OAuth requires a publicly accessible URL - `localhost` won't work for the OAuth callback.

You can override the config file location:

```bash
CONFIG_PATH=/path/to/config.yaml ./bskyarchive
```

## OAuth Flow

This tool uses Bluesky's OAuth 2.0 with PKCE flow via the [bskyoauth](https://github.com/shindakun/bskyoauth) library.

**No client ID or client secret is required** - the OAuth flow uses your application's base URL (`http://localhost:8080`) as the client identifier. This is a simpler, more secure approach than traditional OAuth.

## Architecture

- **Language**: Go 1.21+
- **Database**: SQLite with FTS5 full-text search
- **Web**: chi router, html/template, HTMX, Pico CSS
- **Auth**: bskyoauth (Bluesky OAuth 2.0 with PKCE)
- **AT Protocol**: indigo SDK

## Project Structure

```
bskyarchive/
├── cmd/bskyarchive/      # Main application
├── internal/
│   ├── auth/             # OAuth & session management
│   ├── config/           # Configuration loading
│   ├── models/           # Data models
│   ├── storage/          # Database operations
│   └── web/
│       ├── handlers/     # HTTP handlers
│       ├── middleware/   # HTTP middleware
│       ├── templates/    # HTML templates
│       └── static/       # CSS, JS, images
├── specs/                # Feature specifications
└── config.yaml           # Configuration file
```

## Development

### Building

```bash
go build -o bskyarchive ./cmd/bskyarchive
```

### Testing

```bash
go test ./...
```

### Code Style

Follow standard Go conventions. Use `gofmt` for formatting.

## Security

- Sessions are stored with HTTP-only cookies (7-day expiration)
- All passwords and secrets should be set via environment variables
- Database uses write-ahead logging (WAL) for safe concurrent access
- OAuth uses PKCE (Proof Key for Code Exchange) for security

## Core Principles

1. **Data Privacy & Local-First**: All data stays on your machine
2. **Comprehensive & Accurate**: Complete archive of your content
3. **Multiple Export Formats**: JSON, CSV, and more (coming soon)
4. **Fast & Efficient**: Full-text search with SQLite FTS5
5. **Incremental Operations**: Only fetch new content on updates

## License

See LICENSE file for details.

## Author

Steve Layton
- GitHub: [@shindakun](https://github.com/shindakun)
- Bluesky: [@shindakun.net](https://bsky.app/profile/shindakun.net)
