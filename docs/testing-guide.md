# Testing Guide

## Current Test Coverage

### Existing Tests

**Location**: `internal/validators/validate_test.go`

**Coverage**: Password validator functions only

**Test Summary**:

- MinNumbersInString validation
- MinSymbolsInString validation
- MinUppercaseLettersInString validation
- MinLowercaseLettersInString validation

**Running Tests**:

```bash
go test ./...
```

**Example Output**:

```
ok      github.com/netresearch/ldap-selfservice-password-changer/internal/validators    0.002s
```

### Coverage Gaps

**Missing Unit Tests**:

- RPC handler logic (internal/rpc/change_password.go)
- Configuration parsing (internal/options/app.go)
- Template rendering (internal/web/templates/templates.go)
- Frontend validators (internal/web/static/js/validators.ts)

**Missing Integration Tests**:

- RPC endpoint testing
- LDAP connection testing (with mocks)
- Full password change workflow

**Missing E2E Tests**:

- Browser-based form submission
- Validation error display
- Success state transitions
- Password reveal functionality

## Testing Strategy Recommendations

### Unit Testing

#### Backend Unit Tests

**Goal**: Test pure functions and business logic in isolation

**Priority 1: RPC Handler Tests**

**Location**: Create `internal/rpc/change_password_test.go`

**Test Cases**:

```go
func TestChangePassword_Success(t *testing.T) {
    // Mock LDAP client
    // Valid parameters
    // Assert success response
}

func TestChangePassword_EmptyUsername(t *testing.T) {
    // Test empty username validation
    // Assert error response
}

func TestChangePassword_PasswordTooShort(t *testing.T) {
    // Test length validation
    // Assert error with correct message
}

func TestChangePassword_InsufficientNumbers(t *testing.T) {
    // Test number requirement
}

func TestChangePassword_InsufficientSymbols(t *testing.T) {
    // Test symbol requirement
}

func TestChangePassword_InsufficientUppercase(t *testing.T) {
    // Test uppercase requirement
}

func TestChangePassword_InsufficientLowercase(t *testing.T) {
    // Test lowercase requirement
}

func TestChangePassword_PasswordMatchesCurrent(t *testing.T) {
    // Test password uniqueness
}

func TestChangePassword_UsernameInPassword(t *testing.T) {
    // Test username exclusion (when enabled)
}

func TestChangePassword_LDAPError(t *testing.T) {
    // Mock LDAP error
    // Assert error propagation
}
```

**Implementation Pattern**:

```go
package rpc

import (
    "testing"
    "github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

// Mock LDAP client
type mockLDAP struct {
    changePasswordFunc func(user, old, new string) error
}

func (m *mockLDAP) ChangePasswordForSAMAccountName(user, old, new string) error {
    if m.changePasswordFunc != nil {
        return m.changePasswordFunc(user, old, new)
    }
    return nil
}

func TestChangePassword_Success(t *testing.T) {
    opts := &options.Opts{
        MinLength:    8,
        MinNumbers:   1,
        MinSymbols:   1,
        MinUppercase: 1,
        MinLowercase: 1,
    }

    handler := &Handler{
        ldap: &mockLDAP{},
        opts: opts,
    }

    result, err := handler.changePassword([]string{
        "testuser",
        "OldPass123!",
        "NewPass456!",
    })

    if err != nil {
        t.Fatalf("Expected success, got error: %v", err)
    }

    if len(result) != 1 || result[0] != "password changed successfully" {
        t.Errorf("Expected success message, got: %v", result)
    }
}
```

**Priority 2: Configuration Parsing Tests**

**Location**: Create `internal/options/app_test.go`

**Test Cases**:

- Default value application
- Environment variable parsing
- Flag override behavior
- Required field validation
- Integer parsing errors
- Boolean parsing errors

**Priority 3: Template Rendering Tests**

**Location**: Create `internal/web/templates/templates_test.go`

**Test Cases**:

- Successful template rendering
- Options injection
- HTML escaping
- Template syntax errors

#### Frontend Unit Tests

