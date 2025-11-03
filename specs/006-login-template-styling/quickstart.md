# Quickstart Guide: Login Form Template & Styling

**Feature**: 006-login-template-styling
**Date**: 2025-11-02
**Estimated Time**: 1-2 hours

## Overview

This guide provides step-by-step instructions for implementing the login form template migration. Follow these steps in order for a smooth implementation.

---

## Prerequisites

Before starting:

✅ **Review these documents**:
- [spec.md](spec.md) - Feature requirements and user stories
- [research.md](research.md) - Technical decisions and patterns
- [data-model.md](data-model.md) - Template data structure
- [contracts/template-interface.md](contracts/template-interface.md) - Handler-template contract

✅ **Verify existing code**:
- Confirm `internal/auth/oauth.go` contains inline HTML (lines 40-52)
- Check `internal/web/templates/layouts/base.html` exists
- Check `internal/web/templates/pages/export.html` as reference
- Verify Pico CSS is loaded in base template

✅ **Testing environment**:
- Go 1.21+ installed
- Development server running
- Browser for visual testing

---

## Step-by-Step Implementation

### Step 1: Create the Login Template

**File**: `internal/web/templates/pages/login.html`

**Action**: Create new template file with complete structure

**Implementation**:

```html
{{define "title"}}Login - Bluesky Archive{{end}}

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

{{define "content"}}
<section class="container">
    <!-- Page header -->
    <hgroup>
        <h1>Login with Bluesky</h1>
        <h2>Archive your Bluesky posts and media for safekeeping</h2>
    </hgroup>

    <!-- Error display -->
    {{if .Error}}
    <article aria-label="Error" style="background-color: var(--del-color); border-left: 4px solid #dc3545;">
        <header><strong>Error</strong></header>
        <p>{{.Error}}</p>
    </article>
    {{end}}

    <!-- Success message (rarely used for login, but included for consistency) -->
    {{if .Message}}
    <article aria-label="Success">
        <header><strong>Success</strong></header>
        <p>{{.Message}}</p>
    </article>
    {{end}}

    <!-- Login form -->
    <article>
        <header><strong>Sign In</strong></header>

        <p>Enter your Bluesky handle to get started. You'll be redirected to Bluesky to authorize this application.</p>

        <form method="POST" action="/auth/login">
            <label for="handle">
                Bluesky Handle
                <input type="text"
                       id="handle"
                       name="handle"
                       placeholder="user.bsky.social"
                       required
                       value="{{.Handle}}"
                       autocomplete="username">
            </label>
            <small>Your full Bluesky handle, including the domain (e.g., user.bsky.social or user.custom-domain.com)</small>

            <button type="submit">Continue with Bluesky</button>
        </form>
    </article>

    <!-- Additional context -->
    <article>
        <header><strong>About This Application</strong></header>
        <p>
            Bluesky Archive is a local-first tool that helps you back up your Bluesky posts,
            media, and profile data. All data is stored locally on your machine, giving you
            complete control and ownership.
        </p>
        <p>
            <strong>Privacy First</strong>: Your data never leaves your computer. OAuth tokens
            are encrypted and stored securely. No telemetry or analytics.
        </p>
        <p>
            <a href="/about">Learn more about how this works →</a>
        </p>
    </article>
</section>
{{end}}
```

**Design Decisions Implemented**:
- ✅ Three-block structure (title, nav, content)
- ✅ Pico CSS classless styling
- ✅ Error display using article with aria-label
- ✅ Contextual help text about the application
- ✅ Minimal navigation (Bluesky Archive logo + About link)
- ✅ Form with proper accessibility (labels, placeholders, required)
- ✅ `autocomplete="username"` for better UX
- ✅ Repopulates handle on validation error via `{{.Handle}}`

**Verification**:
```bash
# Check file exists
ls -la internal/web/templates/pages/login.html

# Verify syntax (Go template check)
go run cmd/bskyarchive/main.go --help
# (Server should start without template parsing errors)
```

---

### Step 2: Define the LoginPageData Struct

**File**: `internal/auth/oauth.go` (or create `internal/models/page_data.go` if you want to centralize)

**Action**: Add struct definition for template data

**Implementation**:

**Option A**: Add to `internal/auth/oauth.go` (simpler, keeps it local)

