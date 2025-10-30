<!--
SYNC IMPACT REPORT
==================
Version change: [INITIAL] → v1.0.0
Modified principles: N/A (initial constitution)
Added sections: All core principles (I-V), Security & Privacy, Development Standards, Governance
Removed sections: N/A
Templates requiring updates:
  ✅ .specify/templates/plan-template.md - reviewed, no updates needed
  ✅ .specify/templates/spec-template.md - reviewed, no updates needed
  ✅ .specify/templates/tasks-template.md - reviewed, no updates needed
Follow-up TODOs: None
-->

# Bluesky Personal Archive Tool Constitution

## Core Principles

### I. Data Privacy & Local-First Architecture

**All user data MUST remain under user control:**
- All data stored locally on user's machine
- No third-party services or cloud dependencies for storage
- No telemetry or analytics sent to external servers
- OAuth tokens and sensitive data encrypted at rest
- User controls all data export, deletion, and sharing

**Rationale**: Users entrust us with their social media history. Privacy is not negotiable. Local-first architecture ensures users maintain complete ownership and control of their data without vendor lock-in.

### II. Comprehensive & Accurate Archival

**The system MUST preserve complete user history:**
- Archive 100% of user posts with timestamps and engagement metrics
- Preserve embedded media (images, videos) with alt text
- Maintain thread context (replies, quote posts, conversations)
- Capture profile snapshots and follower/following history over time
- Support incremental backups to efficiently fetch only new content
- Implement retry logic and error handling for robust data collection

**Rationale**: An incomplete archive is a broken archive. Users rely on this tool to preserve their digital history accurately and comprehensively.

### III. Multiple Export Formats

**Archives MUST be accessible in multiple formats:**
- JSON for structured data and programmatic access
- Markdown for human-readable text with frontmatter
- HTML for static websites with built-in search
- CSV for spreadsheet compatibility
- All exports maintain data integrity and relationships

**Rationale**: Different users have different needs. Supporting multiple formats ensures archives remain useful across various contexts (backup, analysis, presentation, migration).

### IV. Fast & Efficient Search

**Users MUST be able to find content quickly:**
- Full-text search across all posts in <100ms for typical queries
- Filter by date range, engagement metrics, and media presence
- Support tag-based organization and thread reconstruction
- Maintain search indexes (SQLite + full-text search)
- Web interface provides intuitive search and browse capabilities

**Rationale**: An archive without search is just data hoarding. Fast, comprehensive search makes archives valuable and usable.

### V. Incremental & Efficient Operations

**The system MUST respect resources and scale efficiently:**
- Incremental backups fetch only new content
- Support archives of 10,000+ posts efficiently
- Generate exports in <30 seconds for typical archives
- Configurable sync intervals (hourly, daily, weekly)
- Background job scheduler for automation without manual intervention
- Graceful handling of rate limits and API constraints

**Rationale**: Efficient operations respect both user time and system resources. Scalability ensures the tool remains useful as archives grow over years.

## Security & Privacy

**Security measures MUST be comprehensive:**
- OAuth 2.0 flow using bskyoauth library for secure authentication
- Secure session management with token refresh handling
- Encrypt sensitive data at rest; use system keyring where possible
- Optional basic authentication for web interface
- CSRF protection on all state-changing operations
- Rate limiting on API endpoints to prevent abuse
- No credential storage in plaintext

**Rationale**: Security protects user privacy. Even though data is local, authentication tokens and session management require careful handling.

## Development Standards

**Code quality and maintainability standards:**
- Go 1.21+ with standard library practices
- Clear separation of concerns: auth, collector, storage, search, exporter, scheduler, web
- Database migrations for schema versioning
- CLI built with cobra for consistent user experience
- Web framework (net/http stdlib)
- Full-text search with bleve
- Comprehensive error handling with retry logic
- Configuration management via YAML/JSON files
- HTML, Picocss, HTMX, and Vanilla JavaScript

**Testing requirements:**
- Unit tests for business logic
- Integration tests for AT Protocol interactions
- Contract tests for API endpoints
- Archive verification utilities
- Performance benchmarks for search and export operations

**Documentation requirements:**
- User-facing documentation for CLI commands
- Configuration examples and best practices
- API documentation for programmatic access
- Development setup and contribution guidelines

**Rationale**: Maintainable code ensures long-term viability. Clear standards make collaboration and contributions straightforward.

## Governance

**Constitution Authority:**
- This constitution supersedes all other project practices
- All design decisions, PRs, and reviews MUST verify compliance with these principles
- Complexity beyond these principles MUST be explicitly justified in plan.md

**Amendment Procedure:**
1. Proposed changes documented with rationale
2. Impact analysis across all templates and code
3. Update .specify/memory/constitution.md with version increment
4. Propagate changes to dependent templates and documentation
5. Document migration path if backward incompatible

**Versioning Policy:**
- **MAJOR**: Backward incompatible governance/principle removals or redefinitions
- **MINOR**: New principle/section added or materially expanded guidance
- **PATCH**: Clarifications, wording, typo fixes, non-semantic refinements

**Compliance Review:**
- All feature specifications MUST reference constitution principles
- Implementation plans MUST include constitution check section
- Task lists MUST reflect principle-driven requirements
- Code reviews verify adherence to privacy, efficiency, and security standards

**Runtime Guidance:**
- Development guidance follows principles established here
- See bskyarchive.md for feature specifications and implementation phases
- Refer to .specify/templates/ for structured workflows

**Version**: 1.0.0 | **Ratified**: 2025-10-30 | **Last Amended**: 2025-10-30
