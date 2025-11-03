# Template Interface Contract: Login Page

**Feature**: 006-login-template-styling
**Date**: 2025-11-02
**Type**: Internal Template Rendering Contract

## Overview

This document defines the contract between the OAuth handler (`internal/auth/oauth.go`) and the login template (`internal/web/templates/pages/login.html`). Since this is an internal UI feature with no external API, the contract is limited to the template rendering interface.

---

## Handler → Template Contract

### Endpoint: GET /auth/login

**Purpose**: Display the login form to the user

**Handler Responsibility**:
```go
// Handler must provide LoginPageData struct to template
type LoginPageData struct {
    Title   string // Page title, must not be empty
    Error   string // Error message (empty if no error)
    Message string // Info message (rarely used, can be empty)
    Handle  string // Pre-filled handle (empty on first load)
}

// Render template
tmpl.ExecuteTemplate(w, "login.html", data)
```

**Template Requirements**:
- Must define `{{define "title"}}` block
- Must define `{{define "nav"}}` block
- Must define `{{define "content"}}` block
- Must handle `.Error` field (display if non-empty)
- Must render form with handle input and submit button
- Must use POST method to same endpoint

**HTTP Response**:
- Status: `200 OK`
- Content-Type: `text/html; charset=utf-8`
- Body: Rendered HTML from template

---

### Endpoint: POST /auth/login

**Purpose**: Process handle submission and initiate OAuth flow

**Request Format**:
```
POST /auth/login HTTP/1.1
Content-Type: application/x-www-form-urlencoded

handle=user.bsky.social
```

**Response Scenarios**:

#### Success: OAuth Flow Initiated
```
HTTP/1.1 302 Found
Location: https://bsky.social/oauth/authorize?client_id=...&state=...
```

Handler redirects to Bluesky OAuth authorization page. No template rendering occurs.

#### Failure: Validation Error
```
HTTP/1.1 200 OK
Content-Type: text/html; charset=utf-8

[Rendered login template with Error field populated]
```

Handler re-renders login template with error message:
```go
data := LoginPageData{
    Title:  "Login - Bluesky Archive",
    Error:  "Bluesky handle is required",
    Handle: r.FormValue("handle"),
}
tmpl.ExecuteTemplate(w, "login.html", data)
```

#### Failure: OAuth Initiation Error
```
HTTP/1.1 200 OK
Content-Type: text/html; charset=utf-8

[Rendered login template with Error field populated]
```

Handler re-renders with server error:
```go
data := LoginPageData{
    Title: "Login - Bluesky Archive",
    Error: "Failed to connect to Bluesky. Please try again.",
}
tmpl.ExecuteTemplate(w, "login.html", data)
```

---

## Template → Handler Contract

### Form Submission Contract

**Form Element**:
```html
<form method="POST" action="/auth/login">
    <label for="handle">Bluesky Handle</label>
    <input type="text"
           id="handle"
           name="handle"
           placeholder="user.bsky.social"
           required
           value="{{.Handle}}">
    <button type="submit">Login with Bluesky</button>
</form>
```

**Required Attributes**:
- `method="POST"` - Handler expects POST
- `name="handle"` - Handler reads via `r.FormValue("handle")`
- `required` - HTML5 client-side validation
- `value="{{.Handle}}"` - Repopulate on error

**Form Data Contract**:
```
Field Name: handle
Type: string
Required: yes
Format: Bluesky handle (e.g., "user.bsky.social" or "user.com")
Validation: Non-empty, valid handle format
```

---

## Template Structure Contract

### Required Template Blocks

The login template **MUST** implement these three blocks to work with the base layout:

#### 1. Title Block
```html
{{define "title"}}Login - Bluesky Archive{{end}}
```

**Purpose**: Sets `<title>` tag and may be used in page headers
**Required**: Yes
**Format**: Plain text, no HTML

#### 2. Navigation Block
```html
{{define "nav"}}
<nav class="container">
    <ul>
        <li><strong>Bluesky Archive</strong></li>
    </ul>
    <ul>
        <li><a href="/about">About</a></li>
    </ul>
</nav>
{{end}}
```

**Purpose**: Defines page navigation
**Required**: Yes (can be minimal for login page)
**Format**: HTML nav element with Pico CSS styling

#### 3. Content Block
```html
{{define "content"}}
<section class="container">
    <!-- Error display -->
    {{if .Error}}
    <article aria-label="Error">
        <header><strong>Error</strong></header>
        <p>{{.Error}}</p>
    </article>
    {{end}}

    <!-- Login form -->
    <article>
        <header><strong>Login with Bluesky</strong></header>
        <form method="POST">
            <!-- Form fields -->
        </form>
    </article>
</section>
{{end}}
```

**Purpose**: Main page content
**Required**: Yes
**Format**: HTML with Pico CSS classes

---

## Error Display Contract

### Error Message Format

**Handler → Template**:
```go
data.Error = "User-friendly error message"
```