**Goal**: Test TypeScript validators independently

**Tool Recommendation**: Vitest (fast, TypeScript-native)

**Setup**:

```bash
pnpm add -D vitest @vitest/ui
```

**Configuration**: Create `vitest.config.ts`

```typescript
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "jsdom",
    include: ["internal/web/static/js/**/*.test.ts"]
  }
});
```

**Location**: Create `internal/web/static/js/validators.test.ts`

**Test Cases**:

```typescript
import { describe, it, expect } from "vitest";
import {
  mustNotBeEmpty,
  mustBeLongerThan,
  mustIncludeNumbers,
  mustIncludeSymbols,
  mustIncludeUppercase,
  mustIncludeLowercase
} from "./validators";

describe("mustNotBeEmpty", () => {
  it("returns error for empty string", () => {
    expect(mustNotBeEmpty("")).toBeTruthy();
  });

  it("returns no error for non-empty string", () => {
    expect(mustNotBeEmpty("test")).toBe("");
  });
});

describe("mustBeLongerThan", () => {
  it("returns error when too short", () => {
    const validator = mustBeLongerThan(8);
    expect(validator("short")).toBeTruthy();
  });

  it("returns no error when long enough", () => {
    const validator = mustBeLongerThan(8);
    expect(validator("longenough")).toBe("");
  });
});

describe("mustIncludeNumbers", () => {
  it("returns error when insufficient numbers", () => {
    const validator = mustIncludeNumbers(2);
    expect(validator("abc1")).toBeTruthy();
  });

  it("returns no error with enough numbers", () => {
    const validator = mustIncludeNumbers(2);
    expect(validator("abc123")).toBe("");
  });
});

// Similar tests for symbols, uppercase, lowercase
```

### Integration Testing

#### RPC Endpoint Integration Tests

**Goal**: Test HTTP endpoints with real Fiber app, mock LDAP

**Tool**: Go standard library `net/http/httptest`

**Location**: Create `internal/rpc/handler_integration_test.go`

**Example**:

```go
package rpc

import (
    "bytes"
    "encoding/json"
    "net/http/httptest"
    "testing"

    "github.com/gofiber/fiber/v2"
    "github.com/netresearch/ldap-selfservice-password-changer/internal/options"
)

func TestRPCEndpoint_ChangePassword_Success(t *testing.T) {
    // Setup
    opts := &options.Opts{
        MinLength: 8,
        // ... other opts
    }

    handler := &Handler{
        ldap: &mockLDAP{},
        opts: opts,
    }

    app := fiber.New()
    app.Post("/api/rpc", handler.Handle)

    // Request
    payload := JSONRPC{
        Method: "change-password",
        Params: []string{"testuser", "OldPass123!", "NewPass456!"},
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest("POST", "/api/rpc", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    // Execute
    resp, _ := app.Test(req)

    // Assert
    if resp.StatusCode != 200 {
        t.Fatalf("Expected 200, got %d", resp.StatusCode)
    }

    var response JSONRPCResponse
    json.NewDecoder(resp.Body).Decode(&response)

    if !response.Success {
        t.Errorf("Expected success=true, got false")
    }
}

func TestRPCEndpoint_MethodNotFound(t *testing.T) {
    // Test invalid method
    // Assert 400 status code
}

func TestRPCEndpoint_InvalidJSON(t *testing.T) {
    // Test malformed JSON
    // Assert error response
}
```

#### LDAP Mock Testing

**Tool**: Interface-based mocking

**Pattern**:

```go
// Define interface
type LDAPClient interface {
    ChangePasswordForSAMAccountName(username, oldPass, newPass string) error
}

// Production implementation uses real LDAP
// Test implementation uses mock

type mockLDAP struct {
    changePasswordErr error
}

func (m *mockLDAP) ChangePasswordForSAMAccountName(username, oldPass, newPass string) error {
    return m.changePasswordErr
}
```

**Test Scenarios**:

- LDAP connection timeout
- Invalid credentials (current password wrong)
- User not found
- Password policy violation from LDAP server
- Network errors

### End-to-End Testing

