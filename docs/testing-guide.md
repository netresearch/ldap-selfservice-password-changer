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

- RPC handler logic (internal/rpchandler/change_password.go)
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

**Location**: Create `internal/rpchandler/change_password_test.go`

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

**Location**: Create `internal/rpchandler/handler_integration_test.go`

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

## Feature-Specific Testing

### Density Toggle Manual Testing

The adaptive density toggle feature requires comprehensive manual testing across different devices and system preferences.

#### Test Environment Setup

1. Navigate to application URL
2. Open browser DevTools (F12)
3. Have access to system display settings for preference testing

#### Core Functionality Tests

**Test 1: Button Cycling**

Steps:

1. Click density toggle button
2. Verify icon changes from SparklesIcon (auto) → Squares2x2Icon (comfortable)
3. Click again → SquaresPlusIcon (compact)
4. Click again → SparklesIcon (auto)

Expected: Icons cycle correctly, button stays at fixed position

**Test 2: Label Visibility**

Steps:

1. Set density to comfortable mode
2. Observe all form field labels visible
3. Set density to compact mode
4. Verify labels are visually hidden but present in DOM (sr-only)

Expected: Comfortable shows labels, compact hides visually but keeps accessible

**Test 3: Help Button Visibility**

Steps:

1. Set density to comfortable mode → help buttons visible
2. Set density to compact mode → help buttons hidden

Expected: Help buttons toggle with density mode

#### Auto Mode Detection Tests

**Test 4: Touch Device Detection**

Steps:

1. Set density to auto mode
2. Open DevTools Device Toolbar (Ctrl+Shift+M)
3. Select mobile device (e.g., iPhone 12)
4. Reload page or toggle off/on auto mode

Expected: Auto mode detects touch device and applies comfortable state

**Test 5: High Contrast Detection**

Steps:

1. Enable high contrast mode in OS
   - Windows: Settings → Accessibility → High Contrast
   - macOS: System Settings → Accessibility → Display → Increase Contrast
2. Reload page with density set to auto
3. Verify comfortable mode applied

Expected: Auto mode detects high contrast preference and applies comfortable state

**Test 6: Desktop with Mouse**

Steps:

1. Set density to auto mode
2. Use desktop browser with mouse (not touch)
3. Disable high contrast mode in OS
4. Reload page or toggle off/on auto mode

Expected: Auto mode detects mouse input + normal contrast and applies compact state

#### Reactive Behavior Tests

**Test 7: Reactive Preference Monitoring**

Steps:

1. Set density to auto mode in desktop browser
2. Open DevTools Device Toolbar (Ctrl+Shift+M)
3. Toggle between desktop and mobile device without reloading
4. Observe density state changes

Expected: Switching to mobile changes to comfortable, switching back to desktop changes to compact (no page reload required)

**Test 8: localStorage Persistence**

Steps:

1. Set density to comfortable mode
2. Reload page (F5)
3. Verify density remains comfortable
4. Repeat for compact and auto modes

Expected: Selected density mode persists across page reloads

#### Accessibility Tests

**Test 9: ARIA Labels**

Steps:

1. Cycle through all three density modes
2. Check aria-label attribute on density button after each click

Expected ARIA Labels:

- Auto: "Density: Auto density (follows system preferences). Click to switch to Comfortable mode"
- Comfortable: "Density: Comfortable mode (WCAG AAA, spacious layout). Click to switch to Compact mode"
- Compact: "Density: Compact mode (WCAG AA, simplified layout). Click to switch to Auto density"

**Test 10: Screen Reader Announcements**

Steps:

1. Use screen reader (NVDA/JAWS/VoiceOver)
2. Navigate to density button
3. Verify current mode and next action announced

Expected: Screen reader announces current mode and what clicking will do

#### System Preference Tests

**Test 11: Reduced Motion Preference**

Steps:

1. Enable reduced motion in OS
   - Windows: Settings → Accessibility → Visual effects → Animation effects (OFF)
   - macOS: System Settings → Accessibility → Display → Reduce motion
2. Reload page
3. Interact with toggles and form elements

Expected: All animations disabled (duration: 0.01ms), transitions instant

**Test 12: Low Contrast Preference**

Steps:

1. Enable low contrast mode (if available in OS)
2. Reload page
3. Observe text and background colors

Expected: Reduced contrast colors applied (lighter text, softer backgrounds)

#### WCAG 2.2 Compliance Verification

**Comfortable Mode (AAA)**

- ✅ 1.4.6 Contrast (Enhanced): 7:1 contrast ratio for text
- ✅ 2.5.5 Target Size (Enhanced): 44×44px minimum touch targets
- ✅ 1.4.12 Text Spacing: Generous padding and line-height
- ✅ 1.3.1 Info and Relationships: All labels visible

**Compact Mode (AA)**

- ✅ 1.4.3 Contrast (Minimum): 4.5:1 contrast ratio for text
- ✅ 2.5.8 Target Size (Minimum): 36×36px minimum touch targets
- ✅ 1.3.1 Info and Relationships: Labels accessible via sr-only
- ✅ 4.1.2 Name, Role, Value: Proper ARIA labels

**System Preference Support**

- ✅ 1.4.6 Contrast (Enhanced): prefers-contrast:more supported
- ✅ 2.3.3 Animation from Interactions: prefers-reduced-motion supported
- ✅ 1.4.10 Reflow: Responsive design maintained

#### Code References

- TypeScript Source: `internal/web/static/js/app.ts:79-169`
- CSS Variants: `internal/web/tailwind.css:26-71`
- HTML Template: `internal/web/templates/index.html:354-376`

## Related Documentation

- [API Reference](api-reference.md) - API contracts for integration tests
- [Accessibility Guide](accessibility.md) - WCAG compliance details
- [Development Guide](development-guide.md) - Test execution workflows
- [Architecture](architecture.md) - System design and testability considerations

---

_Last updated: 2025-10-09_