**Template → User**:
```html
{{if .Error}}
<article aria-label="Error" style="background-color: var(--del-color); border-left: 4px solid #dc3545;">
    <header><strong>Error</strong></header>
    <p>{{.Error}}</p>
</article>
{{end}}
```

**Error Message Guidelines**:

✅ **Good Error Messages** (user-friendly):
- "Bluesky handle is required"
- "Failed to connect to Bluesky. Please try again."
- "Invalid handle format. Please use format: user.bsky.social"

❌ **Bad Error Messages** (internal details):
- "panic: nil pointer dereference"
- "OAuth client initialization failed: invalid config"
- "database connection error: timeout"

**Security Note**: Never expose internal system details, stack traces, or configuration in error messages.

---

## CSS Styling Contract

### Pico CSS Classes

The template must use these Pico CSS classes for consistency:

| Element | CSS Class | Purpose |
|---------|-----------|---------|
| Main container | `container` | Responsive centering and padding |
| Navigation | `<nav class="container">` | Consistent navigation styling |
| Error article | `<article aria-label="Error">` | Semantic error display |
| Form fieldset | `<fieldset>` | Form grouping (optional) |
| Input fields | (no class) | Pico CSS auto-styles inputs |
| Buttons | (no class) | Pico CSS auto-styles buttons |
| Help text | `<small>` | Muted help text |

**Classless Styling**: Pico CSS automatically styles most HTML elements without requiring classes. Only use classes for containers and semantic elements.

---

## Validation Contract

### Client-Side Validation (HTML5)

**Template Responsibility**:
```html
<input type="text" name="handle" required>
```

- `required` attribute: Browser prevents empty submission
- `type="text"`: Standard text input (no special validation)
- `placeholder`: Provides format example

**Browser Behavior**:
- Prevents form submission if field is empty
- Shows browser default error message
- No JavaScript required

### Server-Side Validation (Handler)

**Handler Responsibility**:
```go
handle := r.FormValue("handle")
if handle == "" {
    // Render template with error
    data.Error = "Bluesky handle is required"
    return
}

// Validate handle format
if !isValidHandle(handle) {
    data.Error = "Invalid handle format"
    return
}
```

**Validation Rules**:
1. Non-empty (redundant with HTML5, but required for security)
2. Valid handle format (domain name or subdomain.domain)
3. Reasonable length (< 255 characters)

---

## OAuth Flow Contract

### Successful OAuth Initiation

**Handler Action**:
```go
// Start OAuth flow
flowState, err := om.client.StartAuthFlow(ctx, handle)
if err != nil {
    // Return error via template
    return
}

// Redirect to Bluesky
http.Redirect(w, r, flowState.AuthURL, http.StatusFound)
```

**Template Not Involved**: On success, handler redirects to Bluesky. Template is not rendered.

### OAuth Callback (Out of Scope)

**Note**: The OAuth callback flow (`/auth/callback`) is **not** part of this feature. It remains unchanged and does not use the login template.

---

## Testing Contract

### Manual Testing Checklist

**Visual Consistency**:
- [ ] Login page matches design of export.html, dashboard.html
- [ ] Pico CSS styling applied correctly
- [ ] Responsive design works on mobile/tablet/desktop
- [ ] Error messages display in red article with proper formatting

**Functional Testing**:
- [ ] Empty form submission shows HTML5 validation message
- [ ] Valid handle initiates OAuth flow (redirect to Bluesky)
- [ ] Invalid handle shows server-side error message
- [ ] Error message displays with previously entered handle repopulated
- [ ] Navigation links work correctly
- [ ] Page loads in < 500ms

**Security Testing**:
- [ ] XSS attempt in handle field is escaped by template engine
- [ ] Error messages don't expose internal details
- [ ] OAuth flow initiates correctly (no CSRF vulnerability)

---

## Backward Compatibility

### Breaking Changes: None

This feature is a **drop-in replacement** for the existing inline HTML:

**Before** (inline HTML):
```go
w.Write([]byte(`<html>...</html>`))
```

**After** (template rendering):
```go
tmpl.ExecuteTemplate(w, "login.html", data)
```

**User Experience**: Identical OAuth flow, same form fields, same POST endpoint. Only visual styling changes.

**API Contract**: No API changes. Internal implementation detail only.

---

## Summary

| Contract Element | Handler Responsibility | Template Responsibility |
|-----------------|------------------------|-------------------------|
| Data Structure | Provide LoginPageData | Render with {{.Field}} |
| Error Display | Set .Error field | Show error if present |
| Form Submission | Process POST /auth/login | Submit with name="handle" |
| OAuth Flow | Redirect on success | N/A (not involved) |
| Validation | Server-side validation | Client-side HTML5 required |
| Styling | N/A (not involved) | Apply Pico CSS classes |
| Navigation | N/A (not involved) | Define nav block |

This contract ensures clean separation of concerns: handlers manage logic and data, templates manage presentation.