#### Browser-Based E2E Tests

**Goal**: Test complete user workflows in real browser

**Tool Recommendation**: Playwright (TypeScript-native, modern)

**Setup**:

```bash
pnpm add -D @playwright/test
pnpm exec playwright install
```

**Configuration**: Create `playwright.config.ts`

```typescript
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  webServer: {
    command: "pnpm dev",
    port: 3000,
    reuseExistingServer: !process.env.CI
  },
  use: {
    baseURL: "http://localhost:3000"
  }
});
```

**Location**: Create `tests/e2e/password-change.spec.ts`

**Test Cases**:

```typescript
import { test, expect } from "@playwright/test";

test("successful password change flow", async ({ page }) => {
  await page.goto("/");

  // Fill form
  await page.fill("#username input", "testuser");
  await page.fill("#current input", "OldPass123!");
  await page.fill("#new input", "NewPass456!");
  await page.fill("#new2 input", "NewPass456!");

  // Submit
  await page.click('button[type="submit"]');

  // Assert success
  await expect(page.locator('[data-purpose="successContainer"]')).toBeVisible();
  await expect(page.locator("#form")).not.toBeVisible();
});

test("validation errors display correctly", async ({ page }) => {
  await page.goto("/");

  // Fill with invalid password (too short)
  await page.fill("#username input", "testuser");
  await page.fill("#current input", "OldPass123!");
  await page.fill("#new input", "short");
  await page.fill("#new2 input", "short");

  // Trigger validation (blur field)
  await page.click("#username input");

  // Assert error displayed
  await expect(page.locator('#new [data-purpose="errors"]')).toContainText("at least 8");

  // Assert submit button disabled
  await expect(page.locator('button[type="submit"]')).toBeDisabled();
});

test("password reveal toggle works", async ({ page }) => {
  await page.goto("/");

  const passwordInput = page.locator("#current input");
  const revealButton = page.locator('#current button[data-purpose="reveal"]');

  // Initially password type
  await expect(passwordInput).toHaveAttribute("type", "password");

  // Click reveal
  await revealButton.click();

  // Now text type
  await expect(passwordInput).toHaveAttribute("type", "text");

  // Click again to hide
  await revealButton.click();

  // Back to password
  await expect(passwordInput).toHaveAttribute("type", "password");
});

test("loading state during submission", async ({ page }) => {
  await page.goto("/");

  // Fill form
  await page.fill("#username input", "testuser");
  await page.fill("#current input", "OldPass123!");
  await page.fill("#new input", "NewPass456!");
  await page.fill("#new2 input", "NewPass456!");

  // Submit
  const submitButton = page.locator('button[type="submit"]');
  await submitButton.click();

  // Assert loading state
  await expect(submitButton).toHaveAttribute("data-loading", "true");
  await expect(submitButton).toBeDisabled();

  // Assert loading icon visible
  await expect(submitButton.locator("svg")).toBeVisible();
});

test("error from backend displays correctly", async ({ page }) => {
  await page.goto("/");

  // Mock API to return error
  await page.route("/api/rpc", (route) => {
    route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({
        success: false,
        data: ["the old password can't be same as the new one"]
      })
    });
  });

  // Fill and submit
  await page.fill("#username input", "testuser");
  await page.fill("#current input", "SamePass123!");
  await page.fill("#new input", "SamePass123!");
  await page.fill("#new2 input", "SamePass123!");
  await page.click('button[type="submit"]');

  // Assert error message
  await expect(page.locator('[data-purpose="submit"] [data-purpose="errors"]')).toContainText("same as the new one");
});
```

**Running E2E Tests**:

```bash
# Run all tests
pnpm exec playwright test

# Run with UI
pnpm exec playwright test --ui

# Debug mode
pnpm exec playwright test --debug
```

## Test Organization

### Recommended Directory Structure

