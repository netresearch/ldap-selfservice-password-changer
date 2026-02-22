# Security Quick Fix Guide

**Immediate Actions for Critical Findings**

**Assessment Date:** 2025-10-09
**Target Completion:** Within 72 hours

---

## ⚠️ Critical Findings Summary

| ID          | Issue                 | Severity   | Effort  | Files                                    |
| ----------- | --------------------- | ---------- | ------- | ---------------------------------------- |
| CRITICAL-01 | Case sensitivity bug  | 7.5 (High) | 2 hours | change_password.go:64, validators.ts:170 |
| CRITICAL-02 | Missing rate limiting | 8.1 (High) | 4 hours | change_password.go, main.go              |
| HIGH-01     | LDAP injection risk   | 7.2 (High) | 6 hours | change_password.go, reset_password.go    |

**Total Estimated Effort:** 12 hours

---

## Fix #1: Case Sensitivity Bug (2 hours)

### Backend Fix

**File:** `internal/rpchandler/change_password.go`
**Line:** 64

**Current Code (VULNERABLE):**

```go
if !c.opts.PasswordCanIncludeUsername && strings.Contains(sAMAccountName, newPassword) {
    return nil, fmt.Errorf("the new password must not include the username")
}
```

**Fixed Code:**

```go
if !c.opts.PasswordCanIncludeUsername &&
   strings.Contains(strings.ToLower(newPassword), strings.ToLower(sAMAccountName)) {
    return nil, fmt.Errorf("the new password must not include the username")
}
```

### Frontend Fix

**File:** `internal/web/static/js/validators.ts`
**Line:** 170

**Current Code (VULNERABLE):**

```typescript
export const mustNotIncludeUsername = (fieldName: string) => (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#username input`);
  return v.includes(passwordInput.value) ? `${fieldName} must not include the username` : "";
};
```

**Fixed Code:**

```typescript
export const mustNotIncludeUsername = (fieldName: string) => (v: string) => {
  const passwordInput = form.querySelector<HTMLInputElement>(`#username input`);
  return v.toLowerCase().includes(passwordInput.value.toLowerCase())
    ? `${fieldName} must not include the username`
    : "";
};
```

### Testing

```bash
# Test case 1: Same case (should fail)
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{"method":"change-password","params":["admin","OldPass1!","admin123!"]}'
# Expected: {"success":false,"data":["the new password must not include the username"]}

# Test case 2: Different case (should fail after fix)
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{"method":"change-password","params":["admin","OldPass1!","Admin123!"]}'
# Expected: {"success":false,"data":["the new password must not include the username"]}

# Test case 3: No username (should succeed)
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{"method":"change-password","params":["admin","OldPass1!","NewPass456!"]}'
# Expected: {"success":true,"data":["password changed successfully"]}
```

---

## Fix #2: Missing Rate Limiting (4 hours)

### Implementation

**File:** `main.go`
**Location:** Before handler.Handle() call

**Add Rate Limiting Middleware:**

```go
// After line 50 (handler initialization)
app.Post("/api/rpc", func(c *fiber.Ctx) error {
    var body struct {
        Method string   `json:"method"`
        Params []string `json:"params"`
    }

    if err := c.BodyParser(&body); err != nil {
        return c.Status(http.StatusBadRequest).JSON(rpc.JSONRPCResponse{
            Success: false,
            Data:    []string{"invalid request format"},
        })
    }

    // Apply rate limiting for sensitive methods
    if body.Method == "change-password" || body.Method == "request-password-reset" {
        // Use IP address as identifier (or username from params[0] for user-based limiting)
        identifier := c.IP()

        if rateLimiter != nil && !rateLimiter.AllowRequest(identifier) {
            return c.Status(http.StatusTooManyRequests).JSON(rpc.JSONRPCResponse{
                Success: false,
                Data:    []string{"too many requests, please try again later"},
            })
        }
    }

    return handler.Handle(c)
})
```

### Configuration Options

**Option 1: IP-Based Rate Limiting (Recommended for initial fix)**

```go
identifier := c.IP()
```

**Option 2: User-Based Rate Limiting**

```go
// More targeted protection per user
identifier := body.Params[0] // username
```

**Option 3: Combined IP + User**

```go
identifier := fmt.Sprintf("%s:%s", c.IP(), body.Params[0])
```

### Testing

```bash
# Test rate limiting (should block after 5 attempts)
for i in {1..10}; do
  echo "Attempt $i:"
  response=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -X POST http://localhost:3000/api/rpc \
    -H "Content-Type: application/json" \
    -d '{"method":"change-password","params":["test","wrong","pass"]}')

  echo "$response"
  echo "---"
  sleep 1
