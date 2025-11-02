# HTTP API Contracts: Export Download & Management

**Feature**: 005-export-download
**Date**: 2025-11-01
**Base URL**: `/export`

This document defines all HTTP endpoints for downloading and managing exports.

---

## Endpoints Overview

| Method | Path | Description | Auth Required |
|--------|------|-------------|---------------|
| GET | `/export` | Render export management page with list | Yes |
| GET | `/export/list` | Get list of user's exports (JSON) | Yes |
| GET | `/export/download/{export_id}` | Download export as ZIP | Yes |
| DELETE | `/export/delete/{export_id}` | Delete an export | Yes |

---

## 1. GET `/export`

Renders the export management page showing the export form and list of completed exports.

### Authentication

**Required**: Yes (session cookie)

### Request

**Query Parameters**: None

**Headers**:
- `Cookie: session_id={token}` (required)

### Response

**Status**: `200 OK`

**Content-Type**: `text/html`

**Body**: HTML page with:
- Export creation form (existing functionality)
- Table listing all user's completed exports
- Download and delete buttons for each export

### Example

```http
GET /export HTTP/1.1
Host: localhost:8080
Cookie: session_id=abc123...
```

```html
<!-- Response: HTML page -->
<section>
  <h1>Export Archive</h1>

  <!-- Export form (existing) -->
  <form id="export-form">...</form>

  <!-- Export list (new) -->
  <article>
    <header><strong>Your Exports</strong></header>
    <table>
      <thead>
        <tr>
          <th>Created</th>
          <th>Format</th>
          <th>Date Range</th>
          <th>Posts</th>
          <th>Media</th>
          <th>Size</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <!-- Rows for each export -->
      </tbody>
    </table>
  </article>
</section>
```

### Error Responses

| Status | Condition | Response |
|--------|-----------|----------|
| 302 Found | Not authenticated | Redirect to `/auth/login` |
| 500 Internal Server Error | Database error | Error message |

---

## 2. GET `/export/list`

Returns JSON list of user's completed exports.

### Authentication

**Required**: Yes (session cookie)

### Request

**Query Parameters**:
- `limit` (optional, integer, default: 50): Max exports to return
- `offset` (optional, integer, default: 0): Pagination offset

**Headers**:
- `Cookie: session_id={token}` (required)

### Response

**Status**: `200 OK`

**Content-Type**: `application/json`

**Body**:
```json
{
  "exports": [
    {
      "id": "did:plc:abc123/2025-11-01_14-30-00",
      "did": "did:plc:abc123",
      "format": "json",
      "created_at": "2025-11-01T14:30:00Z",
      "directory_path": "./exports/did:plc:abc123/2025-11-01_14-30-00",
      "post_count": 1234,
      "media_count": 567,
      "size_bytes": 123456789,
      "date_range_start": "2024-01-01T00:00:00Z",
      "date_range_end": "2024-12-31T23:59:59Z",
      "manifest_path": "./exports/did:plc:abc123/2025-11-01_14-30-00/manifest.json"
    }
  ],
  "total": 1
}
```

### Example

```http
GET /export/list?limit=10&offset=0 HTTP/1.1
Host: localhost:8080
Cookie: session_id=abc123...
```

### Error Responses

| Status | Condition | Response |
|--------|-----------|----------|
| 401 Unauthorized | Not authenticated | `{"error": "Unauthorized"}` |
| 500 Internal Server Error | Database error | `{"error": "Internal server error"}` |

---

## 3. GET `/export/download/{export_id}`

Downloads a completed export as a ZIP archive.

### Authentication

**Required**: Yes (session cookie)

### Request

**Path Parameters**:
- `export_id` (required, string): Export ID (format: `{did}/{timestamp}`, URL-encoded)

**Query Parameters**:
- `delete_after` (optional, boolean, default: false): Delete export after successful download

**Headers**:
- `Cookie: session_id={token}` (required)

### Response

**Status**: `200 OK`

**Content-Type**: `application/zip`

**Headers**:
- `Content-Disposition: attachment; filename="export-{timestamp}.zip"`
- `Content-Type: application/zip`
- `Content-Length: {size}` (if known in advance)

**Body**: Binary ZIP archive containing:
```
export-{timestamp}.zip
├── posts.json (or posts.csv)
├── manifest.json
└── media/
    ├── {filename1}.jpg
    ├── {filename2}.png
    └── ...
```

### Example

```http
GET /export/download/did%3Aplc%3Aabc123%2F2025-11-01_14-30-00 HTTP/1.1
Host: localhost:8080
Cookie: session_id=abc123...
```

