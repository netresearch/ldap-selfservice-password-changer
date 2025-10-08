# Security Documentation

**Security architecture, threat model, and mitigation strategies for LDAP Selfservice Password Changer.**

---

## 🎯 Security Overview

### Security Goals

1. **Confidentiality**: Protect user credentials and password reset tokens
2. **Integrity**: Ensure password changes are authorized and validated
3. **Availability**: Prevent denial of service through rate limiting
4. **Non-Repudiation**: Log all password change attempts with client IP

### Security Principles

- **Defense in Depth**: Multiple layers of security controls
- **Least Privilege**: Service accounts with minimal required permissions
- **Secure by Default**: Safe default configuration out of the box
- **Zero Trust**: Validate all inputs, encrypt all communications
- **Fail Securely**: Errors don't leak sensitive information

---

## 🔐 Threat Model

### Assets

| Asset | Sensitivity | Protection |
|-------|-------------|------------|
| User passwords | **Critical** | Never stored, transmitted over TLS only |
| LDAP credentials | **Critical** | Environment variables, never logged |
| Reset tokens | **High** | Cryptographically random, single-use, time-limited |
| User email addresses | **Medium** | Protected by rate limiting, no enumeration |
| SMTP credentials | **Medium** | Environment variables, never logged |

### Threat Actors

**External Attackers**:
- **Motivation**: Account takeover, data breach, service disruption
- **Capabilities**: Network access, automated tools, credential stuffing
- **Mitigations**: Rate limiting, HTTPS, strong password policy

**Malicious Insiders**:
- **Motivation**: Privilege escalation, data theft
- **Capabilities**: Network access, knowledge of infrastructure
- **Mitigations**: Audit logging, least privilege service accounts

**Accidental Misuse**:
- **Motivation**: None (human error)
- **Capabilities**: Legitimate access
- **Mitigations**: Input validation, user-friendly error messages

### Attack Scenarios

#### 1. Credential Brute Force
**Attack**: Attacker tries multiple password combinations to gain access.

**Mitigations**:
- No login functionality - users authenticate via LDAP directly
- Password change requires current password knowledge
- LDAP server enforces account lockout policies

**Residual Risk**: Low - LDAP handles authentication

---

#### 2. Password Reset Token Theft
**Attack**: Attacker intercepts or guesses password reset tokens.

**Mitigations**:
- 256-bit cryptographically random tokens (2^256 possible values)
- Tokens transmitted via email (out-of-band verification)
- Single-use tokens deleted after consumption
- Short expiration (default 15 minutes)
- HTTPS required for token submission

**Residual Risk**: Low - requires email compromise or timing attack

**Risk Calculation**:
```
Token entropy: 256 bits = 2^256 combinations
Guessing rate: 1000 attempts/sec (aggressive)
Time to guess: 2^256 / 1000 ≈ 10^73 years
```

---

#### 3. Email Enumeration
**Attack**: Attacker discovers valid email addresses by testing reset requests.

**Mitigations**:
- Always return success response regardless of email validity
- Rate limiting prevents mass enumeration (3 requests/hour per IP)
- No timing differences between valid/invalid emails

**Residual Risk**: Medium - rate limiting can be circumvented with distributed IPs

**Detection**: Monitor for:
- High volume of reset requests from single IP
- Pattern of sequential email tests
- Requests from known bad actors (IP reputation)

---

#### 4. Denial of Service (DoS)
**Attack**: Overwhelm application with requests to disrupt availability.

**Mitigations**:
- Rate limiting per IP address (3 requests/hour for reset)
- Reverse proxy rate limiting (recommended 10 req/sec globally)
- Lightweight Go application with low resource footprint
- Stateless design enables horizontal scaling

**Residual Risk**: Medium - DDoS requires network-level mitigation

**Recommendations**:
- Deploy behind CDN (Cloudflare, Fastly)
- Configure reverse proxy connection limits
- Implement IP reputation filtering

---

#### 5. Man-in-the-Middle (MITM)
**Attack**: Intercept traffic to steal credentials or tokens.

