# HTTP API Contracts: Web Interface

**Phase 1 Output** | **Date**: 2025-10-30 | **Plan**: [../plan.md](../plan.md)

## Overview

This document defines all HTTP routes, request/response formats, status codes, and error handling for the Bluesky Personal Archive Tool web interface. All endpoints follow RESTful conventions where applicable.

---

## Base Configuration

**Base URL**: `http://localhost:8080` (configurable)
**Content Type**: `text/html` for pages, `application/json` for API endpoints
**Authentication**: Cookie-based sessions (gorilla/sessions)
**CSRF Protection**: Required for all POST/PUT/DELETE requests

---

## 1. Public Routes

Routes accessible without authentication.

### `GET /`

Landing page for unauthenticated users.

**Auth Required**: No

**Response**: HTML page

**HTTP Status**:
- `200 OK`: Page rendered successfully
- `302 Found`: Redirect to `/dashboard` if already authenticated

**Template Data**:
```go
type LandingData struct {
    Error   string // Optional error message from query param
    Message string // Optional info message
}
```

**Query Parameters**:
- `error`: Optional error code (`auth_required`, `oauth_failed`, etc.)
- `message`: Optional info message

**Example**:
```http
GET / HTTP/1.1
Host: localhost:8080

HTTP/1.1 200 OK
Content-Type: text/html

<!DOCTYPE html>
<html>
  <head><title>Bluesky Archive Tool</title></head>
  <body>
    <h1>Archive Your Bluesky Account</h1>
    <a href="/auth/login">Login with Bluesky</a>
  </body>
</html>
```

---

### `GET /about`

About page with project information and links.

**Auth Required**: No

**Response**: HTML page

**HTTP Status**:
- `200 OK`: Page rendered successfully

**Template Data**:
```go
type AboutData struct {
    Version       string
    AuthorBsky    string // Bluesky handle
    GithubRepo    string // GitHub repository URL
    Authenticated bool   // Whether user is logged in
}
```

**Example**:
```http
GET /about HTTP/1.1
Host: localhost:8080

HTTP/1.1 200 OK
Content-Type: text/html

<!DOCTYPE html>
<html>
  <head><title>About - Bluesky Archive Tool</title></head>
  <body>
    <h1>About This Tool</h1>
    <p>Created by <a href="https://bsky.app/profile/author.bsky.social">@author.bsky.social</a></p>
    <p><a href="https://github.com/shindakun/bskyarchive">View on GitHub</a></p>
  </body>
</html>
```

---

## 2. Authentication Routes

OAuth 2.0 flow with Bluesky.

### `GET /auth/login`

Initiates OAuth login flow with Bluesky.

**Auth Required**: No

**Response**: HTTP redirect

**HTTP Status**:
- `302 Found`: Redirect to Bluesky OAuth authorization URL

**Side Effects**:
- Creates temporary OAuth session with state and code_verifier
- Sets session cookie

**Example**:
```http
GET /auth/login HTTP/1.1
Host: localhost:8080

HTTP/1.1 302 Found
Location: https://bsky.social/oauth/authorize?client_id=...&state=...&code_challenge=...
Set-Cookie: oauth_session=...; Path=/; HttpOnly; SameSite=Lax
```

---

### `GET /auth/callback`

OAuth callback endpoint (called by Bluesky after authorization).

**Auth Required**: No

**Response**: HTTP redirect

**Query Parameters**:
- `code`: Required, OAuth authorization code
- `state`: Required, CSRF protection state value

**HTTP Status**:
- `302 Found`: Redirect to `/dashboard` on success
- `302 Found`: Redirect to `/?error=oauth_failed` on failure

**Side Effects**:
- Exchanges authorization code for access token
- Creates authenticated session
- Stores DID, handle, and tokens in session

**Example Success**:
```http
GET /auth/callback?code=abc123&state=xyz789 HTTP/1.1
Host: localhost:8080

HTTP/1.1 302 Found
Location: /dashboard
Set-Cookie: auth_session=...; Path=/; HttpOnly; SameSite=Lax; Max-Age=604800
```

**Example Failure**:
```http
GET /auth/callback?error=access_denied HTTP/1.1
Host: localhost:8080

HTTP/1.1 302 Found
Location: /?error=oauth_failed
```

