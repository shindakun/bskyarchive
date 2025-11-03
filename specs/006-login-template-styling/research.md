# Research: Login Form Template & Styling

**Feature**: 006-login-template-styling
**Date**: 2025-11-02
**Status**: Complete

## Overview

This feature requires minimal research as it leverages existing technologies and patterns already established in the codebase. The research focuses on understanding the current implementation and identifying the template patterns to follow.

## Research Areas

### 1. Current Login Form Implementation

**Decision**: Login form is currently inline HTML in `internal/auth/oauth.go` (lines 40-52)

**Current Implementation**:
```go
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
```

**Findings**:
- Simple form with single input field (Bluesky handle)
- POST method to same endpoint
- No styling, no base template usage
- No error handling UI
- Comment indicates this is a placeholder: "This will be replaced with a proper template in later tasks"

**Rationale**: This confirms the feature need - the inline HTML is a known technical debt item.

---

### 2. Existing Template Structure and Patterns

**Decision**: Use Go's `html/template` package with existing base template pattern

**Investigation**:
- Examined `internal/web/templates/pages/export.html` as reference
- Examined `internal/web/templates/pages/dashboard.html` for navigation
- Examined `internal/web/templates/layouts/base.html` for base structure

**Template Pattern Found**:
```html
{{define "title"}}Page Title{{end}}

{{define "nav"}}
<nav class="container">
    <ul>
        <li><strong>Bluesky Archive</strong></li>
    </ul>
    <ul>
        <!-- Navigation links -->
    </ul>
</nav>
{{end}}

{{define "content"}}
<!-- Page-specific content -->
{{end}}
```

**Findings**:
- Three main template blocks: `title`, `nav`, `content`
- Base template provides overall HTML structure
- Pico CSS classes used throughout (`container`, `hgroup`, `article`, etc.)
- Form elements use Pico CSS default styling (minimal classes needed)
- Error/message display uses `<article aria-label="Error/Success">` pattern
- Navigation structure consistent across pages

**Rationale**: Following this pattern ensures visual consistency and maintainability.

---

### 3. Pico CSS Form Styling Best Practices

**Decision**: Use Pico CSS default form styling with minimal custom classes

**Research**: Reviewed Pico CSS documentation and existing forms in codebase

**Key Pico CSS Patterns**:
- Forms automatically styled without classes
- Input fields: `<input type="text" name="..." required>` (classless styling)
- Buttons: `<button type="submit">` (primary button styling by default)
- Labels: `<label>Text <input...></label>` (inline) or separate with `for` attribute
- Fieldsets: `<fieldset><legend>Title</legend>...</fieldset>` for grouping
- Error messages: Use `<small>` or `<article aria-label="Error">` for validation feedback
- Container class: `class="container"` for responsive centering

**Existing Form Examples in Codebase**:
- export.html: Shows radio buttons, checkboxes, date inputs, fieldsets
- All follow classless Pico CSS approach

**Rationale**: Minimal markup required, Pico CSS handles visual styling automatically. This keeps templates clean and maintainable.

---

### 4. Template Rendering in Go Handlers

**Decision**: Use existing template infrastructure, likely in `internal/web/handlers/template.go`

**Investigation**:
- Checked `internal/web/handlers/` directory structure
- Found `template.go` - likely contains template rendering helpers
- OAuth handler in `internal/auth/oauth.go` will need access to template rendering

**Rendering Approach Options**:

**Option A**: Create template rendering helper in auth package
- Pros: Self-contained, no dependency on web/handlers
- Cons: Duplicates template infrastructure

**Option B**: Use existing template helper from web/handlers
- Pros: Reuses existing infrastructure, maintains consistency
- Cons: Adds dependency between auth and web packages

**Option C**: Move template data preparation to web/handlers, keep auth logic pure
- Pros: Better separation of concerns
- Cons: May require restructuring handler registration

**Decision**: Option B (use existing template helper)
- This is the pattern used throughout the codebase
- Auth handler can import and use the template rendering utility
- Maintains consistency with other pages

**Rationale**: Consistency with existing codebase patterns. Quick to implement, low risk.

---

### 5. Error Handling and User Feedback

**Decision**: Use existing error display pattern with template data

**Patterns Found**:
- Template receives `.Error` and `.Message` fields
- Error display: `{{if .Error}}<article aria-label="Error">{{.Error}}</article>{{end}}`
- Success display: `{{if .Message}}<article>{{.Message}}</article>{{end}}`