```go
// Add near the top of the file after imports

// LoginPageData represents the data passed to the login template
type LoginPageData struct {
    Title   string
    Error   string
    Message string
    Handle  string
}
```

**Option B**: Add to `internal/models/page_data.go` (better organization)

```go
// Create new file: internal/models/page_data.go
package models

// LoginPageData represents the data passed to the login template
type LoginPageData struct {
    Title   string // Page title
    Error   string // Error message (empty if no error)
    Message string // Info message (rarely used)
    Handle  string // Pre-filled handle value (for form repopulation)
}
```

**Recommendation**: Option B (separate file) for better code organization, especially if you plan to add more page data structs later.

**Verification**:
```bash
# Verify Go compiles
go build ./internal/models/
```

---

### Step 3: Update OAuth Handler to Use Template

**File**: `internal/auth/oauth.go`

**Action**: Replace inline HTML with template rendering

**Current Code** (lines 34-54):
```go
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
	// ... rest of POST handling ...
}
```

**New Code**:

First, add template support. You'll need to load and parse templates. Check how other handlers do this - likely there's a template helper in `internal/web/handlers/template.go`.

**Investigation needed**:
```bash
# Check template helper structure
cat internal/web/handlers/template.go | grep -A 10 "func.*Template"
```

**Assuming template helper exists** (adjust based on actual implementation):

```go
import (
    "net/http"
    "fmt"

    "github.com/shindakun/bskyarchive/internal/models"
    "github.com/shindakun/bskyarchive/internal/web/handlers" // For template rendering
)

func (om *OAuthManager) HandleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Render login template with empty data
		data := models.LoginPageData{
			Title:  "Login - Bluesky Archive",
			Error:  "",
			Handle: "",
		}

		// Use existing template rendering helper
		// NOTE: Adjust this based on actual template helper implementation
		if err := handlers.RenderTemplate(w, "login.html", data); err != nil {
			http.Error(w, "Failed to render login page", http.StatusInternalServerError)
			log.Printf("Template rendering error: %v", err)
			return
		}
		return
	}

	// POST: Start OAuth flow with handle
	handle := r.FormValue("handle")
	if handle == "" {
		// Validation error - re-render with error message
		data := models.LoginPageData{
			Title:  "Login - Bluesky Archive",
			Error:  "Bluesky handle is required",
			Handle: handle, // Empty in this case, but included for consistency
		}
		if err := handlers.RenderTemplate(w, "login.html", data); err != nil {
			http.Error(w, "Failed to render login page", http.StatusInternalServerError)
			return
		}
		return
	}

	// Start OAuth flow
	ctx := r.Context()
	flowState, err := om.client.StartAuthFlow(ctx, handle)
	if err != nil {
		// OAuth error - re-render with error message
		data := models.LoginPageData{
			Title:  "Login - Bluesky Archive",
			Error:  "Failed to connect to Bluesky. Please try again.",
			Handle: handle,
		}
		if err := handlers.RenderTemplate(w, "login.html", data); err != nil {
			http.Error(w, "Failed to render login page", http.StatusInternalServerError)
			return
		}
		return
	}

	// Store OAuth state and redirect
	om.sessionManager.Put(r.Context(), "oauth_state", flowState.State)
	http.Redirect(w, r, flowState.AuthURL, http.StatusFound)
}
```

**Important Notes**:
1. **Template Helper**: You MUST check the actual implementation of template rendering in your codebase. The `handlers.RenderTemplate` function name is a guess - adjust based on reality.
2. **Error Handling**: Always handle template rendering errors gracefully
3. **Logging**: Log template errors for debugging (use existing logger)
4. **Context**: Existing OAuth logic remains 100% unchanged

**Verification**:
```bash
# Compile check
go build ./internal/auth/

# Run tests
go test ./internal/auth/
```

---

### Step 4: Investigate and Integrate Template Rendering

**Action**: Find the existing template rendering infrastructure

**Commands**:
```bash
# Find template initialization
grep -r "template.New\|template.ParseFiles\|template.ParseGlob" internal/

# Find template execution
grep -r "ExecuteTemplate\|Execute" internal/web/handlers/

# Check how other pages render templates
cat internal/web/handlers/export.go | grep -A 5 "ExecuteTemplate"
```