---

### `GET /auth/logout`

Logs out the current user.

**Auth Required**: Yes

**Response**: HTTP redirect

**HTTP Status**:
- `302 Found`: Redirect to `/` after clearing session

**Side Effects**:
- Destroys authenticated session
- Clears session cookies

**Example**:
```http
GET /auth/logout HTTP/1.1
Host: localhost:8080
Cookie: auth_session=...

HTTP/1.1 302 Found
Location: /
Set-Cookie: auth_session=; Path=/; HttpOnly; Max-Age=0
```

---

## 3. Protected Routes (Authenticated)

All routes below require valid authenticated session.

### `GET /dashboard`

Main dashboard showing archive status and quick actions.

**Auth Required**: Yes

**Response**: HTML page

**HTTP Status**:
- `200 OK`: Dashboard rendered
- `302 Found`: Redirect to `/?error=auth_required` if not authenticated

**Template Data**:
```go
type DashboardData struct {
    User    UserInfo      // Current user info
    Status  ArchiveStatus // Archive status (posts, media, last sync)
    Profile Profile       // Latest profile snapshot
}

type UserInfo struct {
    DID    string
    Handle string
    DisplayName string
}
```

**Example**:
```http
GET /dashboard HTTP/1.1
Host: localhost:8080
Cookie: auth_session=...

HTTP/1.1 200 OK
Content-Type: text/html

<!DOCTYPE html>
<html>
  <head><title>Dashboard - Bluesky Archive</title></head>
  <body>
    <h1>Welcome, @user.bsky.social</h1>
    <p>Total Posts: 1,234</p>
    <p>Last Sync: 2025-10-29 14:30:00</p>
    <a href="/archive">Manage Archive</a>
  </body>
</html>
```

---

### `GET /archive`

Archive management page for initiating and monitoring sync operations.

**Auth Required**: Yes

**Response**: HTML page

**HTTP Status**:
- `200 OK`: Page rendered
- `302 Found`: Redirect to login if not authenticated

**Template Data**:
```go
type ArchivePageData struct {
    User              UserInfo
    Status            ArchiveStatus
    ActiveOperation   *ArchiveOperation // nil if no active operation
    RecentOperations  []ArchiveOperation
}
```

**Example**:
```http
GET /archive HTTP/1.1
Host: localhost:8080
Cookie: auth_session=...

HTTP/1.1 200 OK
Content-Type: text/html

<!DOCTYPE html>
<html>
  <head><title>Archive Management - Bluesky Archive</title></head>
  <body>
    <h1>Archive Management</h1>
    <button hx-post="/archive/start" hx-target="#status">Start Full Sync</button>
    <div id="status" hx-get="/archive/status" hx-trigger="every 2s">
      <!-- Archive status loaded via HTMX -->
    </div>
  </body>
</html>
```

---

### `POST /archive/start`

Initiates a new archive operation.

**Auth Required**: Yes

**Content-Type**: `application/x-www-form-urlencoded` or `multipart/form-data`

**Request Body**:
```
operation_type=full_sync
```
or
```
operation_type=incremental_sync
```

**Response**: HTML fragment (HTMX target) or JSON

**HTTP Status**:
- `200 OK`: Operation started, returns status HTML
- `400 Bad Request`: Invalid operation type or operation already running
- `500 Internal Server Error`: Failed to start operation

**Success Response** (HTML fragment):
```html
<div id="archive-status">
  <progress value="0" max="100"></progress>
  <p>Starting archive operation...</p>
</div>
```

**Success Response** (JSON, if `Accept: application/json`):
```json
{
  "operation_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "message": "Archive operation started successfully"
}
```

**Error Response**:
```json
{
  "error": "operation_already_running",
  "message": "An archive operation is already in progress"
}
```

---

### `GET /archive/status`

Polls current archive operation status.

**Auth Required**: Yes

**Response**: HTML fragment (HTMX target) or JSON

**HTTP Status**:
- `200 OK`: Status returned
- `404 Not Found`: No active operation

**Success Response** (HTML fragment):
```html
<div id="archive-status">
  <progress value="750" max="1000"></progress>
  <p>Archived 750 of 1,000 posts (75%)</p>
  <p>Downloading media... 45 files remaining</p>
</div>
```

