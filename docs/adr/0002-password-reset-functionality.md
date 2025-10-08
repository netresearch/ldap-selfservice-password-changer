# ADR 0002: Self-Service Password Reset Functionality

**Status**: Accepted
**Date**: 2025-10-08
**Authors**: Sebastian Mendel

## Context

The LDAP self-service password changer application originally only supported authenticated password changes, requiring users to know their current password. When users forget their passwords, they must contact administrators for manual reset, creating support burden and user friction.

A self-service password reset feature reduces administrative overhead and improves user experience by allowing users to reset forgotten passwords independently through email verification.

### Requirements

- Users can request password reset via email address
- Secure token-based authentication for reset links
- Time-limited tokens to minimize attack surface
- Protection against user enumeration attacks
- Rate limiting to prevent abuse
- SMTP integration for email delivery
- LDAP integration to verify users and update passwords
- Configurable security parameters

## Decision

Implement token-based password reset with email verification following these architectural principles:

### 1. Security Model

**Token Generation**: Cryptographically secure random tokens
- 32 bytes (256 bits) from `crypto/rand`
- Base64 URL-safe encoding without padding
- No collision risk with sufficient entropy

**Token Storage**: In-memory thread-safe store
- `sync.RWMutex` for concurrent access protection
- Metadata: token, username, email, timestamps, used flag
- Automatic cleanup of expired tokens via background goroutine

**Token Lifecycle**:
- 15-minute expiration (configurable via `RESET_TOKEN_EXPIRY_MINUTES`)
- Single-use enforcement with `Used` flag
- Automatic periodic cleanup to prevent memory growth

### 2. User Enumeration Prevention

**Generic Response Pattern**: All requests return identical success message
- "If an account exists, a reset email has been sent"
- No indication whether email exists in system
- Internal logging only for debugging

**Silent Failure Handling**:
- User not found → generic success
- LDAP errors → generic success
- Email delivery failures → generic success
- Rate limiting → generic success

### 3. Rate Limiting

**Sliding Window Implementation**:
- Configurable limits (default: 3 requests per 60 minutes)
- Per-email/username tracking with `sync.RWMutex`
- Automatic cleanup of expired entries
- Prevents brute force and enumeration attempts

**Configuration Parameters**:
- `RESET_RATE_LIMIT_REQUESTS`: Max requests in window
- `RESET_RATE_LIMIT_WINDOW_MINUTES`: Time window duration

### 4. Email Delivery

**SMTP Integration**:
- Standard Go `net/smtp` library
- STARTTLS support for secure communication
- Plain auth with configurable credentials
- RFC 5322 compliant message formatting

**Email Content**:
- Clear reset link with embedded token
- 15-minute expiration notice
- Security reminder about ignoring unsolicited emails
- Plain text format for maximum compatibility

**Configuration Parameters**:
- `SMTP_HOST`, `SMTP_PORT`: Server connection details
- `SMTP_USERNAME`, `SMTP_PASSWORD`: Authentication credentials
- `SMTP_FROM_ADDRESS`: Sender email address
- `APP_BASE_URL`: Base URL for reset links

### 5. LDAP Integration

**User Lookup**:
- Find user by email address (`FindUserByMail`)
- Retrieve SAMAccountName for token association
- Validates user exists before token generation

**Password Update**:
- Uses readonly user credentials by default
- Optional dedicated reset service account (`LDAP_RESET_USER`) for separation of concerns
- Validates token before LDAP modification
- Enforces same password complexity rules as regular change

### 6. Architecture Components

**Package Structure**:
```
internal/resettoken/     # Token generation and storage
internal/email/          # SMTP service for sending emails
internal/ratelimit/      # Sliding window rate limiter
internal/rpc/           # RPC handlers (request, reset)
internal/options/        # Configuration parsing
```

**Interfaces for Testing**:
- `EmailService`: Email delivery abstraction
- `RateLimiter`: Rate limiting abstraction
- `TokenStore`: Token storage abstraction

### 7. Frontend Integration

**User Flow**:
1. User visits `/forgot-password`
2. Enters email address
3. System sends email (or silently fails)
4. User clicks link in email (`/reset-password?token=XXX`)
5. User enters new password (validates complexity)
6. System validates token and updates LDAP password

**Templates**:
- `forgot-password.html`: Email input form
- `reset-password.html`: New password form
- Atomic design pattern with reusable components

## Consequences

### Positive

1. **Reduced Administrative Burden**: Users can reset passwords independently without admin intervention

