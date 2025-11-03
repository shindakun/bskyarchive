# Implementation Plan: Login Form Template & Styling

**Branch**: `006-login-template-styling` | **Date**: 2025-11-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/006-login-template-styling/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Migrate the login form from inline HTML in `internal/auth/oauth.go` to a proper template file at `internal/web/templates/pages/login.html`. Apply Pico CSS styling to match the rest of the application's design using the existing base template, navigation, and styling patterns. This is a straightforward UI refactoring that maintains 100% functional parity with the existing OAuth authentication flow while improving visual consistency and code maintainability.

## Technical Context

**Language/Version**: Go 1.21+ (existing project standard)
**Primary Dependencies**: Go standard library (html/template, net/http), Pico CSS (existing), bskyoauth (existing OAuth library)
**Storage**: N/A (no data storage changes, UI-only refactoring)
**Testing**: Go testing package (go test), manual UI testing for visual consistency
**Target Platform**: Web application running on user's local machine (existing deployment model)
**Project Type**: Single web application
**Performance Goals**: Page load <500ms (from success criteria), template rendering <50ms
**Constraints**: Must maintain 100% functional parity with existing OAuth flow, zero breaking changes to authentication
**Scale/Scope**: Single login page template, ~1-2 files modified (oauth.go, new login.html), minimal scope UI-only change

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### ✅ I. Data Privacy & Local-First Architecture
**Status**: PASS - No impact

This feature is a UI-only refactoring that does not affect data storage, privacy, or local-first architecture. All OAuth tokens and session management remain unchanged.

### ✅ II. Comprehensive & Accurate Archival
**Status**: PASS - No impact

No changes to archival functionality. This is purely a login UI enhancement.

### ✅ III. Multiple Export Formats
**Status**: PASS - No impact

No changes to export functionality.

### ✅ IV. Fast & Efficient Search
**Status**: PASS - No impact

No changes to search functionality.

### ✅ V. Incremental & Efficient Operations
**Status**: PASS - No impact

No changes to operational efficiency. Template rendering is negligible performance overhead (<50ms).

### ✅ Security & Privacy
**Status**: PASS - Compliant

- OAuth 2.0 flow using bskyoauth library: ✅ Preserved (no changes to OAuth logic)
- Secure session management: ✅ Preserved (no changes)
- CSRF protection: ✅ Will verify if needed for public login page
- No credential storage in plaintext: ✅ Unchanged

### ✅ Development Standards
**Status**: PASS - Compliant

- Go 1.21+ with standard library: ✅ Using html/template from stdlib
- Clear separation of concerns: ✅ Improved (separating HTML from Go code)
- HTML, Pico CSS, HTMX, Vanilla JavaScript: ✅ Using existing Pico CSS and template structure
- Testing requirements: ✅ Will add manual UI testing for visual consistency

**GATE RESULT**: ✅ **PASS** - All constitutional principles satisfied. This is a low-risk UI refactoring that improves code maintainability and visual consistency without affecting core functionality.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── auth/
│   └── oauth.go                      # MODIFIED: Remove inline HTML, use template rendering
├── web/
│   ├── handlers/
│   │   └── template.go               # POTENTIALLY MODIFIED: May need helper for login page rendering
│   ├── middleware/
│   │   └── csrf.go                   # REVIEW: Verify if CSRF needed for public login
│   ├── templates/
│   │   ├── layouts/
│   │   │   └── base.html             # EXISTING: Used by login template
│   │   ├── pages/
│   │   │   ├── export.html           # REFERENCE: Template structure to match
│   │   │   ├── dashboard.html        # REFERENCE: Navigation structure
│   │   │   └── login.html            # NEW: Login page template
│   │   └── partials/
│   │       └── nav.html              # EXISTING: Navigation component
│   └── static/
│       └── css/
│           └── pico.css              # EXISTING: CSS framework

tests/
├── integration/
│   └── login_template_test.go        # NEW: Integration test for login page rendering
└── unit/
    └── auth_test.go                  # POTENTIALLY MODIFIED: Update tests if needed
```

**Structure Decision**: Single web application using Go's standard project layout. The feature involves:
1. Creating a new template file at `internal/web/templates/pages/login.html`
2. Modifying `internal/auth/oauth.go` to use template rendering instead of inline HTML
3. Leveraging existing template infrastructure (base.html, Pico CSS, navigation partials)
4. Adding tests to verify template rendering and OAuth flow preservation

This follows the existing pattern used by other pages (export.html, dashboard.html) in the application.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitutional violations. This feature aligns with all principles and requires no complexity justification.