**Success Response** (JSON):
```json
{
  "operation_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "operation_type": "full_sync",
  "progress_current": 750,
  "progress_total": 1000,
  "progress_percent": 75,
  "started_at": "2025-10-30T10:00:00Z"
}
```

**No Active Operation** (HTML fragment):
```html
<div id="archive-status">
  <p>No active archive operation</p>
  <button hx-post="/archive/start" hx-vals='{"operation_type": "full_sync"}'>
    Start Full Sync
  </button>
</div>
```

---

### `GET /browse`

Browse archived posts with pagination and search.

**Auth Required**: Yes

**Response**: HTML page

**Query Parameters**:
- `page`: Optional, default 1
- `page_size`: Optional, default 20
- `q`: Optional search query
- `filter`: Optional, `media` | `replies` | `all` (default)

**HTTP Status**:
- `200 OK`: Posts page rendered

**Template Data**:
```go
type BrowsePageData struct {
    User         UserInfo
    Posts        []Post
    Page         int
    PageSize     int
    TotalCount   int
    TotalPages   int
    HasNext      bool
    HasPrev      bool
    Query        string // If search active
    Filter       string
}
```

**Example**:
```http
GET /browse?page=2&page_size=20 HTTP/1.1
Host: localhost:8080
Cookie: auth_session=...

HTTP/1.1 200 OK
Content-Type: text/html

<!DOCTYPE html>
<html>
  <head><title>Browse Archive - Bluesky Archive</title></head>
  <body>
    <h1>Your Archive</h1>
    <form hx-get="/browse" hx-target="#posts" hx-push-url="true">
      <input name="q" placeholder="Search posts...">
      <button type="submit">Search</button>
    </form>
    <div id="posts">
      <!-- Post cards rendered here -->
      <article class="post-card">
        <p>Post text...</p>
        <time>2025-10-29 14:30</time>
      </article>
    </div>
    <nav>
      <a href="/browse?page=1">Previous</a>
      <a href="/browse?page=3">Next</a>
    </nav>
  </body>
</html>
```

---

### `GET /browse/posts/:uri`

View a single post with full details (HTMX partial or full page).

**Auth Required**: Yes

**Path Parameters**:
- `uri`: AT Protocol post URI (URL-encoded)

**Response**: HTML fragment or full page

**HTTP Status**:
- `200 OK`: Post rendered
- `404 Not Found`: Post not found in archive

**Template Data**:
```go
type PostDetailData struct {
    User   UserInfo
    Post   Post
    Media  []Media
    Thread []Post // If post is part of a thread
}
```

**Example**:
```http
GET /browse/posts/at%3A%2F%2Fdid%3Aplc%3Aabc123%2Fapp.bsky.feed.post%2Fxyz789 HTTP/1.1
Host: localhost:8080
Cookie: auth_session=...

HTTP/1.1 200 OK
Content-Type: text/html

<article class="post-detail">
  <header>
    <img src="/media/2025/10/avatar.jpg" alt="Avatar">
    <h2>@user.bsky.social</h2>
    <time>2025-10-29 14:30:00</time>
  </header>
  <p>Post text goes here...</p>
  <div class="media">
    <img src="/media/2025/10/abc123.jpg" alt="Embedded image">
  </div>
  <footer>
    <span>❤️ 42 likes</span>
    <span>🔁 12 reposts</span>
    <span>💬 5 replies</span>
  </footer>
</article>
```

---

### `GET /api/search`

API endpoint for searching posts (JSON response).

**Auth Required**: Yes

**Response**: JSON

**Query Parameters**:
- `q`: Required, search query
- `page`: Optional, default 1
- `page_size`: Optional, default 20

**HTTP Status**:
- `200 OK`: Search results returned
- `400 Bad Request`: Missing or invalid query

**Success Response**:
```json
{
  "query": "bluesky",
  "posts": [
    {
      "uri": "at://did:plc:abc123/app.bsky.feed.post/xyz789",
      "text": "Loving Bluesky!",
      "created_at": "2025-10-29T14:30:00Z",
      "like_count": 42,
      "repost_count": 12,
      "reply_count": 5,
      "has_media": true
    }
  ],
  "page": 1,
  "page_size": 20,
  "total_count": 156,
  "has_more": true
}
```