```
ldap-selfservice-password-changer/
├── internal/
│   ├── validators/
│   │   ├── validate.go
│   │   └── validate_test.go           # ✅ Existing unit tests
│   ├── rpc/
│   │   ├── handler.go
│   │   ├── change_password.go
│   │   ├── change_password_test.go    # ❌ TODO: Unit tests
│   │   └── handler_integration_test.go # ❌ TODO: Integration tests
│   ├── options/
│   │   ├── app.go
│   │   └── app_test.go                # ❌ TODO: Unit tests
│   └── web/
│       ├── templates/
│       │   ├── templates.go
│       │   └── templates_test.go      # ❌ TODO: Unit tests
│       └── static/js/
│           ├── validators.ts
│           ├── validators.test.ts     # ❌ TODO: Frontend unit tests
│           ├── app.ts
│           └── app.test.ts            # ❌ TODO: Frontend unit tests
│
└── tests/
    └── e2e/
        ├── password-change.spec.ts    # ❌ TODO: E2E tests
        └── validation.spec.ts         # ❌ TODO: E2E tests
```

## Test Automation

### Continuous Integration

**Dockerfile Already Includes Tests**: Line 25

```dockerfile
RUN go test ./...
```

**Docker build fails if tests fail** (fail-fast pattern).

**GitHub Actions Recommendation**:

Create `.github/workflows/test.yml`:

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v6
        with:
          go-version: "1.25"

      - uses: pnpm/action-setup@v4
        with:
          version: 10.17.1

      - name: Install dependencies
        run: pnpm install

      - name: Build frontend
        run: pnpm build:assets

      - name: Run Go tests
        run: go test -v -cover ./...

      - name: Run TypeScript tests
        run: pnpm test

      - name: Run E2E tests
        run: pnpm exec playwright test
```

### Pre-Commit Hooks

**Tool**: Husky (already in dependencies)

**Setup**: Create `.husky/pre-commit`

```bash
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

# Run tests before commit
go test ./...
pnpm test

# Run linters
pnpm prettier --check .
gofmt -l .
```

**Installation**:

```bash
pnpm add -D husky
pnpm exec husky install
```

## Coverage Analysis

### Go Coverage

**Generate coverage report**:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

**View in browser**:

```bash
open coverage.html
```

**Coverage goals**:

- Validators: 100% (pure functions)
- RPC handlers: 80%+ (business logic)
- Configuration: 70%+ (parsing logic)

### TypeScript Coverage

**With Vitest**:

```bash
pnpm exec vitest --coverage
```

**Configuration**: Update `vitest.config.ts`

```typescript
export default defineConfig({
  test: {
    coverage: {
      provider: "v8",
      reporter: ["text", "html"],
      include: ["internal/web/static/js/**/*.ts"],
      exclude: ["**/*.test.ts"]
    }
  }
});
```

## Testing Best Practices

### Unit Tests

✅ **Do**:

- Test one function per test file
- Use descriptive test names (TestFunctionName_Scenario_ExpectedBehavior)
- Test happy path and error cases
- Use table-driven tests for multiple scenarios
- Mock external dependencies (LDAP, HTTP)

❌ **Don't**:

- Test implementation details
- Couple tests to each other
- Use real LDAP connections in unit tests
- Skip error case testing

### Integration Tests

✅ **Do**:

- Test complete request/response cycles
- Use real HTTP server (httptest)
- Mock external services (LDAP)
- Test error handling and edge cases
- Verify HTTP status codes

❌ **Don't**:

- Mix unit and integration tests
- Rely on external services
- Skip negative test cases

### E2E Tests

✅ **Do**:

- Test critical user workflows
- Use real browser automation
- Test UI interactions (clicks, form fills)
- Verify visual elements (success/error states)
- Mock backend errors for error paths

❌ **Don't**:

- Test every possible validation (that's unit test territory)
- Rely on production LDAP for tests
- Create brittle selectors (use data-purpose attributes)

## Related Documentation

- [API Reference](api-reference.md) - API contracts for integration tests
- [Architecture Patterns](architecture-patterns.md) - Testability considerations
- [Development Guide](development-guide.md) - Test execution workflows
- [Component Reference](component-reference.md) - Component interfaces for mocking

---

_Generated by /sc:index on 2025-10-04_