**Common Patterns**:

**Pattern 1**: Global template variable
```go
// In main.go or server initialization
var templates *template.Template

func init() {
    templates = template.Must(template.ParseGlob("internal/web/templates/**/*.html"))
}

// In handler
templates.ExecuteTemplate(w, "login.html", data)
```

**Pattern 2**: Template helper function
```go
// In internal/web/handlers/template.go
func RenderTemplate(w http.ResponseWriter, name string, data interface{}) error {
    return templates.ExecuteTemplate(w, name, data)
}
```

**Pattern 3**: Per-request template parsing (less common, slower)
```go
tmpl, err := template.ParseFiles("internal/web/templates/pages/login.html")
if err != nil {
    return err
}
return tmpl.Execute(w, data)
```

**Action**: Once you identify the pattern, integrate it into the OAuth handler as shown in Step 3.

**Verification**:
```bash
# Start the development server
go run cmd/bskyarchive/main.go

# Test in browser
open http://localhost:8080/auth/login

# Check for template parsing errors in server logs
```

---

### Step 5: Manual Testing

**Action**: Comprehensive manual testing checklist

#### Visual Testing

1. **Load login page**:
   ```
   http://localhost:8080/auth/login
   ```

   ✅ **Verify**:
   - [ ] Page loads without errors
   - [ ] Pico CSS styling applied (centered layout, styled form)
   - [ ] Navigation shows "Bluesky Archive" and "About" link
   - [ ] Page title shows "Login - Bluesky Archive"
   - [ ] Form has handle input with placeholder
   - [ ] Button says "Continue with Bluesky"
   - [ ] "About This Application" context section visible

2. **Responsive design testing**:
   - [ ] Desktop (1920px): Layout centered, readable
   - [ ] Tablet (768px): Layout adapts, no horizontal scroll
   - [ ] Mobile (375px): Form stacks vertically, touch-friendly

3. **Compare with other pages**:
   - Open http://localhost:8080/export
   - Compare:
     - [ ] Same header style
     - [ ] Same navigation structure
     - [ ] Same button styling
     - [ ] Same color scheme
     - [ ] Same typography (fonts, sizes)

#### Functional Testing

4. **Empty form submission** (HTML5 validation):
   - Leave handle field empty
   - Click "Continue with Bluesky"
   - ✅ **Expected**: Browser shows "Please fill out this field" message
   - ✅ **Expected**: Form does NOT submit

5. **Valid handle submission** (OAuth flow):
   - Enter: `user.bsky.social`
   - Click "Continue with Bluesky"
   - ✅ **Expected**: Redirect to `https://bsky.social/oauth/authorize?...`
   - ✅ **Expected**: OAuth flow initiates (Bluesky login page)

6. **Server-side validation** (backend error):
   - Temporarily modify handler to always return error:
     ```go
     data.Error = "Test error message"
     ```
   - Submit form
   - ✅ **Expected**: Error article displays in red
   - ✅ **Expected**: Error message: "Test error message"
   - ✅ **Expected**: Handle field repopulated with entered value
   - ✅ **Expected**: Page does NOT redirect

7. **OAuth initiation error** (network failure):
   - Disconnect network OR modify OAuth client to fail
   - Submit valid handle
   - ✅ **Expected**: Error article displays
   - ✅ **Expected**: User-friendly error message (not stack trace)
   - ✅ **Expected**: Handle field repopulated

#### Security Testing

8. **XSS attempt**:
   - Enter handle: `<script>alert('XSS')</script>`
   - Submit form
   - ✅ **Expected**: Script does NOT execute
   - ✅ **Expected**: Handle displayed as plain text in error (if shown)
   - ✅ **Reason**: Go html/template escapes all variables

9. **SQL injection attempt** (irrelevant but good to verify):
   - Enter handle: `admin' OR '1'='1`
   - ✅ **Expected**: Handled as regular string, no errors

10. **Long input**:
    - Enter 300-character handle
    - ✅ **Expected**: Validation error (invalid handle format)
    - ✅ **Expected**: Page renders without layout breaking

#### Performance Testing

11. **Page load speed**:
    - Open browser DevTools (Network tab)
    - Load http://localhost:8080/auth/login
    - ✅ **Expected**: Page loads in < 500ms (Success Criteria SC-001)
    - ✅ **Expected**: Template rendering < 50ms (logged if you add metrics)