done

# Expected output:
# Attempts 1-5: HTTP 500 (authentication failure)
# Attempts 6-10: HTTP 429 (rate limited)
```

### Verify Rate Limiter Configuration

```bash
# Check existing rate limiter settings
grep -A 10 "NewLimiter" main.go

# Expected configuration:
# - Window: 15 minutes
# - Max requests: 5
```

---

## Fix #3: LDAP Injection Protection (6 hours)

### Step 1: Verify simple-ldap-go Escaping (2 hours)

**Action:** Review `simple-ldap-go` source code for DN and filter escaping.

```bash
# Check library location
cd /srv/www/sme/ldap-selfservice-password-changer
grep -r "simple-ldap-go" go.mod

# Read library source
cat ~/go/pkg/mod/github.com/netresearch/simple-ldap-go@v1.6.0/ldap.go | grep -A 20 "escape\|sanitize"
```

**Questions to Answer:**

1. Does the library escape DN special characters: (, ), \*, \, NUL, =
2. Does the library escape filter special characters
3. Are there any public CVEs for this library version

### Step 2: Add Input Validation (2 hours)

**File:** Create `internal/validators/ldap.go`

```go
package validators

import (
    "fmt"
    "regexp"
)

// ValidateLDAPInput ensures username contains only safe characters
func ValidateLDAPInput(username string) error {
    // Whitelist approach: alphanumeric + underscore + hyphen + period
    // Adjust regex based on organization's AD username policy
    validPattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

    if !validPattern.MatchString(username) {
        return fmt.Errorf("username contains invalid characters")
    }

    // Additional length validation
    if len(username) < 1 || len(username) > 64 {
        return fmt.Errorf("username length must be between 1 and 64 characters")
    }

    return nil
}

// EscapeLDAPSpecialChars provides defense-in-depth escaping
func EscapeLDAPSpecialChars(input string) string {
    replacements := map[string]string{
        "\\": "\\5c",
        "*":  "\\2a",
        "(":  "\\28",
        ")":  "\\29",
        "\x00": "\\00",
        "/":  "\\2f",
    }

    result := input
    for char, escape := range replacements {
        result = strings.ReplaceAll(result, char, escape)
    }
    return result
}
```

**Apply in change_password.go:**

```go
// Add after line 26 (parameter extraction)
if err := validators.ValidateLDAPInput(sAMAccountName); err != nil {
    return nil, fmt.Errorf("invalid username format")
}
```

**Apply in reset_password.go:**

```go
// Add validation for token.Username before LDAP operations
if err := validators.ValidateLDAPInput(token.Username); err != nil {
    return nil, fmt.Errorf("invalid username in token")
}
```

### Step 3: Create Security Tests (2 hours)

**File:** Create `internal/validators/ldap_test.go`

```go
package validators

import "testing"

func TestLDAPInjectionProtection(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        shouldPass  bool
    }{
        {"Valid username", "john.doe", true},
        {"Valid with numbers", "user123", true},
        {"Valid with underscore", "john_doe", true},
        {"Valid with hyphen", "john-doe", true},
        {"LDAP injection - parentheses", "admin)(objectClass=*)", false},
        {"LDAP injection - asterisk", "user*", false},
        {"LDAP injection - backslash", "admin\\*", false},
        {"LDAP injection - equals", "user=admin", false},
        {"LDAP injection - comma", "user,ou=test", false},
        {"LDAP injection - plus", "user+admin", false},
        {"LDAP injection - quotes", "user\"admin", false},
        {"LDAP injection - angle brackets", "user<admin>", false},
        {"LDAP injection - semicolon", "user;admin", false},
        {"Empty username", "", false},
        {"Too long username", string(make([]byte, 65)), false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateLDAPInput(tt.input)

            if tt.shouldPass && err != nil {
                t.Errorf("Expected valid, got error: %v", err)
            }

            if !tt.shouldPass && err == nil {
                t.Errorf("Expected error, got valid for input: %s", tt.input)
            }
        })
    }
}