2. **Improved Security Posture**:
   - No credentials transmitted over insecure channels
   - Time-limited tokens minimize attack surface
   - Rate limiting prevents brute force attempts
   - User enumeration protection prevents reconnaissance

3. **Better User Experience**:
   - Self-service reduces wait time for password resets
   - Email-based verification familiar to users
   - Clear communication about security

4. **Operational Flexibility**:
   - Feature toggle via `PASSWORD_RESET_ENABLED` flag
   - Configurable security parameters adapt to threat model
   - Optional dedicated service account for security separation

5. **Code Quality**:
   - Clean package boundaries with clear responsibilities
   - Testable design with interface abstractions
   - Thread-safe concurrent operation
   - Comprehensive test coverage

### Neutral

1. **In-Memory Storage**: Tokens stored in memory, not persistent database
   - Trade-off: Simplicity vs persistence across restarts
   - Acceptable: 15-minute lifetime, low volume
   - Limitation: Tokens lost on server restart (users request new ones)

2. **SMTP Dependency**: Requires external SMTP server configuration
   - Necessary: Email delivery fundamental to design
   - Flexible: Supports any SMTP provider

### Negative

1. **Email Delivery Reliability**: Silent failures if SMTP misconfigured
   - Mitigation: Comprehensive logging for debugging
   - Monitoring: Check logs for `password_reset_email_failed` events

2. **Memory Footprint**: Token storage consumes memory proportional to active reset requests
   - Mitigation: Automatic cleanup of expired tokens
   - Scale: 1000 concurrent tokens ≈ 100KB (negligible)

3. **Single Point of Failure**: Application restart clears all pending tokens
   - Impact: Users must request new reset links
   - Acceptable: 15-minute window minimizes disruption

4. **No Admin Approval**: Phase 1 implementation provides immediate reset
   - Future: `RequiresApproval` flag designed for Phase 2
   - Security: Rate limiting and email verification provide baseline protection

## Alternatives Considered

### Alternative 1: Security Questions
**Rejected**:
- Weak security (answers often guessable or publicly available)
- Poor UX (users forget answers)
- Storage complexity (encrypted answers in database)

### Alternative 2: SMS/Two-Factor Authentication
**Rejected**:
- Requires phone number collection and storage
- SMS delivery costs and reliability issues
- Regulatory compliance complexity (GDPR, telecoms)
- Overkill for internal LDAP system

### Alternative 3: Admin-Only Reset
**Rejected**:
- Original problem: high administrative burden
- Doesn't scale with organization size
- Creates bottleneck and user friction

### Alternative 4: Persistent Database Storage
**Rejected**:
- Adds database dependency and complexity
- Unnecessary for 15-minute token lifecycle
- In-memory sufficient for expected load
- Can revisit if scale requirements change

### Alternative 5: Magic Links (Passwordless)
**Rejected**:
- Requires session management complexity
- Doesn't match LDAP authentication model
- Users expect password-based auth for enterprise systems

## Implementation Timeline

**Phase 1** (Current):
- Token-based reset with email verification
- In-memory token storage
- Rate limiting and enumeration protection
- SMTP email delivery
- Configurable security parameters

**Phase 2** (Future):
- Admin approval workflow (designed but not implemented)
- Enhanced monitoring and alerting
- Audit trail for compliance
- Multi-language email templates

## Configuration Reference

### Required (when `PASSWORD_RESET_ENABLED=true`):
- `SMTP_FROM_ADDRESS`: Sender email address
- `APP_BASE_URL`: Application base URL for links

### Security Configuration:
- `RESET_TOKEN_EXPIRY_MINUTES`: Token validity (default: 15)
- `RESET_RATE_LIMIT_REQUESTS`: Max requests per window (default: 3)
- `RESET_RATE_LIMIT_WINDOW_MINUTES`: Rate limit window (default: 60)

### SMTP Configuration:
- `SMTP_HOST`: SMTP server hostname (default: smtp.gmail.com)
- `SMTP_PORT`: SMTP server port (default: 587)
- `SMTP_USERNAME`: SMTP authentication username (optional)
- `SMTP_PASSWORD`: SMTP authentication password (optional)

### LDAP Configuration (Optional):
- `LDAP_RESET_USER`: Dedicated service account for resets
- `LDAP_RESET_PASSWORD`: Password for dedicated account
- Falls back to `LDAP_READONLY_USER` if not specified

## References

- [OWASP Forgot Password Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html)
- [RFC 5322 - Internet Message Format](https://tools.ietf.org/html/rfc5322)
- [NIST SP 800-63B - Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [Go crypto/rand Package Documentation](https://pkg.go.dev/crypto/rand)