#### Edge Case Testing

12. **Missing template file**:
    - Temporarily rename `login.html` to `login.html.bak`
    - Try to load page
    - ✅ **Expected**: HTTP 500 error (graceful failure)
    - ✅ **Expected**: Server logs show template error
    - Restore file

13. **Malformed template**:
    - Add syntax error to template: `{{.InvalidField}}`
    - Try to load page
    - ✅ **Expected**: HTTP 500 error OR template parsing error on startup
    - Fix syntax error

---

### Step 6: Integration Testing (Optional but Recommended)

**File**: `tests/integration/login_template_test.go`

**Action**: Create automated test for template rendering

**Implementation**:

```go
package integration

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/shindakun/bskyarchive/internal/auth"
	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/web/handlers"
)

func TestLoginPageRenders(t *testing.T) {
	// TODO: Set up test server with template rendering
	// This is a placeholder - actual implementation depends on your test infrastructure

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	w := httptest.NewRecorder()

	// TODO: Call handler
	// handler(w, req)

	resp := w.Result()
	body := w.Body.String()

	// Verify response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify template rendered correctly
	expectedStrings := []string{
		"<title>Login - Bluesky Archive</title>",
		"Login with Bluesky",
		`<input type="text" name="handle"`,
		"Continue with Bluesky",
		"About This Application",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(body, expected) {
			t.Errorf("Response missing expected string: %s", expected)
		}
	}
}

func TestLoginPageShowsError(t *testing.T) {
	// TODO: Create request that triggers error
	// Verify error message displays in template
}

func TestLoginFormSubmission(t *testing.T) {
	// TODO: Submit form with valid handle
	// Verify redirect to OAuth URL
}

func TestLoginFormValidation(t *testing.T) {
	// TODO: Submit form with empty handle
	// Verify error message and form repopulation
}
```

**Note**: Integration testing setup depends on your existing test infrastructure. The above is a starting point - adapt based on how other handlers are tested in your codebase.

**Verification**:
```bash
# Run integration tests
go test ./tests/integration/ -v

# Run all tests
go test ./...
```

---

### Step 7: Code Review and Cleanup

**Action**: Final review before committing

**Checklist**:

- [ ] **Code Quality**:
  - [ ] No commented-out code left behind
  - [ ] Removed inline HTML completely from oauth.go
  - [ ] Proper error handling (no panics)
  - [ ] Consistent naming conventions
  - [ ] Added comments for non-obvious code

- [ ] **Testing**:
  - [ ] All manual tests passed (Steps 5)
  - [ ] Integration tests pass (if implemented)
  - [ ] OAuth flow still works end-to-end
  - [ ] No regressions in other pages

- [ ] **Documentation**:
  - [ ] Code comments explain template data structure
  - [ ] TODO comments removed
  - [ ] Commit message descriptive

- [ ] **Performance**:
  - [ ] Page load < 500ms verified
  - [ ] No memory leaks (check with long-running server)
  - [ ] Template caching working (not re-parsing on every request)

- [ ] **Visual Consistency**:
  - [ ] Login page matches export page styling
  - [ ] Responsive design tested
  - [ ] Pico CSS classes used correctly
  - [ ] No visual regressions

---

### Step 8: Commit and Document

**Action**: Commit changes with clear message

**Git Workflow**:

```bash
# Stage changes
git add internal/web/templates/pages/login.html
git add internal/auth/oauth.go
git add internal/models/page_data.go  # If created
git add tests/integration/login_template_test.go  # If created

# Commit with descriptive message
git commit -m "feat: migrate login form to template with Pico CSS styling

- Create login.html template using three-block structure
- Add LoginPageData struct for template rendering
- Update HandleOAuthLogin to use template instead of inline HTML
- Maintain 100% functional parity with existing OAuth flow
- Add contextual help text and improved error handling
- Follows existing template patterns from export.html

Closes #006 (if you use issue tracking)"

# Push to feature branch
git push origin 006-login-template-styling
```

**Documentation Updates**:

1. **Update CLAUDE.md** (if needed):
   - Add note about template structure if not already documented
   - Mention LoginPageData as example of template data pattern