func TestLDAPEscaping(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"normal", "normal"},
        {"with*asterisk", "with\\2aasterisk"},
        {"with(paren)", "with\\28paren\\29"},
        {"with\\backslash", "with\\5cbackslash"},
        {"with/slash", "with\\2fslash"},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            result := EscapeLDAPSpecialChars(tt.input)
            if result != tt.expected {
                t.Errorf("Expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

**Run Tests:**

```bash
go test -v ./internal/validators -run TestLDAP

# Expected output:
# === RUN   TestLDAPInjectionProtection
# --- PASS: TestLDAPInjectionProtection
# === RUN   TestLDAPEscaping
# --- PASS: TestLDAPEscaping
# PASS
```

---

## Build and Deploy

### 1. Make Changes

```bash
# Apply all fixes above
```

### 2. Build

```bash
pnpm build
# Expected: No errors
```

### 3. Run Tests

```bash
go test ./...
# Expected: All tests pass
```

### 4. Test Locally

```bash
./ldap-selfservice-password-changer --port 9999
# Test with curl commands from each fix section
```

### 5. Commit

```bash
git add .
git commit -m "security: fix critical authentication vulnerabilities

- Fix case sensitivity bug in username inclusion check (CRITICAL-01)
- Add rate limiting to password change endpoint (CRITICAL-02)
- Add LDAP injection protection with input validation (HIGH-01)

CVSS Scores: 7.5, 8.1, 7.2
See docs/security-assessment-2025-10-09.md for full details"
```

---

## Verification Checklist

### CRITICAL-01: Case Sensitivity Bug

- [ ] Backend: `strings.ToLower()` applied in change_password.go:64
- [ ] Frontend: `.toLowerCase()` applied in validators.ts:170
- [ ] Test: Mixed-case username in password rejected
- [ ] Test: Valid password (no username) accepted

### CRITICAL-02: Missing Rate Limiting

- [ ] Rate limiter applied to password change method
- [ ] Test: 5 attempts succeed or fail based on credentials
- [ ] Test: 6th attempt returns HTTP 429
- [ ] Rate limiter resets after configured window (15 min default)

### HIGH-01: LDAP Injection

- [ ] Input validation function created (ValidateLDAPInput)
- [ ] Validation applied in change_password.go
- [ ] Validation applied in reset_password.go
- [ ] Test: LDAP special characters rejected
- [ ] Test: Valid usernames accepted
- [ ] Security tests pass

### Build & Deploy

- [ ] `pnpm build` succeeds
- [ ] `go test ./...` passes
- [ ] Manual testing with curl commands successful
- [ ] Changes committed with security commit message
- [ ] Deployed to staging environment
- [ ] Smoke tests pass in staging
- [ ] Ready for production deployment

---

## Next Steps (Post-Critical Fixes)

### Week 2-4: High Severity Fixes

1. **Add security headers** (4 hours)
   - Install Fiber helmet middleware
   - Configure CSP, X-Frame-Options, HSTS

2. **Implement secrets management** (12-20 hours)
   - Choose: Kubernetes Secrets / Vault / AWS Secrets Manager
   - Migrate LDAP and SMTP credentials
   - Document secret rotation procedures

### Week 5-12: Medium Severity Fixes

3. **Wrap LDAP errors** (4 hours)
4. **Add max password length** (2 hours)
5. **Automated vulnerability scanning** (4 hours)
6. **CSRF protection** (6 hours)
7. **Persistent token store** (8-12 hours)

**Full Roadmap:** See `docs/security-assessment-2025-10-09.md`

---

## Support

**Questions or Issues?**

- Review full assessment: `docs/security-assessment-2025-10-09.md`
- Check existing security patterns in `internal/ratelimit/` and `internal/resettoken/`
- Consult OWASP resources linked in main assessment document

---

**Quick Fix Guide Version:** 1.0
**Created:** 2025-10-09
**Target Completion:** 2025-10-12 (72 hours)
