# Contributing to LDAP Selfservice Password Changer

Thank you for your interest in contributing! This document provides guidelines for contributing to this project.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Architecture Decision Records](#architecture-decision-records)
- [Community](#community)

---

## Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inclusive environment for all contributors, regardless of background, experience level, gender identity, sexual orientation, disability, personal appearance, race, ethnicity, age, religion, or nationality.

### Expected Behavior

- Use welcoming and inclusive language
- Be respectful of differing viewpoints and experiences
- Gracefully accept constructive criticism
- Focus on what is best for the community
- Show empathy towards other community members

### Unacceptable Behavior

- Harassment, discrimination, or intimidation
- Trolling, insulting/derogatory comments, and personal attacks
- Public or private harassment
- Publishing others' private information without permission
- Other conduct which could reasonably be considered inappropriate

### Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be reported by opening an issue or contacting the project maintainers. All complaints will be reviewed and investigated promptly and fairly.

---

## Getting Started

### Prerequisites

- **Go** 1.25 or higher
- **Node.js** 24.x or higher
- **pnpm** 10.18 or higher (install via `corepack enable`)
- **Git** for version control
- **Docker** (optional, for testing with LDAP)

### Initial Setup

1. **Fork and clone the repository**:

   ```bash
   git clone https://github.com/YOUR-USERNAME/ldap-selfservice-password-changer.git
   cd ldap-selfservice-password-changer
   ```

2. **Install dependencies**:

   ```bash
   pnpm install
   ```

3. **Configure environment**:

   ```bash
   cp .env.local.example .env.local
   # Edit .env.local with your LDAP/SMTP settings
   ```

4. **Start development server**:

   ```bash
   pnpm dev
   ```

5. **Verify setup**:
   - Application runs on `http://localhost:3000` (default)
   - Hot reload works for TypeScript and Go changes

For detailed setup instructions, see the [Development Guide](docs/development-guide.md).

---

## Development Workflow

### Branching Strategy

We use a feature branch workflow:

1. **Create a feature branch** from `main`:

   ```bash
   git checkout main
   git pull origin main
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** with descriptive commits:

   ```bash
   git add .
   git commit -m "feat: add password strength indicator"
   ```

3. **Push to your fork**:

   ```bash
   git push -u origin feature/your-feature-name
   ```

4. **Open a Pull Request** against `main`

### Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types**:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, no logic change)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements

**Examples**:

```bash
feat(reset): add email-based password reset
fix(validators): correct uppercase character detection
docs(api): update JSON-RPC examples
test(ratelimit): add concurrent request tests
refactor(rpc): extract token validation logic
```

---

## Code Standards

### Go Code

**Style Guide**:

- Use `gofmt` for formatting (automatic)
- Follow [Effective Go](https://go.dev/doc/effective_go) principles
- Keep functions focused (single responsibility)
- Use descriptive names (avoid abbreviations)

**GoDoc Comments**:

```go
// GenerateToken generates a cryptographically secure random token.
// Returns a base64 URL-safe encoded string of 32 random bytes (256 bits).
func GenerateToken() (string, error) {
    // Implementation
}
```

**Error Handling**:

```go
// ✅ Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to connect to LDAP: %w", err)
}

// ❌ Bad: Return raw errors
if err != nil {
    return err
}
```

**Package Structure**:

- Keep packages focused on single responsibility
- Internal packages in `internal/` (not exported)
- Add package documentation in `doc.go` or first file

### TypeScript/JavaScript Code

**Style Guide**:

- Use TypeScript strict mode (enforced in `tsconfig.json`)
- Use `prettier` for formatting
- No `any` types (use proper type annotations)
- Use ES modules with `.js` extensions in imports

**JSDoc Comments** (for exported functions):

```typescript
/**
 * Validates that a password contains minimum required numbers.
 * @param amount - Minimum number of digits required
 * @param fieldName - Name of the field for error messages
 * @returns Validation function that returns error message or empty string
 */
export const mustIncludeNumbers =
  (amount: number, fieldName: string) =>
  (v: string): string => {
    // Implementation
  };
```

**Type Safety**:

```typescript
// ✅ Good: Explicit types
const form = document.querySelector<HTMLFormElement>("#form");
if (!form) throw new Error("Form not found");

// ❌ Bad: No type checking
const form = document.querySelector("#form");
```

### HTML Templates

**Accessibility**:

- Use semantic HTML (`<button>` not `<div onclick>`)
- Include ARIA labels for icon-only buttons
- Ensure 7:1 contrast ratios (WCAG AAA)
- Support keyboard navigation

**Atomic Design**:

- **Atoms**: Basic elements (buttons, icons, links) in `templates/atoms/`
- **Molecules**: Composite components (forms, headers) in `templates/molecules/`
- **Pages**: Full page templates in `templates/`

### CSS/Tailwind

**Guidelines**:

- Use Tailwind utility classes (avoid custom CSS)
- Follow density variants: `comfortable:` and `compact:`
- Support dark mode with `dark:` variants
- Use CSS variables for theme colors

---

## Testing Requirements

### Test Coverage Goals

- **Validators**: 100% (maintain)
- **RateLimit**: 70%+ (current: 72.3%)
- **ResetToken**: 70%+ (current: 71.7%)
- **RPC**: 50%+ (current: 45.6%)
- **Email**: 30%+ (current: 31.2%)

### Running Tests

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package
go test ./internal/validators -v

# Run integration tests (requires Docker)
go test ./internal/rpchandler -v
```

### Writing Tests

**Unit Tests**:

```go
func TestValidateMinLength(t *testing.T) {
    tests := []struct {
        name      string
        password  string
        minLength int
        wantErr   bool
    }{
        {"valid length", "12345678", 8, false},
        {"too short", "1234567", 8, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateMinLength(tt.password, tt.minLength)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Integration Tests** (use testcontainers):

```go
func TestPasswordReset_Integration(t *testing.T) {
    // Start LDAP container
    ctx := context.Background()
    ldapContainer, err := openldap.Run(ctx, "bitnami/openldap:latest")
    // ... test with real LDAP server
}
```

### Test Organization

- Test files: `*_test.go` in same package as source
- Table-driven tests for multiple cases
- Use `testify/assert` for assertions (if needed)
- Mock external dependencies (LDAP, SMTP)

---

## Pull Request Process

### Before Submitting

1. **Run tests**:

   ```bash
   go test ./... -cover
   ```

2. **Format code**:

   ```bash
   gofmt -w .
   pnpm prettier --write .
   ```

3. **Build assets**:

   ```bash
   pnpm build:assets
   ```

4. **Update documentation** if needed:
   - API changes → update `docs/api-reference.md`
   - New features → update `README.md` and relevant docs
   - Breaking changes → create migration guide

5. **Test manually**:
   - Test password change flow
   - Test password reset flow (if modified)
   - Test accessibility (keyboard navigation, screen reader)
   - Test dark mode and density modes

### PR Checklist

- [ ] Code follows project style guidelines
- [ ] All tests pass (`go test ./...`)
- [ ] Test coverage maintained or improved
- [ ] Documentation updated (if applicable)
- [ ] Commit messages follow Conventional Commits format
- [ ] No sensitive data (passwords, tokens) in code or commits
- [ ] Accessibility tested (if UI changes)
- [ ] Dark mode tested (if UI changes)

### PR Description Template

```markdown
## Description

Brief description of changes

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed

## Screenshots (if UI changes)

[Add screenshots here]

## Related Issues

Fixes #123
```

### Review Process

1. **Automated checks** run on PR (CI/CD)
2. **Code review** by maintainers
3. **Address feedback** and push updates
4. **Approval and merge** by maintainers

---

## Architecture Decision Records

For significant architectural or design decisions, create an ADR:

1. **Create ADR file**:

   ```bash
   docs/adr/NNNN-short-title.md
   ```

2. **Use the template**:

   ```markdown
   # ADR-NNNN: Title

   **Status**: Proposed | Accepted | Deprecated
   **Date**: YYYY-MM-DD
   **Authors**: @username

   ## Context

   What is the issue we're facing?

   ## Decision

   What decision did we make?

   ## Consequences

   What are the positive and negative outcomes?

   ## Alternatives Considered

   What other options were evaluated?
   ```

3. **Update index** in `docs/README.md`

**See existing ADRs**:

- [ADR-0001: Standardize Form Field Names](docs/adr/0001-standardize-form-field-names.md)
- [ADR-0002: Password Reset Functionality](docs/adr/0002-password-reset-functionality.md)

---

## Community

### Getting Help

- **Documentation**: Check [docs/](docs/) directory first
- **Issues**: Search [existing issues](https://github.com/netresearch/ldap-selfservice-password-changer/issues)
- **Discussions**: Use GitHub Discussions for questions

### Reporting Bugs

When reporting bugs, include:

- **Environment**: OS, Go version, browser (if frontend issue)
- **Steps to reproduce**: Clear, step-by-step instructions
- **Expected behavior**: What should happen
- **Actual behavior**: What actually happens
- **Logs/Screenshots**: Relevant error messages or screenshots

### Suggesting Features

For feature requests:

- **Use case**: Describe the problem you're solving
- **Proposed solution**: How would you implement it?
- **Alternatives**: What other approaches did you consider?
- **Scope**: Is this a small enhancement or major feature?

### Security Issues

**Do not open public issues for security vulnerabilities.**

Report security issues to the maintainers privately:

- See [SECURITY.md](SECURITY.md) for reporting process
- Use GitHub Security Advisories for responsible disclosure

---

## Resources

### Documentation

- [README](README.md) - Project overview
- [Development Guide](docs/development-guide.md) - Setup and workflows
- [API Reference](docs/api-reference.md) - JSON-RPC API
- [Architecture](docs/architecture.md) - System design
- [Security](docs/security.md) - Threat model and controls
- [Accessibility](docs/accessibility.md) - WCAG compliance

### External Links

- [Go Documentation](https://go.dev/doc/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [WCAG 2.2 Guidelines](https://www.w3.org/WAI/WCAG22/quickref/)

---

## License

By contributing to this project, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).

---

**Thank you for contributing!** Your efforts help make this project better for everyone.