**Mitigations**:
- HTTPS enforced via reverse proxy
- HSTS headers prevent protocol downgrade
- LDAPS (TLS) for all LDAP communications
- SMTP TLS for email delivery

**Residual Risk**: Low - requires certificate authority compromise

---

#### 6. Cross-Site Scripting (XSS)
**Attack**: Inject malicious scripts to steal session data or credentials.

**Mitigations**:
- No session cookies (stateless application)
- Content Security Policy headers (via reverse proxy)
- Go's `html/template` package auto-escapes output
- TypeScript strict mode prevents injection

**Residual Risk**: Very Low - no session state to steal

---

#### 7. LDAP Injection
**Attack**: Manipulate LDAP queries through malicious input.

**Mitigations**:
- `simple-ldap-go` library escapes all user inputs
- Username/email validated before LDAP queries
- No dynamic filter construction from user input

**Residual Risk**: Very Low - library handles escaping

**Example Attack (Prevented)**:
```
Input: user@example.com)(uid=*
Query: (&(mail=user@example.com)(uid=*))(objectClass=person))
Result: Library escapes parentheses, preventing injection
```

---

#### 8. Privilege Escalation
**Attack**: Normal user gains admin privileges or changes others' passwords.

**Mitigations**:
- Password change requires current password (self-service only)
- Password reset requires token from user's email
- No admin interface or elevated privileges
- LDAP enforces access control

**Residual Risk**: Very Low - LDAP controls authorization

---

#### 9. Token Replay Attack
**Attack**: Reuse captured reset token to change password multiple times.

**Mitigations**:
- Tokens are single-use (deleted after consumption)
- Token validation checks expiration before use
- No token persistence across restarts (in-memory only)

**Residual Risk**: Very Low - token cannot be reused

---

#### 10. Information Disclosure
**Attack**: Extract sensitive information from error messages or logs.

**Mitigations**:
- Generic error messages to users ("An error occurred")
- Detailed errors only in server logs (not exposed to client)
- No password or token values in logs
- No stack traces exposed to users

**Residual Risk**: Low - requires server access

---

## 🛡️ Security Controls

### Authentication and Authorization

**Password Change Flow**:
1. User provides username, current password, new password
2. Application authenticates against LDAP using current password
3. If auth succeeds, password changed via LDAP modify operation
4. LDAP enforces password policy and access control

**Security Properties**:
- ✅ No password storage in application
- ✅ Authentication delegated to LDAP
- ✅ Authorization enforced by LDAP ACLs
- ✅ Passwords transmitted over TLS only

**Password Reset Flow**:
1. User provides email address
2. Application rate-limits request (3/hour per IP)
3. Lookup user by email in LDAP (read-only account)
4. Generate 256-bit cryptographic token
5. Store token in memory with email and expiration
6. Send token link via email (out-of-band)
7. User clicks link, submits token + new password
8. Application validates token, retrieves email
9. Lookup user DN in LDAP, reset password via admin account
10. Delete token (single-use)

**Security Properties**:
- ✅ Out-of-band verification via email
- ✅ Cryptographically random tokens
- ✅ Single-use tokens
- ✅ Time-limited tokens (default 15 min)
- ✅ Rate limiting prevents abuse
- ✅ No email enumeration (always return success)

### Cryptography

**Password Reset Tokens**:
```go
// internal/resettoken/token.go
func GenerateToken() (string, error) {
    bytes := make([]byte, 32) // 256 bits
    _, err := rand.Read(bytes) // crypto/rand (not math/rand)
    if err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}
```

**Properties**:
- ✅ `crypto/rand` provides cryptographically secure randomness
- ✅ 256-bit entropy (2^256 ≈ 10^77 combinations)
- ✅ URL-safe base64 encoding (43 characters)
- ✅ No predictable patterns

**Token Storage**:
```go
// internal/resettoken/store.go
type TokenData struct {
    Email     string
    ExpiresAt time.Time
}

// In-memory map with mutex protection
tokens map[string]TokenData
```

**Properties**:
- ✅ In-memory only (no persistence)
- ✅ Automatic expiration cleanup
- ✅ Thread-safe with RWMutex
- ✅ Lost on restart (security feature)

