<!--
Sync Impact Report:
Version change: 0.0.0 → 1.0.0
Modified principles: N/A (new constitution)
Added sections: Code Quality Standards, Testing Standards, User Experience Consistency, Performance Requirements, Developer Experience
Removed sections: N/A
Templates requiring updates:
  ✅ plan-template.md (Constitution Check section updated)
  ✅ spec-template.md (aligned with testing standards)
  ⚠ tasks-template.md (pending review for task categorization)
  ⚠ commands/*.md (pending review for outdated references)
Follow-up TODOs: None
-->

# Fucci Constitution

## Core Principles

### I. Code Quality Standards (NON-NEGOTIABLE)

All code MUST meet minimum quality standards before merge. Code quality is measured through automated tools, peer review, and maintainability metrics.

**Mandatory Requirements:**

- TypeScript strict mode enabled for all TypeScript/JavaScript code
- ESLint configuration with zero warnings allowed
- Prettier formatting enforced with pre-commit hooks
- Cyclomatic complexity ≤ 10 for all functions
- Function length ≤ 50 lines (exceptions require justification)
- Meaningful variable and function names (no abbreviations unless domain-standard)
- Comprehensive inline documentation for public APIs
- No commented-out code in production branches

**Rationale**: High code quality reduces bugs, improves maintainability, and accelerates development velocity for the entire team.

### II. Testing Standards (NON-NEGOTIABLE)

Test-Driven Development (TDD) is mandatory for all new features. Existing code must achieve minimum test coverage before modification.

**Mandatory Requirements:**

- Unit tests: ≥ 80% code coverage for business logic
- Integration tests: All API endpoints and database interactions
- End-to-end tests: Critical user journeys (P1 user stories)
- Test files co-located with source code
- Tests MUST be independent and deterministic
- Mock external dependencies (APIs, databases, file systems)
- Test data factories for consistent test setup
- Performance tests for operations > 100ms

**Rationale**: Comprehensive testing prevents regressions, enables confident refactoring, and serves as living documentation.

### III. User Experience Consistency

All user-facing interfaces MUST follow established design patterns and provide consistent, intuitive experiences across platforms.

**Mandatory Requirements:**

- Design system compliance for all UI components
- Consistent navigation patterns across mobile and web
- Loading states for all async operations
- Error handling with user-friendly messages
- Accessibility compliance (WCAG 2.1 AA minimum)
- Responsive design for all screen sizes
- Consistent typography, spacing, and color usage
- Progressive enhancement for core functionality

**Rationale**: Consistent UX reduces cognitive load, improves user satisfaction, and builds trust in the application.

### IV. Performance Requirements

All features MUST meet performance benchmarks to ensure responsive user experience and efficient resource utilization.

**Mandatory Requirements:**

- Mobile app: < 3s initial load time, < 1s navigation between screens
- Web app: < 2s initial load time, < 500ms for user interactions
- API responses: < 200ms p95 latency for standard operations
- Database queries: < 100ms p95 execution time
- Bundle size: Mobile app < 50MB, Web app < 2MB initial bundle
- Memory usage: < 200MB for mobile app, < 100MB for web app
- Battery impact: Minimal background processing on mobile
- Network efficiency: Implement caching and request optimization

**Rationale**: Performance directly impacts user retention and satisfaction. Poor performance creates negative user experiences and increases infrastructure costs.

### V. Developer Experience

The codebase MUST be approachable for new developers and support efficient development workflows.

**Mandatory Requirements:**

- Comprehensive README with setup instructions
- Clear project structure with logical organization
- Consistent naming conventions across all languages
- API documentation with examples
- Development environment setup automation
- Clear error messages and debugging information
- Modular architecture with single responsibility principle
- Dependency management with lock files
- Environment configuration management
- Code review templates and guidelines

**Rationale**: Good developer experience reduces onboarding time, improves team productivity, and enables faster feature delivery.

## Development Workflow

### Code Review Process

All code changes MUST pass through peer review before merge:

- Minimum 2 approvals for production changes
- Automated testing must pass before review
- Security review for authentication/authorization changes
- Performance impact assessment for database/API changes
- Documentation updates required for public API changes

### Quality Gates

The following gates MUST pass before deployment:

- All tests passing (unit, integration, e2e)
- Code coverage thresholds met
- Security vulnerability scan clean
- Performance benchmarks met
- Accessibility compliance verified
- Bundle size within limits

### Release Management

- Semantic versioning (MAJOR.MINOR.PATCH)
- Automated deployment pipelines
- **Mobile distribution**: Automated builds and distribution for the mobile app:
  - **iOS**: Automatic deploy to TestFlight (or equivalent testing channel) on release triggers (e.g. tag or branch), so testers receive builds without manual upload.
  - **Android**: Optional automatic distribution to an internal testing track (e.g. Google Play internal testing or EAS internal distribution).
- Feature flags for gradual rollouts
- Rollback procedures documented and tested
- Monitoring and alerting for production issues

## Technology Standards

### Frontend (React Native/Next.js)

- TypeScript for type safety
- React Native with Expo for mobile development
- Next.js with App Router for web development
- React Navigation for mobile navigation
- React Query for data fetching and caching
- React Hook Form for form management

### Backend (Go)

- Go 1.22+ with standard library preference
- Gin framework for HTTP routing
- GORM for database operations
- JWT for authentication
- Structured logging with context
- Graceful shutdown handling

### Infrastructure

- AWS for cloud hosting
- Terraform for Infrastructure as Code
- Docker for containerization
- GitHub Actions for CI/CD
- PostgreSQL for primary database
- S3 for media storage

## Governance

This constitution supersedes all other development practices and MUST be followed by all team members. Amendments require:

1. **Proposal**: Document the proposed change with rationale
2. **Review**: Team discussion and impact assessment
3. **Approval**: Consensus from technical leads
4. **Implementation**: Update all affected templates and documentation
5. **Communication**: Team notification of changes

**Compliance**: All pull requests and code reviews MUST verify constitution compliance. Violations require justification or remediation before merge.

**Version**: 1.0.0 | **Ratified**: 2025-01-23 | **Last Amended**: 2025-01-23
