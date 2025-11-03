# Feature Specification: Login Form Template & Styling

**Feature Branch**: `006-login-template-styling`
**Created**: 2025-11-02
**Status**: Draft
**Input**: User description: "Move the login form from inline HTML in internal/auth/oauth.go to a proper template file (internal/web/templates/pages/login.html) styled with Pico CSS to match the rest of the application's design. The form should use the same base template, navigation, and styling as other pages like export.html"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Login Form Displays with Consistent Styling (Priority: P1) 🎯 MVP

Users visit the login page and see a professionally styled form that matches the rest of the application's design language, providing a cohesive user experience from first impression.

**Why this priority**: First impression matters. The login page is the entry point for all users. Having it match the application's design establishes credibility and professionalism. This is the minimum viable change - a properly styled login form.

**Independent Test**: Navigate to the login page, verify that the form uses Pico CSS styling, includes the application header/navigation structure, and visually matches the design of other pages like the export page.

**Acceptance Scenarios**:

1. **Given** a user visits the login page, **When** the page loads, **Then** the login form displays with Pico CSS styling including proper form controls, buttons, and spacing
2. **Given** a user views the login page, **When** comparing it to other application pages (dashboard, export), **Then** the page uses the same base template, header, navigation structure, and color scheme
3. **Given** a user resizes their browser window, **When** viewing the login page, **Then** the form is responsive and maintains proper layout on mobile, tablet, and desktop screen sizes

---

### User Story 2 - Login Form Maintains Full Functionality (Priority: P1) 🎯 MVP

The templated login form preserves all existing OAuth functionality, allowing users to successfully authenticate with their Bluesky handle.

**Why this priority**: This is a non-negotiable requirement. The refactoring must not break existing authentication. Without working login, the application is unusable.

**Independent Test**: Enter a valid Bluesky handle in the login form, submit it, verify that the OAuth flow initiates correctly and the user is redirected to Bluesky for authentication, then successfully returned to the application.

**Acceptance Scenarios**:

1. **Given** a user enters a valid Bluesky handle, **When** they submit the login form, **Then** the OAuth flow initiates and redirects to Bluesky's authorization page
2. **Given** a user submits the form without entering a handle, **When** the form is validated, **Then** an error message displays indicating the handle is required
3. **Given** a user enters an invalid handle format, **When** they submit the form, **Then** appropriate error feedback is shown
4. **Given** a user completes the Bluesky OAuth flow, **When** they are redirected back to the application, **Then** they are successfully authenticated and see the dashboard

---

### User Story 3 - Login Page Shows Helpful Context (Priority: P2)

Users see helpful information on the login page explaining what the application does and why they should sign in, improving the onboarding experience.

**Why this priority**: While not strictly necessary for the template migration, adding contextual information improves user experience and reduces confusion for first-time users. This is a nice-to-have enhancement that leverages the template refactoring.

**Independent Test**: Visit the login page as a new user, verify that descriptive text explains the application's purpose and what signing in will allow the user to do.

**Acceptance Scenarios**:

1. **Given** a new user visits the login page, **When** they read the page content, **Then** they see a brief description of the application's purpose (archiving Bluesky posts)
2. **Given** a user views the login page, **When** looking at the form area, **Then** they see helpful text explaining what a Bluesky handle is (e.g., "Enter your Bluesky handle: user.bsky.social")
3. **Given** a user is on the login page, **When** they want to learn more before signing in, **Then** they see a link to an "About" page or privacy information

---

### Edge Cases

- What happens when the template file is missing or corrupted? (Handler should return an error page rather than crash)
- How does the system handle extremely long Bluesky handles? (Form should validate maximum length)
- What happens if the OAuth client configuration is invalid? (Display user-friendly error message, not internal error details)
- How does the login page render with JavaScript disabled? (Form should still be functional, graceful degradation)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST render the login form using a template file located at `internal/web/templates/pages/login.html`
- **FR-002**: Login page MUST use the same base template structure as other application pages (header, navigation, content area, footer)
- **FR-003**: Login form MUST apply Pico CSS styling to all form elements (input fields, buttons, labels, validation messages)
- **FR-004**: Login form MUST accept a Bluesky handle input with appropriate HTML5 validation attributes (required, type="text", placeholder text)
- **FR-005**: Login form MUST submit via POST to the existing OAuth handler endpoint
- **FR-006**: System MUST preserve all existing OAuth flow functionality (handle validation, authorization redirect, callback handling)
- **FR-007**: Login page MUST display appropriate error messages when login fails (e.g., invalid handle, OAuth error, network error)
- **FR-008**: Login page MUST be responsive and display correctly on mobile, tablet, and desktop screen sizes
- **FR-009**: Login form MUST include CSRF protection using the existing CSRF token mechanism (if applicable to public login page)
- **FR-010**: System MUST remove inline HTML from `internal/auth/oauth.go` and replace with template rendering

### Key Entities

- **LoginPage**: Represents the login page view with properties including page title, form action URL, CSRF token, error messages, and contextual help text
- **LoginForm**: Represents the form structure with handle input field, submit button, validation rules, and error display areas

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Login page loads and displays with consistent styling in under 500ms on standard connections
- **SC-002**: Login form maintains 100% functional parity with existing inline HTML implementation (all OAuth flows work identically)
- **SC-003**: Visual consistency score of 95%+ when comparing login page design elements (colors, fonts, spacing, button styles) to other application pages
- **SC-004**: Login form passes responsive design testing on screen sizes from 320px (mobile) to 1920px (desktop) wide without layout breaking
- **SC-005**: Code maintainability improves: login form HTML is separated from Go code, enabling designers to modify templates without touching backend code

## Assumptions

- Application already uses Pico CSS framework consistently across all pages
- Base template system is established with navigation, header, and content blocks defined
- Existing OAuth flow in `internal/auth/oauth.go` is working correctly and only the form rendering needs to be changed
- CSRF protection middleware is already in place for form submissions (or not required for public login page)
- Template rendering infrastructure (template parsing, execution, error handling) is already implemented
- No changes to OAuth configuration or authentication logic are required
- Current placeholder text "user.bsky.social" is sufficient for handle input guidance

## Out of Scope

- Changes to OAuth authentication logic or flow sequence
- Multi-factor authentication or additional security measures
- Alternative authentication methods (email/password, SSO, etc.)
- Password reset or account recovery flows (Bluesky OAuth handles this externally)
- User registration functionality (handled by Bluesky)
- Internationalization or multi-language support for login page
- Advanced form features like autocomplete handle suggestions or handle validation before submission
- Analytics or tracking on login page
- Dark mode or theme switching (unless already implemented application-wide)
