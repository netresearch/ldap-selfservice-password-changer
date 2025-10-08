# Security Policy

## Supported Versions

We release security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

**Recommendation**: Always use the latest version for security updates and improvements.

---

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

We take security seriously and appreciate responsible disclosure of vulnerabilities. If you discover a security issue, please report it privately to allow us to address it before public disclosure.

### How to Report

**Preferred Method: GitHub Security Advisories**

1. Go to the [Security tab](https://github.com/netresearch/ldap-selfservice-password-changer/security)
2. Click "Report a vulnerability"
3. Fill out the form with:
   - **Description**: Clear explanation of the vulnerability
   - **Impact**: Potential consequences (data exposure, DoS, etc.)
   - **Steps to Reproduce**: Detailed reproduction steps
   - **Affected Versions**: Which versions are vulnerable
   - **Suggested Fix**: If you have a solution (optional)

**Alternative: Email**

If you cannot use GitHub Security Advisories, email the maintainers at:
- **Security Contact**: [Create issue requesting contact email]

### What to Include

Please provide as much information as possible:

- **Vulnerability Type**: SQL injection, XSS, authentication bypass, etc.
- **Attack Vector**: How an attacker could exploit this
- **Impact Assessment**: Severity (low/medium/high/critical)
- **Proof of Concept**: Steps or code to demonstrate the issue
- **Suggested Mitigation**: If you have recommendations
- **Disclosure Timeline**: When you plan to publicly disclose (if applicable)

### Example Report Template

```
**Summary**: Brief one-line description

**Severity**: Critical / High / Medium / Low

**Description**:
Detailed explanation of the vulnerability

**Steps to Reproduce**:
1. Step one
2. Step two
3. Step three

**Impact**:
What an attacker could achieve by exploiting this

**Affected Component**:
Which module/file/function is vulnerable

**Suggested Fix**:
How to mitigate or fix this issue

**Disclosure Plan**:
When you plan to disclose publicly (90 days is standard)
```

---

## Response Process

### Timeline

- **Initial Response**: Within 48 hours
- **Triage and Assessment**: Within 5 business days
- **Fix Development**: Depends on severity and complexity
- **Release**: As soon as fix is tested and validated

### Severity Assessment

We use the following criteria to assess vulnerability severity:

**Critical**:
- Remote code execution
- Authentication bypass
- Data breach affecting multiple users
- Complete system compromise

**High**:
- Privilege escalation
- SQL injection
- LDAP injection
- Session hijacking
- Sensitive data exposure

**Medium**:
- XSS vulnerabilities
- CSRF vulnerabilities
- Rate limit bypass
- Information disclosure

**Low**:
- Minor information leaks
- Configuration issues
- Non-exploitable edge cases

### Disclosure Coordination

1. **Private Disclosure**: We will work with you privately to understand and fix the issue
2. **Fix Development**: We develop and test a fix
3. **Security Advisory**: We prepare a security advisory
4. **Coordinated Release**: We release the fix and advisory together
5. **Credit**: We credit you in the advisory (if desired)

**Standard Disclosure Timeline**: 90 days from initial report

---

## Security Best Practices for Users

### Deployment Security

**Required**:
- ✅ **HTTPS Only**: Never deploy without TLS/SSL
- ✅ **LDAPS**: Use encrypted LDAP connections (port 636)
- ✅ **SMTP TLS**: Enable TLS for email delivery
- ✅ **Strong Secrets**: Use strong passwords for service accounts
- ✅ **Firewall Rules**: Restrict network access to necessary ports

**Recommended**:
- ⚠️ **Dedicated Service Accounts**: Use separate LDAP accounts for read/write operations
- ⚠️ **Rate Limiting**: Configure reverse proxy rate limits in addition to application limits
- ⚠️ **IP Whitelisting**: Restrict access to known IP ranges if possible
- ⚠️ **Security Headers**: Configure all recommended headers in reverse proxy
- ⚠️ **Certificate Pinning**: For self-signed LDAP certificates

### Configuration Security

**Environment Variables**:
```bash
# ✅ Good: Use secret managers
LDAP_PASSWORD=$(vault kv get -field=password secret/ldap)

# ❌ Bad: Hardcode secrets
LDAP_PASSWORD=supersecret123
```

**Secrets Management**:
- Never commit `.env.local` to version control
- Use Docker secrets or Kubernetes secrets in production
- Rotate credentials regularly (every 90 days)
- Use separate service accounts for read/write operations

**LDAP Security**:
```bash
# ✅ Good: LDAPS with certificate validation
LDAP_SERVER=ldaps://ldap.example.com:636

# ❌ Bad: Unencrypted LDAP
LDAP_SERVER=ldap://ldap.example.com:389
```

### Monitoring and Detection

**Monitor for**:
- High volume of password reset requests from single IP
- Failed LDAP authentication attempts
- SMTP delivery failures
- Rate limit threshold hits
- Certificate expiration warnings

**Alerting**:
```yaml
# Example: Alert on rate limit exceeded
alert: HighResetRequestRate
expr: rate(password_reset_requests[5m]) > 10
severity: warning
description: "Unusual password reset activity detected"
```

---

## Known Security Considerations

### Design Decisions

**Email Enumeration Protection**:
- Password reset always returns success (prevents email discovery)
- Timing attacks mitigated by consistent response times
- Rate limiting prevents mass enumeration

**Token Security**:
- 256-bit cryptographically random tokens (2^256 combinations)
- Single-use tokens (deleted after consumption)
- Short expiration (default 15 minutes)
- No token persistence (in-memory only)

**Password Security**:
- Passwords never stored in application
- Transmitted over TLS only (HTTPS + LDAPS)
- Password policy enforced client and server-side
- LDAP handles password hashing

**Session Security**:
- Stateless application (no session management)
- No cookies (no session hijacking risk)
- No authentication state stored

### Attack Surface Analysis

**Public Endpoints**:
- `/` - Password change page (requires current password)
- `/forgot-password` - Request password reset (rate limited)
- `/reset-password` - Complete password reset (requires token)
- `/api/rpc` - JSON-RPC API (authentication via LDAP)

**Dependencies**:
- Go standard library
- Fiber v2 (web framework)
- simple-ldap-go (LDAP client)
- Minimal external dependencies

**Container Security**:
- Runs as non-root user (UID 65534)
- Scratch-based image (minimal attack surface)
- No shell or package manager
- Read-only recommended

---

## Security Updates

### Notification

Security updates are announced via:
- GitHub Security Advisories
- GitHub Releases (tagged as security release)
- CHANGELOG.md with [SECURITY] prefix

### Update Process

**For Users**:
1. Check [Security Advisories](https://github.com/netresearch/ldap-selfservice-password-changer/security/advisories)
2. Review CHANGELOG for security fixes
3. Pull latest Docker image or rebuild from source
4. Test in staging environment
5. Deploy to production

**For Developers**:
```bash
# Check for updates
git fetch origin
git log --oneline main..origin/main | grep -i security

# Update dependencies
go get -u all
pnpm update

# Run security scans
gosec ./...
pnpm audit
```

---

## Security Testing

### Automated Scanning

**Go Security**:
```bash
# Vulnerability scanning
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Security linting
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...
```

**Node Dependencies**:
```bash
# Audit dependencies
pnpm audit

# Fix vulnerabilities
pnpm audit --fix
```

**Container Scanning**:
```bash
# Trivy
docker run aquasec/trivy image \
  ghcr.io/netresearch/ldap-selfservice-password-changer:latest

# Grype
grype ghcr.io/netresearch/ldap-selfservice-password-changer:latest
```

### Manual Testing

**Recommended Tools**:
- OWASP ZAP for DAST scanning
- Burp Suite for penetration testing
- Nikto for web server scanning

**Test Checklist**:
- [ ] HTTPS enforcement
- [ ] Security headers
- [ ] Rate limiting effectiveness
- [ ] Token unpredictability
- [ ] LDAP injection prevention
- [ ] XSS prevention
- [ ] Email enumeration protection
- [ ] Authentication bypass attempts

---

## Compliance

### Standards

This application is designed with the following standards in mind:

- **OWASP Top 10 2021**: Mitigations for all top 10 vulnerabilities
- **OWASP ASVS**: Application Security Verification Standard compliance
- **NIST Cybersecurity Framework**: Risk management alignment
- **CIS Controls**: Security best practices

See [docs/security.md](docs/security.md) for detailed compliance documentation.

### Certifications

**WCAG 2.2 Level AAA**: Accessibility compliance
- No impact on security
- Ensures application usable by all users
- Prevents exclusion of users with disabilities

---

## Bug Bounty

**Status**: No formal bug bounty program at this time

We appreciate security research but do not currently offer monetary rewards. We do offer:
- Public acknowledgment in security advisories (if desired)
- Credit in CHANGELOG.md
- Our sincere gratitude for responsible disclosure

---

## Security Contacts

- **Security Issues**: Use GitHub Security Advisories (preferred)
- **General Security Questions**: Open a GitHub Discussion
- **Sensitive Information**: Request security contact email via issue

---

## Additional Resources

- [Security Documentation](docs/security.md) - Comprehensive threat model and mitigations
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [CWE Top 25](https://cwe.mitre.org/top25/)

---

**Last Updated**: 2025-10-08

Thank you for helping keep LDAP Selfservice Password Changer secure!
