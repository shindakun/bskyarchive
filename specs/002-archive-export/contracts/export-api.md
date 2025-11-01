# Export API Contract

**Feature**: 002-archive-export
**Date**: 2025-10-30

## HTTP Endpoints

### 1. GET /export

**Purpose**: Display export UI page

**Auth**: Required (RequireAuth middleware)

**Response**: HTML page with export form (format selection, date range picker)

**Template**: `internal/web/templates/pages/export.html`

---

### 2. POST /export/start

**Purpose**: Initiate export operation

**Auth**: Required

**Request Body** (form data):
```
format: "json" | "csv"
include_media: "true" | "false"
start_date: "2024-01-01" (optional, ISO 8601)
end_date: "2025-12-31" (optional, ISO 8601)
```

**Response** (JSON):
```json
{
  "job_id": "2025-10-30_14-30-45",
  "status": "running",
  "message": "Export started successfully"
}
```

**Status Codes**:
- 200: Export started
- 400: Invalid parameters
- 401: Not authenticated
- 500: Server error

---

### 3. GET /export/progress/:job_id

**Purpose**: Poll export progress

**Auth**: Required

**Response** (JSON):
```json
{
  "job_id": "2025-10-30_14-30-45",
  "status": "running",
  "posts_processed": 450,
  "posts_total": 1000,
  "media_copied": 120,
  "media_total": 300,
  "percent_complete": 45
}
```

**Status Values**: "running", "completed", "failed"

**Status Codes**:
- 200: Progress retrieved
- 404: Job not found
- 401: Not authenticated

---

### 4. GET /export/download/:job_id

**Purpose**: Download completed export as ZIP

**Auth**: Required

**Response**: application/zip file

**Status Codes**:
- 200: Download started
- 404: Export not found or not completed
- 401: Not authenticated

**Note**: Implementation may serve directory directly or create ZIP on-the-fly

---

## HTMX Integration

Export UI uses HTMX for progress polling:

```html
<div hx-get="/export/progress/{{.JobID}}"
     hx-trigger="every 2s"
     hx-swap="innerHTML">
  Loading progress...
</div>
```

**Polling**: Every 2 seconds while status is "running"
**Auto-stop**: When status becomes "completed" or "failed"

---

## Error Responses

All endpoints return consistent error format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE"
}
```

**Common Error Codes**:
- `INVALID_FORMAT`: Format not "json" or "csv"
- `INVALID_DATE_RANGE`: End date before start date
- `DISK_SPACE`: Insufficient disk space
- `NO_POSTS`: No posts match criteria
- `EXPORT_FAILED`: Export operation failed