2. **Update CHANGELOG.md** (if you have one):
   ```markdown
   ## [Unreleased]

   ### Changed
   - Login form now uses proper template with Pico CSS styling
   - Improved login page UX with contextual information
   ```

---

## Troubleshooting

### Problem: Template not found error

**Symptom**: `template: login.html not found`

**Solution**:
1. Verify file path: `internal/web/templates/pages/login.html`
2. Check template glob pattern in template initialization
3. Ensure template parsing happens before handler registration
4. Check file permissions (readable)

---

### Problem: Template parses but displays blank page

**Symptom**: HTTP 200 response, but empty body

**Solution**:
1. Check template block names match base layout expectations
2. Verify `{{define "title"}}`, `{{define "nav"}}`, `{{define "content"}}` are all present
3. Check template execution error (may be silently failing)
4. Add logging: `log.Printf("Template exec error: %v", err)`

---

### Problem: Styling doesn't match other pages

**Symptom**: Login page looks different from export page

**Solution**:
1. Verify Pico CSS is loaded (check base.html)
2. Check class names: `container`, `article`, `<nav>` structure
3. Compare HTML structure with export.html side-by-side
4. Check for CSS syntax errors in template
5. Clear browser cache (Ctrl+Shift+R)

---

### Problem: OAuth flow broken after changes

**Symptom**: Form submits but OAuth doesn't initiate

**Solution**:
1. Check form `method="POST"` attribute
2. Verify form `action="/auth/login"` points to correct endpoint
3. Check `name="handle"` attribute on input field
4. Verify handler reads `r.FormValue("handle")` correctly
5. Add debug logging: `log.Printf("Handle: %s", handle)`
6. Check OAuth client initialization (should be unchanged)

---

### Problem: Error messages not displaying

**Symptom**: Validation fails but no error shown

**Solution**:
1. Verify `{{if .Error}}` block in template
2. Check handler sets `data.Error` field
3. Verify template data passed correctly: `log.Printf("Data: %+v", data)`
4. Check template execution succeeds (no early return on error)
5. Inspect HTML source: error may be rendered but CSS issue hiding it

---

### Problem: Form doesn't repopulate handle after error

**Symptom**: User enters handle, gets error, handle disappears

**Solution**:
1. Check template has `value="{{.Handle}}"` attribute
2. Verify handler sets `data.Handle = r.FormValue("handle")` on error
3. Test with debug output: `log.Printf("Repopulating handle: %s", data.Handle)`

---

## Success Criteria Verification

Before marking this feature complete, verify all success criteria from [spec.md](spec.md):

- [ ] **SC-001**: Login page loads in < 500ms
- [ ] **SC-002**: 100% functional parity with existing OAuth flow
- [ ] **SC-003**: 95%+ visual consistency with other pages
- [ ] **SC-004**: Responsive design (320px - 1920px)
- [ ] **SC-005**: Code maintainability improved (HTML separated from Go)

---

## Next Steps

After completing implementation:

1. **Run `/speckit.tasks`**: Generate detailed task breakdown from this plan
2. **Create PR**: Submit for code review
3. **User Acceptance Testing**: Have team members test the login flow
4. **Document learnings**: Note any gotchas for future template migrations
5. **Consider follow-up**: Other pages that need template migration?

---

## Estimated Timeline

- **Step 1** (Create template): 30 minutes
- **Step 2** (Define struct): 5 minutes
- **Step 3** (Update handler): 20 minutes
- **Step 4** (Integrate template rendering): 10 minutes
- **Step 5** (Manual testing): 30 minutes
- **Step 6** (Integration tests): 30 minutes (optional)
- **Step 7** (Code review): 15 minutes
- **Step 8** (Commit/document): 10 minutes

**Total**: ~2 hours (or ~1.5 hours if skipping automated tests)

**Complexity**: 🟢 Low - Straightforward template migration with clear reference implementation

---

## Questions?

If you encounter issues not covered in this guide:

1. Review [research.md](research.md) for technical decisions
2. Check [contracts/template-interface.md](contracts/template-interface.md) for handler-template contract
3. Compare with existing pages (export.html, dashboard.html)
4. Check Go html/template documentation: https://pkg.go.dev/html/template
5. Review Pico CSS docs: https://picocss.com/docs

---

**Happy coding! 🚀**
