# Onboarding Checklist

Progressive learning path for new developers joining the LDAP Selfservice Password Changer project.

## Prerequisites

Before starting, ensure you have:

- [ ] Go 1.24+ installed (`go version`)
- [ ] Node.js v16+ installed (`node --version`)
- [ ] pnpm installed via Corepack (`corepack enable`)
- [ ] Git configured with your credentials
- [ ] Access to LDAP/AD test server (or credentials for development)
- [ ] Code editor with Go and TypeScript support

## Day 1: Project Understanding

### Morning: High-Level Architecture

- [ ] Read [Project Context](project-context-2025-10-04.md) - Overview and tech stack
- [ ] Review [Architecture Patterns: Core Architecture](architecture-patterns.md#core-architecture)
- [ ] Understand the SPA + Go backend model
- [ ] Review [Architecture Diagram](../docs/architecture.md)

**Key Concepts to Grasp**:

- JSON-RPC over REST rationale
- Embedded static assets (embed.FS)
- Dual validation pattern (frontend + backend)
- No password storage (LDAP passthrough)

### Afternoon: Setup Development Environment

- [ ] Clone repository: `git clone https://github.com/netresearch/ldap-selfservice-password-changer.git`
- [ ] Install dependencies: `pnpm install`
- [ ] Create `.env.local` with LDAP credentials (see [Development Guide](development-guide.md#4-configure-ldap-connection))
- [ ] Build project: `pnpm build`
- [ ] Verify build output: `./ldap-selfservice-password-changer --help`

**Success Criteria**:

- ✅ Project builds without errors
- ✅ Help text displays all configuration flags
- ✅ No missing dependencies

### End of Day: First Run

- [ ] Start development server: `pnpm dev`
- [ ] Open http://localhost:3000 in browser
- [ ] Inspect form fields (username, current, new, new2)
- [ ] Try password reveal toggle
- [ ] Observe hot reload (edit tailwind.css, save, see changes)

**Reflection Questions**:

1. What are the 4 input fields and their purposes?
2. How does hot reload work (which processes are running)?
3. What port does the server listen on?

---

## Day 2: Code Structure

### Morning: Backend Packages

- [ ] Read [Component Reference: main](component-reference.md#go-package-main)
- [ ] Explore `main.go` - Entry point and routing
- [ ] Read [Component Reference: internal/options](component-reference.md#go-package-internaloptions)
- [ ] Understand configuration priority (flags > env vars > defaults)
- [ ] Read [Component Reference: internal/rpchandler](component-reference.md#go-package-internalrpc)
- [ ] Trace JSON-RPC request flow

**Exercises**:

1. Locate where the server port is defined (`main.go:50`)
2. Find where body size limit is set (`main.go:31`)
3. Identify the single RPC method (`handler.go:49`)

### Afternoon: Validation Logic

- [ ] Read [Component Reference: internal/validators](component-reference.md#go-package-internalvalidators)
- [ ] Understand ASCII range checks for symbols
- [ ] Read [API Reference: Validation Rules](api-reference.md#validation-rules)
- [ ] Compare backend and frontend validators
- [ ] Review [Architecture Patterns: Dual Validation](architecture-patterns.md#dual-validation-frontend--backend)

**Exercises**:

1. What ASCII ranges define symbols? (lines 14-23 in validate.go)
2. Why validate both frontend and backend?
3. How many validators exist? (4: numbers, symbols, uppercase, lowercase)

### End of Day: Make a Small Change

- [ ] Add console.log to `app.ts` to print validation errors
- [ ] Save file and observe TypeScript recompile
- [ ] Test in browser, check browser console
- [ ] Remove the console.log

**Success Criteria**:

- ✅ Understand hot reload triggers TypeScript recompile
- ✅ Can locate and modify TypeScript files
- ✅ Familiar with developer console debugging

---

## Day 3: Frontend Architecture

### Morning: TypeScript Modules

- [ ] Read [Component Reference: validators.ts](component-reference.md#typescript-module-validatorsts)
- [ ] Understand validator function factories (`mustBeLongerThan(n)` returns validator)
- [ ] Read [Component Reference: app.ts](component-reference.md#typescript-module-appts)
- [ ] Trace form submission flow (`app.ts:134-188`)

**Exercises**:

1. How is `specialCharacters` array generated? (ASCII loops)
2. What's the difference between `mustNotBeEmpty` and `mustBeLongerThan`? (static vs parameterized)
3. Where is form validation triggered? (`form.onchange`)

### Afternoon: Template System

- [ ] Read [Component Reference: internal/web/templates](component-reference.md#go-package-internalwebtemplates)
- [ ] Explore `index.html` template structure
- [ ] Understand how server options inject into JavaScript (`index.html:185-192`)
- [ ] Review [Architecture Patterns: Template Rendering](architecture-patterns.md#template-rendering-with-embedded-html)

**Exercises**:

1. Where are password reveal icons defined? (EyeSlashIcon, EyeIcon templates)
2. How does the server pass `MinLength` to frontend? (template data injection)
3. What happens on successful password change? (form hidden, success container shown)

### End of Day: API Call Flow

- [ ] Set breakpoint in `app.ts:148` (fetch call)
- [ ] Submit form with valid data
- [ ] Inspect network request in DevTools
- [ ] Review JSON-RPC request structure
- [ ] Check response format

**Success Criteria**:

- ✅ Can trace request from form submit to LDAP
- ✅ Understand JSON-RPC request/response structure
- ✅ Familiar with browser DevTools network tab

---

## Day 4: Testing & Quality

### Morning: Existing Tests

- [ ] Read [Testing Guide: Current Test Coverage](testing-guide.md#current-test-coverage)
- [ ] Run tests: `go test ./...`
- [ ] Explore `internal/validators/validate_test.go`
- [ ] Understand table-driven test pattern

**Exercises**:

1. How many test functions exist? (4: MinNumbers, MinSymbols, MinUppercase, MinLowercase)
2. Run tests with coverage: `go test -cover ./...`
3. What's the current coverage percentage?

### Afternoon: Write Your First Test

- [ ] Read [Testing Guide: Unit Testing](testing-guide.md#unit-testing)
- [ ] Choose a simple function to test (e.g., `pluralize` in `change_password.go:10`)
- [ ] Write test cases for edge cases
- [ ] Run your test and verify it passes

**Example Test**:

```go
func TestPluralize(t *testing.T) {
    if pluralize("number", 1) != "number" {
        t.Error("Expected singular form")
    }
    if pluralize("number", 2) != "numbers" {
        t.Error("Expected plural form")
    }
}
```

### End of Day: Code Review Practice

- [ ] Review recent commits: `git log --oneline -5`
- [ ] Pick a commit and review changes: `git show <commit-hash>`
- [ ] Read [Architecture Patterns: Design Decisions Summary](architecture-patterns.md#design-decisions-summary)
- [ ] Understand trade-offs made

**Reflection Questions**:

1. Why use JSON-RPC instead of REST?
2. Why embed static assets instead of serving from disk?
3. What are benefits of dual validation?

---

## Day 5: Make Your First Contribution

### Morning: Identify Improvement Area

Options (pick one):

- [ ] **Testing**: Write integration test for RPC endpoint ([Testing Guide](testing-guide.md#integration-testing))
- [ ] **Documentation**: Add inline JSDoc comments to `validators.ts`
- [ ] **Feature**: Add new validator (e.g., `mustNotContainCommonPasswords`)
- [ ] **Refactoring**: Extract magic numbers to constants

### Afternoon: Implementation

- [ ] Create feature branch: `git checkout -b feature/your-improvement`
- [ ] Make changes following project conventions
- [ ] Write tests for your changes
- [ ] Run all tests: `go test ./...`
- [ ] Format code: `pnpm prettier --write .` and `gofmt -w .`
- [ ] Commit with conventional message: `feat: add password strength indicator`

### End of Day: Code Review

- [ ] Review your changes: `git diff main`
- [ ] Ensure no unintended modifications
- [ ] Check for console.log or debug statements
- [ ] Verify tests pass
- [ ] (Optional) Push branch and create PR for review

**Success Criteria**:

- ✅ Made meaningful contribution to codebase
- ✅ Followed project conventions and patterns
- ✅ Wrote tests for changes
- ✅ Comfortable with git workflow

---

## Week 2 and Beyond

### Advanced Topics

#### Security Deep Dive

- [ ] Read [API Reference: Security Considerations](api-reference.md#security-considerations)
- [ ] Understand LDAPS requirement for ActiveDirectory
- [ ] Review password security patterns (no storage, readonly user)
- [ ] Learn about attack surface minimization

#### Performance Optimization

- [ ] Read [Architecture Patterns: Performance Optimizations](architecture-patterns.md#performance-optimizations)
- [ ] Understand compression strategy (Brotli LevelBestSpeed)
- [ ] Review caching patterns (24-hour static asset cache)
- [ ] Learn about request body limits (4KB)

#### Docker & Deployment

- [ ] Read [Development Guide: Docker Development](development-guide.md#docker-development)
- [ ] Understand multi-stage build process
- [ ] Build Docker image: `docker build -t ldap-password-changer .`
- [ ] Run container with env vars
- [ ] Review Alpine Linux optimizations

#### E2E Testing

- [ ] Read [Testing Guide: E2E Testing](testing-guide.md#end-to-end-testing)
- [ ] Set up Playwright: `pnpm add -D @playwright/test`
- [ ] Write first E2E test for password change flow
- [ ] Run tests in headless mode
- [ ] Debug with Playwright inspector

---

## Knowledge Checkpoints

### Checkpoint 1: Architecture (End of Day 2)

Can you answer these without looking?

1. What are the 6 Go packages in the project?
2. What does embed.FS do and why is it used?
3. Name the 4 password validation categories.
4. How does configuration priority work?

### Checkpoint 2: Frontend (End of Day 3)

Can you answer these without looking?

1. How many TypeScript files are there? (2: app.ts, validators.ts)
2. What triggers form validation? (onchange event)
3. How does password reveal work? (toggle input type)
4. What happens when RPC call fails? (error displayed, form re-enabled, submit disabled)

### Checkpoint 3: Full Stack (End of Day 5)

Can you trace the complete flow?

1. User submits form → ?
2. TypeScript validates → ?
3. Fetch POST to `/api/rpc` → ?
4. Fiber parses JSON → ?
5. Handler dispatches to `changePassword` → ?
6. Backend validates → ?
7. LDAP password change → ?
8. Response returned → ?
9. Success state shown → ?

---

## Common Beginner Questions

### Q: Why are there two validation implementations?

**A**: Frontend for UX (immediate feedback), backend for security (can't bypass client-side checks). See [Architecture Patterns: Dual Validation](architecture-patterns.md#dual-validation-frontend--backend).

### Q: Can I modify the HTML template directly?

**A**: Yes! Edit `internal/web/templates/index.html`, but note it uses Go templates syntax (`{{ .opts.MinLength }}`). Server restart required to see changes.

### Q: How do I add a new configuration option?

**A**: Follow [Development Guide: Adding Configuration Option](development-guide.md#adding-configuration-option). Update `options.Opts` struct, add flag, inject into template.

### Q: Where do I put new Go packages?

**A**: Under `internal/` directory. This makes them private to the project (Go convention).

### Q: Can I use a different CSS framework?

**A**: Technically yes, but project is built around Tailwind CSS v4. Major refactor required. Not recommended.

### Q: Why TypeScript instead of plain JavaScript?

**A**: Type safety prevents runtime errors. Strict mode enabled (`tsconfig.json`) catches bugs at compile time.

### Q: How do I debug the Go backend?

**A**: Use `log.Printf()` for simple debugging or Delve debugger for breakpoints: `dlv debug`.

### Q: What's the difference between `.env` and `.env.local`?

**A**: `.env` is committed (defaults), `.env.local` is gitignored (your local overrides). Never commit credentials!

---

## Resources by Role

### Frontend Developer

**Priority Reading**:

1. [Component Reference: validators.ts](component-reference.md#typescript-module-validatorsts)
2. [Component Reference: app.ts](component-reference.md#typescript-module-appts)
3. [Architecture Patterns: Frontend Patterns](architecture-patterns.md#frontend-patterns)

**Key Files**: `app.ts`, `validators.ts`, `index.html`, `tailwind.css`

### Backend Developer

**Priority Reading**:

1. [Component Reference: internal/rpchandler](component-reference.md#go-package-internalrpc)
2. [Component Reference: internal/validators](component-reference.md#go-package-internalvalidators)
3. [API Reference](api-reference.md)

**Key Files**: `main.go`, `handler.go`, `change_password.go`, `validate.go`

### DevOps Engineer

**Priority Reading**:

1. [Development Guide: Docker Development](development-guide.md#docker-development)
2. [Architecture Patterns: Build and Deployment](architecture-patterns.md#build-and-deployment-patterns)
3. [API Reference: Security Considerations](api-reference.md#security-considerations)

**Key Files**: `Dockerfile`, `.env`, `pnpm scripts`

### QA Engineer

**Priority Reading**:

1. [Testing Guide](testing-guide.md)
2. [API Reference: Validation Rules](api-reference.md#validation-rules)
3. [API Reference: Error Handling](api-reference.md#error-handling)

**Key Files**: `validate_test.go`, test specs (to be created)

---

## Onboarding Success Indicators

After completing this checklist, you should be able to:

- [ ] **Run**: Start dev server and access application
- [ ] **Navigate**: Find any package/file within 30 seconds
- [ ] **Understand**: Explain architecture to a peer
- [ ] **Debug**: Trace request flow from browser to LDAP
- [ ] **Test**: Write and run unit tests
- [ ] **Contribute**: Add small feature with tests
- [ ] **Deploy**: Build Docker image and run container
- [ ] **Document**: Update documentation for your changes

**Estimated Time**: 40 hours (1 week at 8 hours/day)

---

## Next Steps

After onboarding:

- Join team meetings and code reviews
- Pick up issues from backlog
- Propose improvements to architecture
- Mentor future team members using this checklist

**Continuous Learning**:

- Read Go blog: https://go.dev/blog/
- Learn TypeScript best practices
- Study LDAP/AD concepts
- Explore Fiber framework features

---

_Generated by /sc:document for onboarding support - 2025-10-04_
