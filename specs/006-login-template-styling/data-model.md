# Data Model: Login Form Template & Styling

**Feature**: 006-login-template-styling
**Date**: 2025-11-02
**Status**: Complete

## Overview

This feature is primarily a UI refactoring with minimal data modeling requirements. The only "data" involved is the template rendering data structure passed from the handler to the template. No database schema changes, no new persistent entities.

## Template Data Structure

### LoginPageData

Represents the data passed to the login template for rendering.

**Purpose**: Provide all necessary information for rendering the login page, including error messages and contextual information.

**Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Title` | `string` | Yes | Page title (e.g., "Login - Bluesky Archive") |
| `Error` | `string` | No | Error message to display if login failed or validation error occurred |
| `Message` | `string` | No | Informational message (rarely used for login page, but included for consistency) |
| `Handle` | `string` | No | Pre-filled handle value (for repopulating form after validation error) |

**Go Struct Definition**:

```go
type LoginPageData struct {
    Title   string
    Error   string
    Message string
    Handle  string
}
```

**Usage Example**:

```go
// Success case - show login form
data := LoginPageData{
    Title:  "Login - Bluesky Archive",
    Error:  "",
    Handle: "",
}
renderTemplate(w, "login.html", data)

// Error case - validation failed
data := LoginPageData{
    Title:  "Login - Bluesky Archive",
    Error:  "Bluesky handle is required",
    Handle: r.FormValue("handle"), // Repopulate for user convenience
}
renderTemplate(w, "login.html", data)

// OAuth initiation error
data := LoginPageData{
    Title: "Login - Bluesky Archive",
    Error: "Failed to connect to Bluesky. Please try again.",
}
renderTemplate(w, "login.html", data)
```

**Validation Rules**:

- `Title`: Must not be empty
- `Error`: Empty string when no error, non-empty descriptive message on error
- `Message`: Typically empty for login page
- `Handle`: Should be populated only when redisplaying form after error

**State Transitions**:

1. **Initial Load**: `Error=""`, `Handle=""` → Display empty login form
2. **Validation Error**: `Error="..."`, `Handle="user_input"` → Display form with error and preserve user input
3. **OAuth Error**: `Error="..."`, `Handle=""` → Display form with error, clear handle
4. **Success**: Redirect to OAuth provider (no template rendering)

---

## Existing Data Models (No Changes)

This feature does not modify any existing data models. The following existing models remain unchanged:

### OAuth Session Data (in `internal/auth/oauth.go`)

**No Changes**: The OAuth flow state management remains identical. This feature only affects how the login form HTML is generated, not the OAuth logic itself.

**Existing Implementation**:
- `bskyoauth` library handles all OAuth state
- Session management unchanged
- Token storage unchanged

---

## No Database Schema Changes

This feature requires **zero** database migrations or schema changes:

- No new tables
- No new columns
- No new indexes
- No data migration required

**Rationale**: This is purely a presentation layer change. All data handling (OAuth tokens, sessions) remains unchanged.

---

## Data Flow

### Login Page Request Flow

```
1. User requests /auth/login (GET)
   ↓
2. Handler creates empty LoginPageData
   ↓
3. Template renders with empty form
   ↓
4. User submits handle (POST)
   ↓
5a. Validation fails → Handler creates LoginPageData with Error
    ↓
    Template renders with error message

5b. Validation passes → OAuth flow initiates
    ↓
    Redirect to Bluesky (no template rendering)
```

### Error Handling Flow

```
Client-side (HTML5):
- Browser validates required field
- Prevents submission if empty

Server-side:
- Handler validates handle format
- Creates LoginPageData with Error on failure
- Template displays error in Pico CSS styled article
```

---

## Template Data Contracts

### Input Contract (Handler → Template)

The handler must provide:

```go
data := LoginPageData{
    Title:   "Login - Bluesky Archive", // Required, non-empty
    Error:   "",                         // Optional, empty string if no error
    Message: "",                         // Optional, rarely used
    Handle:  "",                         // Optional, for repopulating form
}
```

### Output Contract (Template → User)

The template must render:

1. Page title in `<title>` tag and page header
2. Error message in `<article aria-label="Error">` if present
3. Login form with:
   - Label and input for Bluesky handle
   - Submit button
   - HTML5 validation (required attribute)
4. Helpful context text explaining the login process
5. Minimal navigation with About/Help links

---

## Edge Cases

### 1. Extremely Long Error Messages

**Scenario**: Server returns very long error message (e.g., full OAuth error JSON)

**Handling**: Template should wrap error text properly using Pico CSS article styling. Long messages will word-wrap within the container.

**Data Validation**: Handler should sanitize error messages to be user-friendly, not expose internal details.

### 2. Special Characters in Handle

**Scenario**: User enters handle with special characters (e.g., `<script>alert('xss')</script>`)

**Handling**: Go's `html/template` automatically escapes all template variables. No XSS vulnerability.

**Data Validation**: Handle validation should reject invalid characters before they reach the template.

### 3. Missing Template File

**Scenario**: Template file deleted or corrupted

**Handling**: Go template execution will return error. Handler should catch and display generic error page (existing error handling).

**No Data Impact**: This is a runtime error, not a data error. No data corruption possible.

### 4. Concurrent Requests

**Scenario**: User opens multiple login pages simultaneously

**Handling**: Each request gets independent LoginPageData instance. No shared state. No concurrency issues.

**Stateless Design**: Template rendering is stateless - perfect for concurrent requests.

---

## Data Privacy Considerations

### Sensitive Data Handling

**Handle (Bluesky username)**:
- **Sensitivity**: Low (public identifier)
- **Storage**: Not stored in this feature (passed to OAuth flow)
- **Logging**: Should not be logged in error messages
- **Display**: Safe to repopulate in form after validation error

**Error Messages**:
- **Sensitivity**: Low to Medium (may contain system information)
- **Sanitization**: Must be user-friendly, not expose internal errors
- **Example Good Error**: "Failed to connect to Bluesky. Please try again."
- **Example Bad Error**: "panic: nil pointer dereference at oauth.go:123"

**OAuth Tokens**:
- **Sensitivity**: High (authentication credentials)
- **Template Exposure**: NEVER passed to login template
- **Existing Security**: Handled by bskyoauth library (unchanged)

---

## Comparison with Existing Pages

This data structure follows the exact pattern used by other pages:

**export.html**:
```go
type ExportPageData struct {
    Title      string
    Error      string
    Message    string
    CSRFToken  string
    Status     *ExportStatus
    Exports    []ExportRecord
}
```

**dashboard.html**:
```go
type DashboardData struct {
    Title   string
    Error   string
    Message string
    Stats   *ArchiveStats
}
```

**login.html** (new):
```go
type LoginPageData struct {
    Title   string   // Same pattern
    Error   string   // Same pattern
    Message string   // Same pattern
    Handle  string   // Feature-specific
}
```

**Consistency**: All pages use `Title`, `Error`, `Message` fields. This maintains template rendering consistency across the application.

---

## Summary

This feature has minimal data modeling requirements:

✅ **New Data Structure**: LoginPageData (simple, 4 fields)
✅ **No Database Changes**: Zero schema modifications
✅ **No New Entities**: Purely presentation data
✅ **Consistent Pattern**: Follows existing template data conventions
✅ **Privacy Compliant**: No sensitive data exposure

The data model is intentionally minimal because this is a UI refactoring, not a feature that introduces new business logic or data storage requirements.