### Input Validation

**Password Validation** (`internal/validators`):
```go
// All validators return error if validation fails
ValidateMinLength(password string, minLength int) error
ValidateMinNumbers(password string, minNumbers int) error
ValidateMinSymbols(password string, minSymbols int) error
ValidateMinUppercase(password string, minUppercase int) error
ValidateMinLowercase(password string, minLowercase int) error
ValidateNoUsername(password, username string) error
```

**Default Policy**:
- Minimum length: 8 characters
- Minimum numbers: 1
- Minimum symbols: 1
- Minimum uppercase: 1
- Minimum lowercase: 1
- Username exclusion: enabled

**Properties**:
- ✅ Client-side validation (UX feedback)
- ✅ Server-side validation (security enforcement)
- ✅ Configurable via environment variables
- ✅ 100% test coverage

**Email Validation**:
```typescript
// internal/web/static/js/validators.ts
export const isValidEmail = (email: string): string => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(email)) {
    return "Email must be a valid email address";
  }
  return "";
};
```

**Properties**:
- ✅ Client-side for UX
- ✅ LDAP lookup provides server-side validation
- ✅ Prevents injection via email field

### Rate Limiting

**Implementation** (`internal/ratelimit`):
```go
type Limiter struct {
    maxRequests int           // 3 (default)
    window      time.Duration // 1 hour (default)
    requests    map[string][]time.Time // IP -> timestamps
    mu          sync.RWMutex
}

func (l *Limiter) Allow(ip string) bool {
    // Sliding window algorithm
    // Remove expired requests
    // Check if under limit
}
```

**Configuration**:
```bash
RESET_RATE_LIMIT_REQUESTS=3
RESET_RATE_LIMIT_WINDOW_MINUTES=60
```

**Properties**:
- ✅ Per-IP rate limiting
- ✅ Sliding window algorithm (more accurate than fixed window)
- ✅ Automatic cleanup of expired entries
- ✅ Thread-safe concurrent access
- ✅ Memory-bounded (old entries cleaned)

**Effectiveness**:
- Prevents mass password reset abuse
- Limits email enumeration attempts
- Does not prevent distributed attacks (use reverse proxy for that)

**Limitations**:
- IP-based (can be circumvented with proxies/VPNs)
- Shared IP (NAT) may affect legitimate users
- Recommend: Combine with reverse proxy rate limiting

### Transport Security

