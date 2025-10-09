# Documentation

Official documentation for LDAP Selfservice Password Changer.

## Available Documentation

### [API Reference](api-reference.md)

Complete JSON-RPC API specification including:

- Request/response formats
- Available methods
- Validation rules
- Error handling
- Security considerations

### [Development Guide](development-guide.md)

Setup instructions and development workflows:

- Prerequisites and initial setup
- Development mode (hot reload)
- Production build
- Configuration reference
- Common development tasks
- Docker development
- Troubleshooting

### [Testing Guide](testing-guide.md)

Testing strategies and recommendations:

- Current test coverage
- Unit testing (Go and TypeScript)
- Integration testing
- E2E testing with Playwright
- Feature-specific testing (density toggle)
- Test organization
- Coverage analysis

### [Onboarding Guide](onboarding.md)

Progressive 5-day learning path for new developers:

- Day-by-day onboarding checklist
- Prerequisites and setup verification
- Code structure exploration
- First contribution guidance
- Knowledge checkpoints
- Role-specific learning paths

### [Documentation Maintenance](maintenance.md)

Documentation update workflows and maintenance procedures:

- When to update documentation
- Documentation update workflows
- Automated regeneration process
- Quality checklists
- File-specific maintenance guidance
- Maintenance schedules

### [Accessibility Guide](accessibility.md)

WCAG 2.2 AAA compliance and testing:

- Accessibility features overview
- WCAG 2.2 compliance matrix
- Adaptive dark mode and density
- Screen reader testing procedures
- System preference testing
- Keyboard navigation verification

### [Architecture](architecture.md)

System architecture overview with diagrams.

### [Code Structure Reference](code-structure.md)

Detailed package-level documentation:

- Internal package overview
- Public API surfaces
- Test coverage by package
- Security features
- Code examples and patterns
- Extension guides

### [Deployment Guide](deployment.md)

Production deployment and operations:

- Quick start (Docker, Kubernetes, bare metal)
- Environment variable reference
- LDAP and SMTP configuration
- Reverse proxy setup (nginx, Traefik, Apache)
- Security hardening
- Monitoring and logging
- Troubleshooting

### [Security Documentation](security.md)

Security architecture and threat model:

- Threat model and attack scenarios
- Security controls and mitigations
- OWASP Top 10 compliance
- Cryptography and secrets management
- Container security
- Security testing procedures
- Compliance considerations (GDPR, HIPAA, SOC 2)

### Architecture Decision Records (ADRs)

Design decisions and their rationale:

- [ADR-0001: Standardize Form Field Names](adr/0001-standardize-form-field-names.md) - Password manager compatibility
- [ADR-0002: Password Reset Functionality](adr/0002-password-reset-functionality.md) - Email-based password recovery

---

## Quick Start by Role

### Developers

1. **New to the project?** Follow the [Onboarding Guide](onboarding.md) 5-day learning path
2. **Quick setup?** See [Development Guide: Initial Setup](development-guide.md#initial-setup)
3. **Understanding the code?** Check [Code Structure Reference](code-structure.md)
4. **Need API details?** Review [API Reference](api-reference.md)
5. **Writing tests?** See [Testing Guide](testing-guide.md)

### DevOps/SRE

1. **Deploying to production?** Start with [Deployment Guide](deployment.md)
2. **Security hardening?** See [Security Documentation](security.md)
3. **System architecture?** Check [Architecture](architecture.md)

### QA/Accessibility

1. **Accessibility testing?** See [Accessibility Guide](accessibility.md)
2. **Test coverage?** Check [Testing Guide](testing-guide.md)

### Architects

1. **Design decisions?** Review [Architecture Decision Records](adr/)
2. **System design?** See [Architecture](architecture.md)
3. **Security architecture?** Check [Security Documentation](security.md)

## Contributing

See main [README.md](../README.md) for contributing guidelines.

---

_Project documentation maintained by the development team_
