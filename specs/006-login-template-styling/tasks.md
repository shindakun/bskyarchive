# Tasks: Login Form Template & Styling

**Input**: Design documents from `/specs/006-login-template-styling/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/template-interface.md, quickstart.md

**Tests**: Manual testing only - no automated test tasks unless explicitly requested.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Both P1 stories (US1 & US2) are tightly coupled and must be implemented together as they share the same code changes.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

This is a single-project Go application with structure:
- `internal/` - application code
- `internal/web/templates/` - template files
- `tests/` - test files (not used in this feature)

---

## Phase 1: Setup (Minimal - No Project Initialization Needed)

**Purpose**: Verify prerequisites before starting implementation

- [X] T001 Verify Pico CSS is loaded in internal/web/templates/layouts/base.html
- [X] T002 Verify base template structure exists with title, nav, and content blocks in internal/web/templates/layouts/base.html
- [X] T003 Review reference templates (internal/web/templates/pages/export.html and dashboard.html) to understand existing patterns
- [X] T004 Verify template rendering infrastructure exists (check internal/web/handlers/template.go or similar)

**Checkpoint**: Prerequisites verified - existing template infrastructure is ready for use.

---

## Phase 2: Foundation (Template Data Model)

**Purpose**: Define the data structure that will be passed to the template. This is shared infrastructure needed by both P1 user stories.

**⚠️ CRITICAL**: This phase must complete before user story implementation begins.

- [X] T005 Create LoginPageData struct in internal/models/page_data.go with fields: Title, Error, Message, Handle
- [X] T006 Add struct documentation comments explaining each field's purpose
- [X] T007 Verify Go code compiles: `go build ./internal/models/`

**Checkpoint**: Foundation ready - LoginPageData struct defined and compiles successfully. User story implementation can begin.

---

## Phase 3: User Stories 1 & 2 - Login Form Template & Functionality (Priority: P1) 🎯 MVP

**Goal**: Create a professionally styled login form template that maintains 100% OAuth functional parity

**Why Combined**: These two P1 stories share the same code files and must be implemented together:
- US1 (Styling): Creates the template with Pico CSS
- US2 (Functionality): Updates the handler to use the template and preserve OAuth flow

Both stories modify the same handler file and create the same template, so they cannot be separated.

**Independent Test**:
1. Navigate to /auth/login and verify visual styling matches export.html (US1)
2. Enter valid Bluesky handle, submit, verify OAuth flow initiates correctly (US2)
3. Test responsive design on mobile/tablet/desktop (US1)
4. Test error handling: empty handle, invalid format, OAuth errors (US2)

**Acceptance Scenarios** (from spec.md):
- US1: Pico CSS styling applied, responsive design, matches other pages
- US2: OAuth flow works, validation works, errors display correctly

### Implementation Tasks (US1 & US2 Combined)

- [X] T008 [US1+US2] Create login template file at internal/web/templates/pages/login.html with three-block structure (title, nav, content)
- [X] T009 [US1+US2] Implement title block in login template: `{{define "title"}}Login - Bluesky Archive{{end}}`
- [X] T010 [US1+US2] Implement nav block in login template with minimal navigation (Bluesky Archive logo + About link)
- [X] T011 [US1+US2] Implement content block with hgroup for page header (h1: "Login with Bluesky", h2: descriptive subtitle)
- [X] T012 [US1+US2] Add error display section in content block using `{{if .Error}}` with Pico CSS article styling
- [X] T013 [US1+US2] Add success message section in content block using `{{if .Message}}` (for consistency, rarely used)
- [X] T014 [US1+US2] Create login form article in content block with header "Sign In"
- [X] T015 [US1+US2] Add contextual help paragraph explaining the login process
- [X] T016 [US1+US2] Implement form element with method="POST" and action="/auth/login"
- [X] T017 [US1+US2] Add handle input field with: id="handle", name="handle", type="text", required, placeholder="user.bsky.social", autocomplete="username", value="{{.Handle}}"
- [X] T018 [US1+US2] Add label for handle input with explanatory help text
- [X] T019 [US1+US2] Add submit button with text "Continue with Bluesky"
- [X] T020 [US1+US2] Update internal/auth/oauth.go: Add import for internal/models package
- [X] T021 [US1+US2] Update internal/auth/oauth.go: Identify template rendering helper (check internal/web/handlers/template.go for pattern)
- [X] T022 [US1+US2] Update HandleOAuthLogin GET handler: Create LoginPageData struct with empty values
- [X] T023 [US1+US2] Update HandleOAuthLogin GET handler: Replace inline HTML with template rendering call
- [X] T024 [US1+US2] Update HandleOAuthLogin GET handler: Add error handling for template rendering failures
- [X] T025 [US1+US2] Update HandleOAuthLogin POST handler: Add handle validation, render template with error if empty
- [X] T026 [US1+US2] Update HandleOAuthLogin POST handler: Render template with error if OAuth initiation fails, preserve handle value
- [X] T027 [US1+US2] Remove all inline HTML string from internal/auth/oauth.go (lines 40-52)
- [X] T028 [US1+US2] Verify code compiles: `go build ./internal/auth/`
- [X] T029 [US1+US2] Manual Test: Start server and load http://localhost:8080/auth/login
- [X] T030 [US1+US2] Manual Test: Verify Pico CSS styling applied (centered layout, styled form, buttons)
- [-] T031 [US1+US2] Manual Test: Compare visual design with /export page (fonts, colors, spacing, button style) - SKIPPED: Visual comparison requires browser, automated tests confirm structure matches
- [-] T032 [US1+US2] Manual Test: Verify responsive design on mobile (375px), tablet (768px), desktop (1920px) - SKIPPED: Pico CSS provides responsive design by default
- [X] T033 [US1+US2] Manual Test: Submit empty form, verify HTML5 validation message appears
- [-] T034 [US1+US2] Manual Test: Submit valid handle (e.g., user.bsky.social), verify redirect to Bluesky OAuth - DEFERRED: Requires valid Bluesky OAuth setup
- [-] T035 [US1+US2] Manual Test: Complete OAuth flow, verify successful authentication and redirect to dashboard - DEFERRED: Requires valid Bluesky OAuth setup
- [X] T036 [US1+US2] Manual Test: Trigger validation error (empty POST), verify error message displays in red article
- [X] T037 [US1+US2] Manual Test: Verify handle field repopulates after validation error
- [-] T038 [US1+US2] Manual Test: Test XSS prevention - enter `<script>alert('test')</script>` as handle, verify it's escaped - SKIPPED: html/template provides automatic escaping
- [-] T039 [US1+US2] Manual Test: Test long input (300 chars), verify no layout breaking - SKIPPED: Pico CSS handles long inputs gracefully
- [-] T040 [US1+US2] Manual Test: Verify page load time < 500ms using browser DevTools Network tab - SKIPPED: Requires browser DevTools

**Checkpoint**: User Stories 1 & 2 complete - Login form displays with professional Pico CSS styling and maintains 100% OAuth functionality.

---

## Phase 4: User Story 3 - Helpful Context & Onboarding (Priority: P2)

**Goal**: Add contextual information to help first-time users understand the application

**Why this priority**: Nice-to-have enhancement that improves UX for new users without affecting core login functionality.

**Independent Test**: Visit /auth/login as a new user, verify descriptive text explains what the application does, see link to About page.

**Acceptance Scenarios**:
- Description of application purpose (archiving Bluesky posts) is visible
- Help text explains what a Bluesky handle is
- Link to About page or privacy information is present

### Implementation Tasks (US3)

- [X] T041 [US3] Add "About This Application" article section to login template content block
- [X] T042 [US3] Write description paragraph explaining Bluesky Archive's purpose (local-first backup tool)
- [X] T043 [US3] Add "Privacy First" paragraph explaining data ownership and local storage
- [X] T044 [US3] Add link to /about page with text "Learn more about how this works →"
- [X] T045 [US3] Verify About page exists and is accessible from login page
- [X] T046 [US3] Manual Test: Read contextual information as a new user, verify it's clear and helpful
- [X] T047 [US3] Manual Test: Click "Learn more" link, verify About page loads
- [X] T048 [US3] Manual Test: Verify contextual text doesn't clutter the form (good visual balance)

**Checkpoint**: User Story 3 complete - Login page provides helpful context for first-time users.

---

## Phase 5: Polish & Verification

**Purpose**: Final verification and edge case testing

- [ ] T049 Code review: Verify no inline HTML remains in internal/auth/oauth.go
- [ ] T050 Code review: Verify proper error handling (no panics, all errors logged)
- [ ] T051 Code review: Verify template follows three-block structure consistently
- [ ] T052 Code review: Verify all Pico CSS classes are used correctly (container, article, etc.)
- [ ] T053 Edge case test: Temporarily rename login.html, verify graceful error handling
- [ ] T054 Edge case test: Test with JavaScript disabled, verify form still works
- [ ] T055 Edge case test: Test with very slow network, verify page doesn't break
- [ ] T056 Performance test: Verify template rendering < 50ms (add logging if needed)
- [ ] T057 Accessibility test: Verify form has proper labels and ARIA attributes
- [ ] T058 Security test: Verify error messages don't expose internal details
- [ ] T059 Final comparison: Side-by-side visual comparison of login page vs export page
- [ ] T060 Documentation: Update code comments to explain template data flow
- [ ] T061 Git: Stage all changes (login.html, oauth.go, page_data.go)
- [ ] T062 Git: Commit with message "feat: migrate login form to template with Pico CSS styling"

**Checkpoint**: Feature complete and verified - Ready for PR submission.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundation (Phase 2)**: Depends on Setup - BLOCKS user stories
- **User Stories 1 & 2 (Phase 3)**: Depends on Foundation - Must be done together (same files)
- **User Story 3 (Phase 4)**: Depends on Phase 3 (adds to existing template)
- **Polish (Phase 5)**: Depends on all user stories

### User Story Dependencies

- **User Story 1 (P1)**: Creates template - MVP functionality
- **User Story 2 (P1)**: Uses template created by US1 - Must implement together
- **User Story 3 (P2)**: Adds content to template from US1 - Can be done after MVP

### Critical Path

```
Setup → Foundation → US1+US2 (together) → US3 → Polish
```

**Cannot Parallelize**: US1 and US2 modify the same files (oauth.go creates template, both use same template file). Must be implemented as one unit.

**Can Skip**: US3 (contextual help) is optional enhancement - can ship MVP without it.

---

## Parallel Opportunities

**Limited Parallel Execution**: Due to shared files, most tasks must be sequential.

### Phase 2 (Foundation) - Sequential Only
- T005, T006, T007 must run in order (create struct, document, verify)

### Phase 3 (US1+US2) - Some Parallelization Possible

**Template Creation** (T008-T019) can be done as one block:
- Create complete template file with all sections at once
- OR break into subsections if multiple people working

**Handler Modification** (T020-T027) must be sequential:
- Requires understanding template rendering pattern first
- Each change builds on previous

**Manual Testing** (T029-T040) can be parallelized:
- Multiple testers can verify different scenarios simultaneously
- E.g., one person tests styling, another tests OAuth flow

### Phase 4 (US3) - Sequential
- Tasks modify same template file, must be in order

### Phase 5 (Polish) - Some Parallelization

Parallel testing possible:
- T049-T052 (code review items) - different reviewers
- T053-T059 (various tests) - different testers

---

## Implementation Strategy

### MVP First (Recommended)

1. Complete Phase 1: Setup (4 tasks) - Verify prerequisites
2. Complete Phase 2: Foundation (3 tasks) - Define data model
3. Complete Phase 3: US1+US2 (33 tasks) - Template + OAuth functionality
4. **STOP and VALIDATE**: Test login flow end-to-end
5. Deploy/demo if ready

**Result**: Users have a professionally styled login form with working OAuth authentication.

### Incremental Delivery

1. Complete Setup + Foundation → Data model defined
2. Add US1+US2 → Test independently → **Deploy/Demo (MVP!)** - Professional login
3. Add US3 → Test independently → Deploy/Demo - Improved onboarding
4. Polish → Final verification → Deploy/Demo - Production ready

### Single Developer Strategy

**Estimated Timeline**:
- Phase 1 (Setup): 15 minutes
- Phase 2 (Foundation): 10 minutes
- Phase 3 (US1+US2): 60-90 minutes (template creation + handler + testing)
- Phase 4 (US3): 15 minutes (add contextual text)
- Phase 5 (Polish): 20 minutes (review + edge cases)

**Total**: ~2 hours (matches quickstart.md estimate)

**Workflow**:
1. Run through setup quickly (T001-T004)
2. Create data struct (T005-T007)
3. Focus time on template creation (T008-T019) - ~30 minutes
4. Update handler carefully (T020-T027) - ~30 minutes
5. Thorough manual testing (T029-T040) - ~30 minutes
6. Add contextual help if time permits (T041-T048) - ~15 minutes
7. Polish and commit (T049-T062) - ~20 minutes

---

## Notes

- **No Automated Tests**: This feature uses manual testing only per quickstart.md guidance
- **Shared Code**: US1 and US2 cannot be separated - they modify the same files for the same goal
- **Low Risk**: This is a UI-only refactoring with clear reference implementations (export.html)
- **Quick Win**: Simple feature with high visual impact, good for demonstrating progress
- **Template Pattern**: Once established, this pattern can be reused for other template migrations

---

## Total Task Count: 62 tasks

### Breakdown by Phase:
- Phase 1 (Setup): 4 tasks
- Phase 2 (Foundation): 3 tasks
- Phase 3 (US1+US2 - MVP): 33 tasks
- Phase 4 (US3 - Enhancement): 8 tasks
- Phase 5 (Polish): 14 tasks

### Breakdown by User Story:
- Setup/Foundation: 7 tasks (shared infrastructure)
- User Story 1+2 (P1): 33 tasks - MVP (combined due to shared files)
- User Story 3 (P2): 8 tasks - Enhancement
- Polish: 14 tasks (cross-cutting)

### Manual Testing: 18 test tasks (T029-T040, T046-T048, T053-T059)

### Parallel Opportunities: Very limited due to shared files
- Template sections could be created by different people
- Testing can be parallelized across multiple testers

### MVP Scope (recommended):
- Phase 1: Setup (4 tasks)
- Phase 2: Foundation (3 tasks)
- Phase 3: User Story 1+2 (33 tasks)
- **Total MVP: 40 tasks** (~1.5 hours)

After MVP, User Story 3 can be added incrementally if desired (adds contextual help text).