**HTTPS** (via reverse proxy):
- TLS 1.2 minimum (TLS 1.3 recommended)
- Strong cipher suites only
- HSTS headers (`max-age=31536000; includeSubDomains`)
- Certificate from trusted CA (Let's Encrypt recommended)

**LDAPS**:
- TLS encryption for all LDAP traffic
- Certificate validation (system CA bundle)
- Custom CA support for self-signed certs
- No fallback to unencrypted LDAP

**SMTP TLS**:
- STARTTLS for email delivery
- Opportunistic TLS (fails if unavailable)
- No plain-text email transmission

### Secrets Management

**Environment Variables**:
```bash
# Sensitive values never hardcoded
LDAP_READONLY_PASSWORD=secret
LDAP_RESET_PASSWORD=secret
SMTP_PASSWORD=secret
```

**Best Practices**:
- ✅ `.env.local` gitignored
- ✅ Never commit secrets to version control
- ✅ Rotate credentials regularly
- ✅ Use secret managers in production (Vault, AWS Secrets Manager)

**Docker Secrets** (Swarm/Kubernetes):
```yaml
environment:
  LDAP_READONLY_PASSWORD_FILE=/run/secrets/ldap_password
secrets:
  - ldap_password
```

**Logging Safety**:
```go
// Passwords and tokens never logged
log.Printf("Password change for user: %s", username) // ✅ Safe
log.Printf("Password: %s", password) // ❌ Never done
```

### Container Security

**Dockerfile** (multi-stage build):
```dockerfile
# Final stage: scratch (minimal attack surface)
FROM scratch AS runner

# Non-root user
USER 65534:65534

# Read-only filesystem
# No shell, no package manager, no utilities
```

**Properties**:
- ✅ Minimal attack surface (only Go binary + CA certs)
- ✅ Non-root execution (UID 65534 = nobody)
- ✅ No shell (prevents shell injection)
- ✅ Static binary (no dynamic library attacks)
- ✅ Immutable (read-only filesystem recommended)

**Kubernetes Security Context**:
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
```

### HTTP Security Headers

**Configured via reverse proxy**:
```nginx
# Prevent clickjacking
add_header X-Frame-Options "DENY" always;

# Prevent MIME sniffing
add_header X-Content-Type-Options "nosniff" always;

# XSS protection
add_header X-XSS-Protection "1; mode=block" always;

# HTTPS enforcement
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

# Control referrer information
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
```

**Content Security Policy** (optional):
```nginx
# Restrict resource loading
add_header Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'" always;
```

---

## 🔍 OWASP Top 10 Mitigations

### A01:2021 - Broken Access Control

**Risk**: Users access resources or perform actions without authorization.

**Mitigations**:
- ✅ Password change requires current password (can't change others' passwords)
- ✅ Password reset requires token from user's email
- ✅ LDAP enforces access control policies
- ✅ No admin interface or elevated privileges
- ✅ Service accounts use least privilege

**Status**: ✅ **Not Vulnerable**

---

### A02:2021 - Cryptographic Failures

**Risk**: Sensitive data exposed due to weak cryptography.

**Mitigations**:
- ✅ Passwords never stored (only transmitted to LDAP over TLS)
- ✅ Reset tokens use `crypto/rand` (256-bit entropy)
- ✅ HTTPS required for all web traffic
- ✅ LDAPS required for all LDAP traffic
- ✅ SMTP TLS for email delivery

**Status**: ✅ **Not Vulnerable**

---

### A03:2021 - Injection

**Risk**: Malicious input executed as code or commands.

**Mitigations**:
- ✅ LDAP queries use parameterized library (simple-ldap-go)
- ✅ HTML templates auto-escape output (Go html/template)
- ✅ Input validation on all user inputs
- ✅ No SQL database (no SQL injection risk)
- ✅ No shell execution from user input

**Status**: ✅ **Not Vulnerable**

---

### A04:2021 - Insecure Design

**Risk**: Flawed design enables attacks.

**Mitigations**:
- ✅ Threat modeling performed
- ✅ Security controls designed into architecture
- ✅ Rate limiting prevents abuse
- ✅ Out-of-band verification (email) for password reset
- ✅ Defense in depth approach

**Status**: ✅ **Not Vulnerable**

---

### A05:2021 - Security Misconfiguration

**Risk**: Insecure default settings or incomplete configurations.

**Mitigations**:
- ✅ Secure defaults (LDAPS, rate limiting enabled)
- ✅ Configuration validation on startup
- ✅ No debug mode in production
- ✅ Minimal attack surface (scratch container)
- ✅ Security headers configured

**Recommendations**:
- ⚠️ Ensure reverse proxy properly configured
- ⚠️ Rotate secrets regularly
- ⚠️ Monitor for configuration drift

**Status**: ✅ **Not Vulnerable** (with proper deployment)

---

### A06:2021 - Vulnerable and Outdated Components

**Risk**: Using libraries with known vulnerabilities.

**Mitigations**:
- ✅ Minimal dependencies (only 3 direct Go deps)
- ✅ Pinned Docker base images with SHA256
- ✅ Regular dependency updates
- ✅ Automated security scanning (Dependabot)

**Dependencies**:
```go
github.com/gofiber/fiber/v2 v2.52.5
github.com/joho/godotenv v1.5.1
github.com/netresearch/simple-ldap-go v1.0.0
```

**Recommendations**:
- ⚠️ Monitor security advisories
- ⚠️ Update dependencies monthly
- ⚠️ Run `go mod tidy` and rebuild regularly

**Status**: ✅ **Not Vulnerable** (requires maintenance)

---

### A07:2021 - Identification and Authentication Failures

**Risk**: Weak authentication or session management.

**Mitigations**:
- ✅ No session management (stateless application)
- ✅ No cookies (no session hijacking risk)
- ✅ LDAP handles authentication
- ✅ Password policy enforced (8+ chars, complexity)
- ✅ Rate limiting prevents credential stuffing

**Status**: ✅ **Not Vulnerable**

---

### A08:2021 - Software and Data Integrity Failures

**Risk**: Unverified updates or deserialization attacks.

**Mitigations**:
- ✅ Docker images signed and published to GitHub Container Registry
- ✅ No deserialization of untrusted data
- ✅ No file uploads
- ✅ JSON parsing uses standard library (safe)

**Status**: ✅ **Not Vulnerable**

---

### A09:2021 - Security Logging and Monitoring Failures

**Risk**: Attacks go undetected due to insufficient logging.

**Current State**:
- ✅ Password change attempts logged (username, IP)
- ✅ Reset requests logged (email, IP)
- ✅ LDAP errors logged
- ⚠️ No centralized logging
- ⚠️ No alerting on suspicious patterns

**Recommendations**:
- Configure centralized logging (Loki, Elasticsearch)
- Alert on high reset request volume
- Alert on LDAP authentication failures
- Monitor rate limit hits

**Status**: ⚠️ **Partially Mitigated** (requires monitoring setup)

---

### A10:2021 - Server-Side Request Forgery (SSRF)

**Risk**: Application makes unauthorized requests to internal resources.

**Mitigations**:
- ✅ No user-controlled URLs
- ✅ LDAP server configured via environment (not user input)
- ✅ SMTP server configured via environment
- ✅ No URL fetch functionality
- ✅ No webhooks or callbacks

**Status**: ✅ **Not Vulnerable**

---

## 📊 Security Testing

### Static Analysis

**Go**:
```bash
# Security linting
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...

# Dependency vulnerability scanning
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

**TypeScript**:
```bash
# Audit dependencies
pnpm audit

# Fix vulnerabilities
pnpm audit --fix
```

### Dynamic Analysis

**DAST Scanning**:
```bash
# OWASP ZAP
docker run -t zaproxy/zap-stable zap-baseline.py -t https://passwd.example.com

# Nikto
nikto -h https://passwd.example.com
```

**Penetration Testing Checklist**:
- [ ] SQL injection (N/A - no SQL)
- [ ] LDAP injection (test with special chars)
- [ ] XSS (test with `<script>alert(1)</script>`)
- [ ] CSRF (N/A - stateless)
- [ ] Authentication bypass
- [ ] Authorization bypass
- [ ] Session fixation (N/A - no sessions)
- [ ] Rate limit bypass
- [ ] Email enumeration
- [ ] Token prediction
- [ ] Information disclosure
- [ ] HTTPS enforcement
- [ ] Security headers

### Vulnerability Scanning

**Container Scanning**:
```bash
# Trivy
docker run aquasec/trivy image ghcr.io/netresearch/ldap-selfservice-password-changer:latest

# Grype
grype ghcr.io/netresearch/ldap-selfservice-password-changer:latest
```

**Expected Results**:
- Zero high/critical vulnerabilities in application code
- Possible low-severity findings in base images (acceptable)

---

## 📝 Security Checklist

### Deployment Security

**Pre-Deployment**:
- [ ] All environment variables configured
- [ ] Secrets not committed to version control
- [ ] LDAPS enabled and tested
- [ ] SMTP TLS enabled
- [ ] Rate limiting configured
- [ ] Password policy matches organizational requirements
- [ ] Service accounts created with least privilege
- [ ] Certificates valid and not expiring soon

**Infrastructure**:
- [ ] HTTPS enforced via reverse proxy
- [ ] Security headers configured
- [ ] Firewall rules allow only necessary ports
- [ ] Container runs as non-root
- [ ] Read-only filesystem enabled (if applicable)
- [ ] Resource limits configured (CPU/memory)
- [ ] Logging configured and tested
- [ ] Monitoring and alerting configured

**Post-Deployment**:
- [ ] Health checks passing
- [ ] SSL Labs grade A or higher
- [ ] Security headers verified (securityheaders.com)
- [ ] Rate limiting tested
- [ ] Password change flow tested
- [ ] Password reset flow tested
- [ ] LDAP connectivity verified
- [ ] SMTP delivery verified

### Maintenance Security

**Monthly**:
- [ ] Review access logs for anomalies
- [ ] Check for dependency updates
- [ ] Verify certificates not expiring soon
- [ ] Review rate limit effectiveness

**Quarterly**:
- [ ] Rotate LDAP service account passwords
- [ ] Rotate SMTP credentials
- [ ] Review and update security configurations
- [ ] Perform vulnerability scanning

**Annually**:
- [ ] Security audit / penetration test
- [ ] Review threat model
- [ ] Update incident response procedures
- [ ] Security awareness training

---

## 🚨 Incident Response

### Security Event Categories

**High Severity**:
- LDAP credential compromise
- Container escape
- Unauthorized password changes
- Mass password reset abuse

**Medium Severity**:
- Rate limit bypass
- Email enumeration
- SMTP credential compromise
- Certificate expiration

**Low Severity**:
- Failed login attempts
- Invalid reset requests
- Configuration errors

### Response Procedures

**LDAP Credential Compromise**:
1. Immediately rotate compromised credentials
2. Review logs for unauthorized password changes
3. Notify affected users
4. Update service account in all environments
5. Investigate root cause

**Mass Password Reset Abuse**:
1. Identify attacking IP addresses
2. Block IPs at firewall/reverse proxy level
3. Review rate limiting configuration
4. Consider reducing rate limits temporarily
5. Monitor for distributed attacks

**Certificate Expiration**:
1. Renew certificates immediately
2. Deploy new certificates
3. Verify HTTPS and LDAPS connectivity
4. Update monitoring to alert 30 days before expiration

---

## 🏛️ Compliance Considerations

### GDPR (EU)

**Data Processed**:
- User email addresses (for password reset)
- Client IP addresses (for rate limiting)
- Password change logs (username, timestamp, IP)

**Compliance**:
- ✅ No persistent storage of personal data
- ✅ Data minimization (only necessary data collected)
- ✅ No data sharing with third parties
- ✅ Right to erasure: data deleted on container restart
- ⚠️ Privacy policy required (organization responsibility)

### HIPAA (US Healthcare)

**Applicable If**: Application used for healthcare user accounts

**Requirements**:
- ✅ Access controls (LDAP enforced)
- ✅ Audit controls (logging)
- ✅ Integrity controls (validation)
- ✅ Transmission security (TLS)
- ⚠️ Requires Business Associate Agreement with SMTP provider

### PCI DSS (Payment Card Industry)

**Not Applicable**: Application does not process payment card data

### SOC 2

**Control Objectives**:
- ✅ CC6.1: Logical access controls implemented
- ✅ CC6.6: Encryption in transit (HTTPS, LDAPS)
- ✅ CC7.2: Vulnerability management (scanning, patching)
- ⚠️ CC7.3: Monitoring required (organization responsibility)

---

## 📚 Security References

### Standards and Frameworks

- **OWASP Top 10**: https://owasp.org/www-project-top-ten/
- **OWASP ASVS**: https://owasp.org/www-project-application-security-verification-standard/
- **NIST Cybersecurity Framework**: https://www.nist.gov/cyberframework
- **CIS Controls**: https://www.cisecurity.org/controls

### Tools

- **OWASP ZAP**: https://www.zaproxy.org/
- **gosec**: https://github.com/securego/gosec
- **govulncheck**: https://golang.org/x/vuln/cmd/govulncheck
- **Trivy**: https://github.com/aquasecurity/trivy

### Documentation

- **Development Guide**: [docs/development-guide.md](development-guide.md)
- **Deployment Guide**: [docs/deployment.md](deployment.md)
- **Architecture**: [docs/architecture.md](architecture.md)

---

**Last Updated**: 2025-10-08
**Security Contact**: See [SECURITY.md](../SECURITY.md) for vulnerability reporting