---

## 4. Static Assets

### `GET /static/*`

Serves static files (CSS, JS, images).

**Auth Required**: No

**Response**: File content

**HTTP Status**:
- `200 OK`: File found and served
- `404 Not Found`: File does not exist

**Cache Headers**:
```
Cache-Control: public, max-age=31536000
```

**Example**:
```http
GET /static/css/pico.min.css HTTP/1.1
Host: localhost:8080

HTTP/1.1 200 OK
Content-Type: text/css
Cache-Control: public, max-age=31536000

/* Pico CSS content */
```

---

### `GET /media/*`

Serves archived media files (images, videos).

**Auth Required**: Yes (media belongs to authenticated user)

**Response**: File content

**HTTP Status**:
- `200 OK`: File found and served
- `403 Forbidden`: File belongs to different user
- `404 Not Found`: File does not exist

**Example**:
```http
GET /media/2025/10/abc123def456.jpg HTTP/1.1
Host: localhost:8080
Cookie: auth_session=...

HTTP/1.1 200 OK
Content-Type: image/jpeg
Content-Length: 245678

[binary image data]
```

---

## 5. Error Responses

### Standard Error Format (JSON)

```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {
    "field": "Additional context"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `auth_required` | 401 | User not authenticated |
| `forbidden` | 403 | User not authorized for resource |
| `not_found` | 404 | Resource does not exist |
| `operation_already_running` | 400 | Archive operation in progress |
| `invalid_operation_type` | 400 | Unknown operation type |
| `oauth_failed` | 400 | OAuth authentication failed |
| `rate_limit_exceeded` | 429 | Too many requests |
| `internal_error` | 500 | Unexpected server error |

### HTML Error Pages

For browser requests (HTML Accept header), errors render custom error pages:

**401 Unauthorized**:
```html
<!DOCTYPE html>
<html>
  <head><title>Authentication Required</title></head>
  <body>
    <h1>Authentication Required</h1>
    <p>Please <a href="/auth/login">log in</a> to continue.</p>
  </body>
</html>
```

**404 Not Found**:
```html
<!DOCTYPE html>
<html>
  <head><title>Page Not Found</title></head>
  <body>
    <h1>Page Not Found</h1>
    <p>The page you're looking for doesn't exist.</p>
    <a href="/">Return to Home</a>
  </body>
</html>
```

**500 Internal Server Error**:
```html
<!DOCTYPE html>
<html>
  <head><title>Server Error</title></head>
  <body>
    <h1>Something Went Wrong</h1>
    <p>We're working on fixing the issue. Please try again later.</p>
  </body>
</html>
```

---

## 6. Middleware

### Authentication Middleware

Applied to all protected routes.

**Logic**:
1. Check for valid session cookie
2. Verify session is not expired
3. Extract user DID and handle
4. Add to request context

**Redirect**: Unauthorized requests redirect to `/?error=auth_required`

---

### CSRF Middleware

Applied to all POST/PUT/DELETE requests.

**Logic**:
1. Verify CSRF token in form data or header
2. Token must match session token

**Response**: `403 Forbidden` if token invalid or missing

**Token Locations**:
- Form field: `csrf_token`
- HTTP header: `X-CSRF-Token`

---

### Logging Middleware

Applied to all routes.

**Logs**:
- Request method and path
- Response status code
- Request duration
- User DID (if authenticated)

**Format**:
```
2025-10-30T10:15:30Z INFO GET /dashboard 200 45ms did:plc:abc123
```

---

## Summary

This HTTP API contract defines:

1. **Public Routes**: Landing page, about page
2. **Authentication**: OAuth login flow with Bluesky
3. **Protected Routes**: Dashboard, archive management, post browsing
4. **API Endpoints**: JSON endpoints for HTMX/AJAX requests
5. **Static Assets**: CSS, JS, and archived media
6. **Error Handling**: Standard error responses and custom error pages
7. **Middleware**: Authentication, CSRF protection, logging

All routes support both traditional server-side rendering (HTML responses) and modern HTMX patterns (HTML fragments and JSON). The API prioritizes usability, security, and progressive enhancement.
