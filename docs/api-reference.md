# API Reference

## JSON-RPC API

### Endpoint

```
POST /api/rpc
Content-Type: application/json
```

### Request Format

```typescript
{
  "method": string,  // RPC method name
  "params": string[] // Method parameters
}
```

### Response Format

```typescript
{
  "success": boolean,  // Operation success status
  "data": string[]     // Response data or error messages
}
```

## Available Methods

### change-password

Changes a user's password in the LDAP/ActiveDirectory server (requires current password).

#### Request

```json
{
  "method": "change-password",
  "params": [
    "username", // sAMAccountName
    "currentPassword", // Current password for authentication
    "newPassword" // New password to set
  ]
}
```

#### Successful Response

```json
{
  "success": true,
  "data": ["password changed successfully"]
}
```

**HTTP Status**: 200 OK

#### Error Responses

##### Validation Errors

```json
{
  "success": false,
  "data": ["error message"]
}
```

**HTTP Status**: 500 Internal Server Error

**Common Validation Errors**:

- `"the username can't be empty"`
- `"the old password can't be empty"`
- `"the new password can't be empty"`
- `"the old password can't be same as the new one"`
- `"the new password must be at least {N} characters long"`
- `"the new password must contain at least {N} number(s)"`
- `"the new password must contain at least {N} symbol(s)"`
- `"the new password must contain at least {N} uppercase letter(s)"`
- `"the new password must contain at least {N} lowercase letter(s)"`
- `"the new password must not include the username"`

##### LDAP Errors

```json
{
  "success": false,
  "data": ["LDAP error message from simple-ldap-go"]
}
```

**HTTP Status**: 500 Internal Server Error

**Common LDAP Errors**:

- Authentication failures (incorrect current password)
- User not found in LDAP directory
- Password policy violations from AD/LDAP server
- Connection errors to LDAP server

##### Invalid Method

```json
{
  "success": false,
  "data": ["method not found"]
}
```

**HTTP Status**: 400 Bad Request

---

### request-password-reset

Initiates password reset process by sending a secure token via email.

**Security Note**: Always returns generic success message to prevent user enumeration, even if email doesn't exist.

#### Request

```json
{
  "method": "request-password-reset",
  "params": [
    "user@example.com" // User's email address
  ]
}
```

#### Successful Response

```json
{
  "success": true,
  "data": ["If an account exists, a reset email has been sent"]
}
```

**HTTP Status**: 200 OK

**Note**: Same response returned regardless of whether email exists in LDAP (security feature to prevent user enumeration).

#### Error Responses

##### Invalid Argument Count

```json
{
  "success": false,
  "data": ["invalid argument count"]
}
```

**HTTP Status**: 500 Internal Server Error

**Rate Limiting**: Silently enforced (3 requests/hour per email by default). Rate-limited requests still return success message.

**Internal Processing**:

1. Validate email format
2. Check rate limit (returns success if exceeded)
3. Query LDAP for user by email (FindUserByMail)
4. Generate cryptographic token (32 bytes, crypto/rand)
5. Store token with 15-minute expiration
6. Send email with reset link
7. Return generic success message

---

### reset-password

Completes password reset using a valid token from email.

#### Request

```json
{
  "method": "reset-password",
  "params": [
    "TOKEN_STRING_FROM_EMAIL", // Token from reset email
    "NewPassword123!" // New password to set
  ]
}
```

#### Successful Response

```json
{
  "success": true,
  "data": ["Password reset successfully. You can now login."]
}
```

**HTTP Status**: 200 OK

#### Error Responses

##### Invalid or Expired Token

```json
{
  "success": false,
  "data": ["Invalid or expired token"]
}
```

**HTTP Status**: 500 Internal Server Error

**Causes**:

- Token doesn't exist in store
- Token expired (>15 minutes old)
- Token already used

##### Password Policy Violations

Same validation errors as `change-password` method:

```json
{
  "success": false,
  "data": ["the new password must be at least 8 characters long"]
}
```

**HTTP Status**: 500 Internal Server Error

##### LDAP Update Failure

```json
{
  "success": false,
  "data": ["Failed to reset password. Please contact your administrator if this problem persists."]
}
```

**HTTP Status**: 500 Internal Server Error

**Causes**:

- LDAP connection failure
- Insufficient permissions (service account lacks reset password permission)
- User account issues

**Internal Processing**:

1. Validate token exists and not expired
2. Validate token not already used
3. Validate new password against policy rules
4. Update password in LDAP (ChangePasswordForSAMAccountName with empty old password)
5. Mark token as used
6. Return success message

**LDAP Permission Requirements**:

- **Active Directory**: Service account (LDAP_RESET_USER or LDAP_READONLY_USER) needs "Reset password" permission
- **OpenLDAP**: Service account needs write access to userPassword attribute
- **Connection**: Must use LDAPS (ldaps://)
- **Security Best Practice**: Use dedicated LDAP_RESET_USER with minimal permissions (only password reset)
- **Backward Compatibility**: Falls back to LDAP_READONLY_USER if LDAP_RESET_USER not configured

---

## Implementation Details

### Backend Validation Flow

**Location**: internal/rpc/change_password.go:18-72

```go
func (c *Handler) changePassword(params []string) ([]string, error) {
  // 1. Validate parameter count
  if len(params) != 3 {
    return nil, ErrInvalidArgumentCount
  }

  // 2. Extract parameters
  sAMAccountName := params[0]
  currentPassword := params[1]
  newPassword := params[2]

  // 3. Empty field validation
  // 4. Password match validation
  // 5. Length validation (c.opts.MinLength)
  // 6. Number validation (validators.MinNumbersInString)
  // 7. Symbol validation (validators.MinSymbolsInString)
  // 8. Uppercase validation (validators.MinUppercaseLettersInString)
  // 9. Lowercase validation (validators.MinLowercaseLettersInString)
  // 10. Username inclusion check (optional, based on c.opts.PasswordCanIncludeUsername)
  // 11. LDAP password change operation

  return []string{"password changed successfully"}, nil
}
```

### Frontend Implementation

**Location**: internal/web/static/js/app.ts:134-188

```typescript
form.onsubmit = async (e) => {
  e.preventDefault();

  // 1. Collect form values
  const [username, oldPassword, newPassword] = fields.map((f) => f.getValue());

  // 2. Validate all fields
  const hasErrors = fields.map(({ validate }) => validate()).some((e) => e === true);
  if (hasErrors) return;

  // 3. Disable form during submission
  toggleFields(false);

  // 4. Make RPC request
  const res = await fetch("/api/rpc", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      method: "change-password",
      params: [username, oldPassword, newPassword]
    })
  });

  // 5. Handle response
  if (!res.ok) {
    // Display error
  } else {
    // Show success state
    form.style.display = "none";
    successContainer.style.display = "block";
  }
};
```

## Validation Rules

### Password Requirements

All validation rules are configurable via environment variables or command-line flags:

| Requirement        | Config Variable                                                     | Default | Backend Validator                       | Frontend Validator                     |
| ------------------ | ------------------------------------------------------------------- | ------- | --------------------------------------- | -------------------------------------- |
| Minimum Length     | `MIN_LENGTH` / `--min-length`                                       | 8       | `len(password) >= minLength`            | `mustBeLongerThan(n)`                  |
| Minimum Numbers    | `MIN_NUMBERS` / `--min-numbers`                                     | 1       | `MinNumbersInString`                    | `mustIncludeNumbers(n)`                |
| Minimum Symbols    | `MIN_SYMBOLS` / `--min-symbols`                                     | 1       | `MinSymbolsInString`                    | `mustIncludeSymbols(n)`                |
| Minimum Uppercase  | `MIN_UPPERCASE` / `--min-uppercase`                                 | 1       | `MinUppercaseLettersInString`           | `mustIncludeUppercase(n)`              |
| Minimum Lowercase  | `MIN_LOWERCASE` / `--min-lowercase`                                 | 1       | `MinLowercaseLettersInString`           | `mustIncludeLowercase(n)`              |
| Username Exclusion | `PASSWORD_CAN_INCLUDE_USERNAME` / `--password-can-include-username` | false   | `!strings.Contains(username, password)` | `mustNotIncludeUsername` (conditional) |

### Symbol Character Set

**ASCII Ranges** (matching implementation in validators.go:14-23 and validators.ts:1-23):

- `!` to `/` (ASCII 33-47): `! " # $ % & ' ( ) * + , - . /`
- `:` to `@` (ASCII 58-64): `: ; < = > ? @`
- `[` to `` ` `` (ASCII 91-96): `[ \ ] ^ \_ \``
- `{` to `~` (ASCII 123-126): `{ | } ~`

**Total Special Characters**: 32 symbols

### Cross-Field Validations

**Frontend Only**:

- `mustMatchNewPassword`: Ensures password confirmation matches new password
- `mustNotMatchCurrentPassword`: Prevents reusing current password

**Backend**:

- Implicitly validates via LDAP authentication (current password must be correct)

## Error Handling

### Client-Side Error Display

**Location**: internal/web/static/js/app.ts:76-91

Errors are displayed below each input field in real-time:

- Red border around input container
- Error messages in red text (text-xs text-red-400)
- Submit button disabled while errors exist

### Server-Side Error Response

**Location**: internal/rpc/handler.go:33-46

All errors are wrapped in consistent JSON-RPC response format:

```go
return c.Status(http.StatusInternalServerError).JSON(JSONRPCResponse{
  Success: false,
  Data:    []string{err.Error()},
})
```

## Security Considerations

### Authentication

- **change-password**: Current password required for authentication
- **Password reset**: Token-based authentication (no password needed)
- LDAP server performs authentication (no password storage in application)
- Password transmitted via HTTPS (enforced by LDAPS requirement)

### Authorization

- **change-password**: Users can only change their own password
- **Password reset**: Token grants temporary authorization to reset password
- **LDAP Accounts**:
  - LDAP_READONLY_USER: Read-only access for authentication
  - LDAP_RESET_USER (optional): Write access ONLY for password reset (principle of least privilege)
- No administrative privileges exposed via API

### Password Reset Security

- **Token Generation**: Cryptographically secure (crypto/rand, 32 bytes)
- **Token Expiration**: 15 minutes (configurable)
- **Single-Use Tokens**: Cannot be reused after password reset
- **Rate Limiting**: 3 requests/hour per email (prevents abuse)
- **User Enumeration Prevention**: Generic responses don't reveal if email exists
- **LDAP Permissions**: Dedicated LDAP_RESET_USER recommended for security isolation (falls back to LDAP_READONLY_USER)

### Input Validation

- Request body size limited to 4KB (main.go:31)
- All parameters validated before LDAP operation
- No SQL injection risk (LDAP-based, not SQL)
- XSS protection via proper input handling

### Transport Security

- LDAPS (LDAP over SSL/TLS) required for ActiveDirectory
- HTTPS recommended for web frontend
- Credentials never logged or stored

## Performance Considerations

### Compression

**Location**: main.go:34-36

All responses compressed with Brotli:

```go
app.Use(compress.New(compress.Config{
  Level: compress.LevelBestSpeed,
}))
```

### Caching

**Location**: main.go:38-41

Static assets cached for 24 hours:

```go
app.Use("/static", filesystem.New(filesystem.Config{
  Root:   http.FS(static.Static),
  MaxAge: 24 * 60 * 60,
}))
```

### Body Limits

**Location**: main.go:29-32

Request size limited to prevent abuse:

```go
app := fiber.New(fiber.Config{
  BodyLimit: 4 * 1024, // 4KB
})
```

## Testing

### Manual Testing

```bash
# Successful password change
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "method": "change-password",
    "params": ["testuser", "OldPass123!", "NewPass456!"]
  }'

# Expected response:
# {"success":true,"data":["password changed successfully"]}

# Validation error (password too short)
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "method": "change-password",
    "params": ["testuser", "OldPass123!", "short"]
  }'

# Expected response:
# {"success":false,"data":["the new password must be at least 8 characters long"]}

# Method not found
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "method": "invalid-method",
    "params": []
  }'

# Expected response:
# {"success":false,"data":["method not found"]}

# Request password reset
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "method": "request-password-reset",
    "params": ["user@example.com"]
  }'

# Expected response (same for valid or invalid email):
# {"success":true,"data":["If an account exists, a reset email has been sent"]}

# Reset password with valid token
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "method": "reset-password",
    "params": ["dGVzdHRva2VuMTIzNDU2Nzg5MGFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6", "NewPass123!"]
  }'

# Expected response:
# {"success":true,"data":["Password reset successfully. You can now login."]}

# Reset password with invalid/expired token
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "method": "reset-password",
    "params": ["invalid_token", "NewPass123!"]
  }'

# Expected response:
# {"success":false,"data":["Invalid or expired token"]}

# Reset password with password policy violation
curl -X POST http://localhost:3000/api/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "method": "reset-password",
    "params": ["valid_token", "short"]
  }'

# Expected response:
# {"success":false,"data":["the new password must be at least 8 characters long"]}

# Test rate limiting (send 4 requests quickly)
for i in {1..4}; do
  curl -X POST http://localhost:3000/api/rpc \
    -H "Content-Type: application/json" \
    -d '{
      "method": "request-password-reset",
      "params": ["user@example.com"]
    }'
  echo ""
done

# Expected: First 3 requests succeed, 4th still returns success but no email sent
```

### Integration Testing

No integration tests currently implemented. See [Testing Guide](testing-guide.md) for recommendations.

## Related Documentation

- [Architecture Patterns](architecture-patterns.md) - Design decisions and patterns
- [Development Guide](development-guide.md) - Setup and workflow
- [Testing Guide](testing-guide.md) - Testing strategies
- [Component Reference](component-reference.md) - Detailed component documentation

---

_Generated by /sc:index on 2025-10-04_