```http
HTTP/1.1 200 OK
Content-Type: application/zip
Content-Disposition: attachment; filename="export-2025-11-01_14-30-00.zip"
Content-Length: 123456789

[Binary ZIP data...]
```

### Example with Delete After

```http
GET /export/download/did%3Aplc%3Aabc123%2F2025-11-01_14-30-00?delete_after=true HTTP/1.1
Host: localhost:8080
Cookie: session_id=abc123...
```

### Error Responses

| Status | Condition | Response |
|--------|-----------|----------|
| 401 Unauthorized | Not authenticated | `Unauthorized` |
| 403 Forbidden | Export belongs to another user | `Forbidden` |
| 404 Not Found | Export not found in database | `Export not found` |
| 404 Not Found | Export directory missing | `Export files not found` |
| 429 Too Many Requests | Rate limit exceeded (>10 concurrent) | `Too many concurrent downloads` |
| 500 Internal Server Error | Disk error, ZIP creation failed | `Failed to create archive` |

---

## 4. DELETE `/export/delete/{export_id}`

Deletes a completed export from disk and database.

### Authentication

**Required**: Yes (session cookie + CSRF token)

### Request

**Path Parameters**:
- `export_id` (required, string): Export ID (format: `{did}/{timestamp}`, URL-encoded)

**Headers**:
- `Cookie: session_id={token}` (required)
- `X-CSRF-Token: {token}` (required)

**Body**: None (or form-encoded CSRF token)

### Response

**Status**: `200 OK`

**Content-Type**: `application/json`

**Body**:
```json
{
  "message": "Export deleted successfully",
  "export_id": "did:plc:abc123/2025-11-01_14-30-00"
}
```

### Example

```http
DELETE /export/delete/did%3Aplc%3Aabc123%2F2025-11-01_14-30-00 HTTP/1.1
Host: localhost:8080
Cookie: session_id=abc123...
X-CSRF-Token: csrf_token_value
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "message": "Export deleted successfully",
  "export_id": "did:plc:abc123/2025-11-01_14-30-00"
}
```

### HTMX Usage

```html
<button hx-delete="/export/delete/{export_id}"
        hx-confirm="Delete this export? This action cannot be undone."
        hx-target="closest tr"
        hx-swap="outerHTML">
  Delete
</button>
```

### Error Responses

| Status | Condition | Response |
|--------|-----------|----------|
| 401 Unauthorized | Not authenticated | `Unauthorized` |
| 403 Forbidden | Export belongs to another user | `Forbidden` |
| 403 Forbidden | CSRF token missing/invalid | `Forbidden` |
| 404 Not Found | Export not found | `Export not found` |
| 500 Internal Server Error | Deletion failed | `{"error": "Failed to delete export"}` |

---

## Security Considerations

### Authorization

All endpoints MUST verify:
1. User is authenticated (valid session cookie)
2. Export belongs to requesting user (DID match)
3. State-changing operations (DELETE) require CSRF token

### Rate Limiting

- Download endpoint limits: Max 10 concurrent downloads per user
- No rate limit on list/delete (already protected by CSRF)

### Path Validation

- Export IDs must match format: `{did}/{timestamp}`
- Directory paths must start with `./exports/`
- Prevent path traversal attacks (e.g., `../../etc/passwd`)

### Audit Logging

Log all operations:
```
[INFO] Export downloaded: user={did}, export={export_id}, size={bytes}
[INFO] Export deleted: user={did}, export={export_id}
[WARN] Unauthorized download attempt: user={did}, attempted_export={export_id}, owner={actual_did}
```

---

## Content Negotiation

### Accept Header Support

- `text/html`: Return HTML page (default for browser requests)
- `application/json`: Return JSON response (for API clients)

Example:
```http
GET /export HTTP/1.1
Accept: application/json
```

Returns JSON instead of HTML.

---

## Error Response Format

All JSON error responses follow this structure:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "additional context"
  }
}
```

Example:
```json
{
  "error": "Export not found",
  "code": "EXPORT_NOT_FOUND",
  "details": {
    "export_id": "did:plc:abc123/2025-11-01_14-30-00"
  }
}
```

---

## Versioning

No API versioning for MVP. All endpoints are v1 implicit.

Future versioning (if needed): `/api/v2/export/...`

---

## Summary

This API contract defines:
- **4 endpoints** for export management
- **Authentication & authorization** via session cookies + DID verification
- **CSRF protection** for state-changing operations
- **Rate limiting** for downloads (10 concurrent per user)
- **Streaming responses** for large ZIP files
- **Error handling** with appropriate HTTP status codes
- **Audit logging** for security monitoring

All endpoints integrate with existing authentication system and follow RESTful conventions.