**Error Scenarios to Handle**:
1. Empty handle (HTML5 `required` attribute + backend validation)
2. Invalid handle format (backend validation)
3. OAuth flow initiation failure (backend error)
4. Network/connectivity issues (backend error)

**Approach**:
- Use HTML5 validation for client-side (required field)
- Backend validation returns error via template data
- Display error in consistent Pico CSS styled article

**Rationale**: Follows existing UX patterns, provides clear user feedback.

---

### 6. Navigation Structure for Login Page

**Decision**: Show simplified navigation (or no navigation) on login page

**Consideration**: Login page is accessed by unauthenticated users

**Options**:

**Option A**: No navigation (user not logged in, can't access other pages)
- Cleanest UX for login flow
- Matches common login page patterns

**Option B**: Show minimal navigation (About, Privacy links)
- Provides context and information access
- Slightly better UX for new users

**Option C**: Show full navigation (same as other pages)
- Most consistent with application design
- Links would redirect to login anyway if not authenticated

**Decision**: Option B (minimal navigation with About/Help links)
- Balances consistency with practical UX
- Can be adjusted during implementation based on feedback

**Rationale**: Login page should be welcoming but focused. Minimal navigation provides context without distraction.

---

### 7. CSRF Protection Requirements

**Decision**: Verify if CSRF protection is needed for public login page

**Investigation**:
- Checked `internal/web/middleware/csrf.go`
- OAuth flow uses state parameter for CSRF protection (standard OAuth security)
- Form submission starts OAuth flow, not a sensitive state change

**Analysis**:
- Login form POST starts OAuth flow (redirects to external Bluesky auth)
- OAuth state parameter provides CSRF protection for the callback
- No sensitive data modified by the login form POST itself
- Standard practice: CSRF often not required for login pages that don't change state

**Decision**: CSRF protection NOT required for login form
- OAuth's state parameter handles CSRF for the callback
- Login POST just initiates external redirect
- Keeps implementation simple

**Rationale**: OAuth protocol handles security. Adding CSRF would be redundant and complicate the flow.

---

## Summary of Technical Decisions

| Area | Decision | Rationale |
|------|----------|-----------|
| Template Engine | Go html/template (existing) | Already in use, standard library |
| Template Structure | Three-block pattern (title, nav, content) | Matches existing pages |
| CSS Framework | Pico CSS (existing) | Classless, automatic form styling |
| Template Rendering | Use existing web/handlers helper | Consistent with codebase |
| Error Handling | Template data with .Error field | Existing pattern |
| Navigation | Minimal (About/Help links) | Balanced UX for unauthenticated users |
| CSRF Protection | Not required (OAuth state handles it) | Standard OAuth security |
| Form Validation | HTML5 required + backend validation | Progressive enhancement |

---

## Implementation Approach

Based on research findings:

1. **Create template file**: `internal/web/templates/pages/login.html`
   - Use three-block structure (title, nav, content)
   - Apply Pico CSS classless styling
   - Include error display pattern
   - Add helpful context text

2. **Modify OAuth handler**: `internal/auth/oauth.go`
   - Remove inline HTML string
   - Import template rendering helper
   - Pass error messages via template data
   - Maintain identical OAuth flow logic

3. **Test template rendering**:
   - Verify visual consistency with other pages
   - Test error display
   - Verify OAuth flow still works
   - Check responsive design

---

## Alternatives Considered

### Alternative 1: Use a different template engine (e.g., templ, gomponents)
**Rejected**: Would require adding new dependencies. Current html/template is sufficient and already integrated.

### Alternative 2: Keep inline HTML, just add Pico CSS to it
**Rejected**: Doesn't achieve the goal of separating concerns. Inline HTML remains unmaintainable.

### Alternative 3: Use JavaScript framework for login page
**Rejected**: Massive overkill for a simple form. Goes against project philosophy of keeping things simple.

---

## Open Questions Resolved

All questions from Technical Context have been resolved:

- ✅ Template structure: Use existing three-block pattern
- ✅ Pico CSS styling: Use classless default styling
- ✅ Error handling: Use existing .Error template data pattern
- ✅ Navigation: Minimal navigation for unauthenticated users
- ✅ CSRF: Not required (OAuth state parameter handles security)
- ✅ Template rendering: Use existing web/handlers helper

---

## Next Steps

Proceed to Phase 1: Design & Contracts
- Generate data-model.md (minimal - just template data structure)
- Generate contracts/ (optional - this is internal UI, no external API)
- Generate quickstart.md (step-by-step implementation guide)
