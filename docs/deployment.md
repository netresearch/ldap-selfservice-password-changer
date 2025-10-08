# Deployment Guide

**Production deployment guide for LDAP Selfservice Password Changer.**

---

## 🚀 Quick Start

### Docker (Recommended)

```bash
# Pull latest image
docker pull ghcr.io/netresearch/ldap-selfservice-password-changer:latest

# Run with environment file
docker run -d \
  --name ldap-passwd \
  -p 3000:3000 \
  --env-file .env \
  ghcr.io/netresearch/ldap-selfservice-password-changer:latest
```

### Docker Compose

```bash
# Clone repository
git clone https://github.com/netresearch/ldap-selfservice-password-changer.git
cd ldap-selfservice-password-changer

# Configure environment
cp .env.local.example .env.local
# Edit .env.local with your LDAP/SMTP settings

# Start services
docker compose --profile dev up -d
```

### Binary (Bare Metal)

```bash
# Download latest release
wget https://github.com/netresearch/ldap-selfservice-password-changer/releases/latest/download/ldap-passwd-linux-amd64

# Make executable
chmod +x ldap-passwd-linux-amd64

# Run with environment variables
LDAP_SERVER=ldaps://ldap.example.com:636 \
LDAP_BASE_DN=dc=example,dc=com \
LDAP_READONLY_USER=cn=readonly,dc=example,dc=com \
LDAP_READONLY_PASSWORD=secret \
SMTP_HOST=smtp.example.com \
APP_BASE_URL=https://passwd.example.com \
./ldap-passwd-linux-amd64
```

---

## 📋 Prerequisites

### Required Infrastructure

- **LDAP/Active Directory Server**
  - LDAPS (port 636) recommended for production
  - Read-only service account for user lookups
  - Admin/reset account for password changes (optional but recommended)

- **SMTP Server**
  - For password reset email delivery
  - TLS/STARTTLS support recommended
  - Authentication optional (depends on server)

- **SSL/TLS Certificates**
  - For HTTPS termination (via reverse proxy)
  - For LDAPS connections (if using self-signed certs)

### System Requirements

**Minimal**:

- CPU: 1 core
- RAM: 128MB
- Disk: 50MB

**Recommended**:

- CPU: 2 cores
- RAM: 256MB
- Disk: 100MB
- Reverse proxy (nginx/Traefik)

---

## 🔧 Configuration

### Environment Variables

#### Required Variables

```bash
# LDAP Server Configuration
LDAP_SERVER=ldaps://ldap.example.com:636
LDAP_BASE_DN=dc=example,dc=com
LDAP_READONLY_USER=cn=readonly,dc=example,dc=com
LDAP_READONLY_PASSWORD=readonly_secret

# SMTP Configuration
SMTP_HOST=smtp.example.com
APP_BASE_URL=https://passwd.example.com
```

#### LDAP Configuration

```bash
# LDAP Server (required)
LDAP_SERVER=ldaps://ldap.example.com:636
# Supported schemes: ldap://, ldaps://
# Port 389 for LDAP, 636 for LDAPS

# Active Directory Mode (optional, default: false)
LDAP_IS_AD=true
# Set to true for Active Directory servers

# Base DN (required)
LDAP_BASE_DN=dc=example,dc=com
# Search base for user lookups

# Read-Only Service Account (required)
LDAP_READONLY_USER=cn=readonly,dc=example,dc=com
LDAP_READONLY_PASSWORD=readonly_secret
# Account with read permissions for user searches

# Password Reset Account (optional, recommended for production)
LDAP_RESET_USER=cn=password-reset,dc=example,dc=com
LDAP_RESET_PASSWORD=reset_secret
# Dedicated account with password modification permissions
# If not set, uses LDAP_READONLY_USER for both read and write
```

#### Password Policy

```bash
# Minimum password length (default: 8)
MIN_LENGTH=8

# Minimum number of digits (default: 1)
MIN_NUMBERS=1

# Minimum special characters (default: 1)
MIN_SYMBOLS=1

# Minimum uppercase letters (default: 1)
MIN_UPPERCASE=1

# Minimum lowercase letters (default: 1)
MIN_LOWERCASE=1

# Allow username in password (default: false)
PASSWORD_CAN_INCLUDE_USERNAME=false
```

#### Password Reset Feature

```bash
# Enable password reset functionality (default: true)
PASSWORD_RESET_ENABLED=true

# Token expiry in minutes (default: 15)
RESET_TOKEN_EXPIRY_MINUTES=15

# Rate limit: max requests per IP (default: 3)
RESET_RATE_LIMIT_REQUESTS=3

# Rate limit: time window in minutes (default: 60)
RESET_RATE_LIMIT_WINDOW_MINUTES=60
```

#### SMTP Configuration

```bash
# SMTP server hostname (required if PASSWORD_RESET_ENABLED=true)
SMTP_HOST=smtp.example.com

# SMTP port (default: 587)
SMTP_PORT=587

# SMTP authentication (optional, leave empty for no auth)
SMTP_USERNAME=user@example.com
SMTP_PASSWORD=smtp_secret

# Email sender (default: noreply@localhost)
SMTP_FROM_ADDRESS=noreply@example.com

# Application base URL (required for reset links)
APP_BASE_URL=https://passwd.example.com
```

#### Server Configuration

```bash
# HTTP port (default: 3000)
PORT=3000

# Trusted proxy IPs (optional, comma-separated)
TRUSTED_PROXIES=192.168.1.1,10.0.0.0/8
# Required if behind reverse proxy for accurate client IP detection
```

### Configuration Files

#### .env File Structure

```bash
# .env - Default values and documentation (committed to git)
# .env.local - Local overrides (gitignored, takes precedence)

# Copy example to get started
cp .env.local.example .env.local
# Edit .env.local with your values
```

#### Docker Compose Environment

```yaml
# compose.yml uses both files:
env_file:
  - .env # Defaults and docs (version controlled)
  - .env.local # Overrides (gitignored)
```

---

## 🐳 Docker Deployment

### Production Docker Compose

```yaml
# compose.production.yml
services:
  ldap-passwd:
    image: ghcr.io/netresearch/ldap-selfservice-password-changer:latest
    restart: unless-stopped
    ports:
      - "127.0.0.1:3000:3000" # Bind to localhost only
    env_file:
      - .env.production
    volumes:
      # Mount custom CA certificates if using self-signed LDAPS certs
      - ./certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt:ro
    networks:
      - ldap-network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3000/"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

networks:
  ldap-network:
    driver: bridge
```

### Custom CA Certificates

For self-signed LDAPS certificates:

```bash
# 1. Obtain CA certificate from LDAP server
openssl s_client -connect ldap.example.com:636 -showcerts < /dev/null 2>/dev/null | \
  openssl x509 -outform PEM > ldap-ca.crt

# 2. Append to system CA bundle
cat ldap-ca.crt >> /etc/ssl/certs/ca-certificates.crt

# 3. Mount into container
docker run -v /etc/ssl/certs:/etc/ssl/certs:ro ...
```

### Docker Secrets (Swarm/Kubernetes)

```yaml
# Use Docker secrets for sensitive data
services:
  ldap-passwd:
    image: ghcr.io/netresearch/ldap-selfservice-password-changer:latest
    secrets:
      - ldap_readonly_password
      - ldap_reset_password
      - smtp_password
    environment: LDAP_READONLY_PASSWORD_FILE=/run/secrets/ldap_readonly_password
      LDAP_RESET_PASSWORD_FILE=/run/secrets/ldap_reset_password
      SMTP_PASSWORD_FILE=/run/secrets/smtp_password

secrets:
  ldap_readonly_password:
    external: true
  ldap_reset_password:
    external: true
  smtp_password:
    external: true
```

---

## ☸️ Kubernetes Deployment

### Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ldap-passwd
  namespace: identity
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ldap-passwd
  template:
    metadata:
      labels:
        app: ldap-passwd
    spec:
      containers:
        - name: ldap-passwd
          image: ghcr.io/netresearch/ldap-selfservice-password-changer:latest
          ports:
            - containerPort: 3000
              name: http
          env:
            - name: LDAP_SERVER
              value: "ldaps://ldap.example.com:636"
            - name: LDAP_BASE_DN
              value: "dc=example,dc=com"
            - name: LDAP_READONLY_USER
              valueFrom:
                secretKeyRef:
                  name: ldap-credentials
                  key: readonly-user
            - name: LDAP_READONLY_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: ldap-credentials
                  key: readonly-password
            - name: SMTP_HOST
              value: "smtp.example.com"
            - name: APP_BASE_URL
              value: "https://passwd.example.com"
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /
              port: 3000
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /
              port: 3000
            initialDelaySeconds: 5
            periodSeconds: 10
          securityContext:
            runAsNonRoot: true
            runAsUser: 65534
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
---
apiVersion: v1
kind: Service
metadata:
  name: ldap-passwd
  namespace: identity
spec:
  selector:
    app: ldap-passwd
  ports:
    - port: 80
      targetPort: 3000
      name: http
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ldap-passwd
  namespace: identity
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - passwd.example.com
      secretName: ldap-passwd-tls
  rules:
    - host: passwd.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: ldap-passwd
                port:
                  number: 80
```

### Secrets

```bash
# Create secret from literals
kubectl create secret generic ldap-credentials \
  --namespace identity \
  --from-literal=readonly-user='cn=readonly,dc=example,dc=com' \
  --from-literal=readonly-password='secret' \
  --from-literal=reset-user='cn=password-reset,dc=example,dc=com' \
  --from-literal=reset-password='reset-secret'

# Or from files
kubectl create secret generic ldap-credentials \
  --namespace identity \
  --from-file=readonly-user=./readonly-user.txt \
  --from-file=readonly-password=./readonly-password.txt
```

---

## 🌐 Reverse Proxy Configuration

### Nginx

```nginx
# /etc/nginx/sites-available/ldap-passwd
server {
    listen 80;
    server_name passwd.example.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name passwd.example.com;

    # SSL Configuration
    ssl_certificate /etc/ssl/certs/passwd.example.com.crt;
    ssl_certificate_key /etc/ssl/private/passwd.example.com.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Logging
    access_log /var/log/nginx/ldap-passwd.access.log;
    error_log /var/log/nginx/ldap-passwd.error.log;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;

        # Preserve client information
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support (if needed in future)
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

Enable site:

```bash
ln -s /etc/nginx/sites-available/ldap-passwd /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

### Traefik (Docker Labels)

```yaml
services:
  ldap-passwd:
    image: ghcr.io/netresearch/ldap-selfservice-password-changer:latest
    labels:
      - "traefik.enable=true"

      # HTTP to HTTPS redirect
      - "traefik.http.routers.ldap-passwd-http.rule=Host(`passwd.example.com`)"
      - "traefik.http.routers.ldap-passwd-http.entrypoints=web"
      - "traefik.http.routers.ldap-passwd-http.middlewares=https-redirect"

      # HTTPS router
      - "traefik.http.routers.ldap-passwd.rule=Host(`passwd.example.com`)"
      - "traefik.http.routers.ldap-passwd.entrypoints=websecure"
      - "traefik.http.routers.ldap-passwd.tls=true"
      - "traefik.http.routers.ldap-passwd.tls.certresolver=letsencrypt"

      # Security headers
      - "traefik.http.routers.ldap-passwd.middlewares=security-headers"
      - "traefik.http.middlewares.security-headers.headers.stsSeconds=31536000"
      - "traefik.http.middlewares.security-headers.headers.stsIncludeSubdomains=true"
      - "traefik.http.middlewares.security-headers.headers.frameDeny=true"
      - "traefik.http.middlewares.security-headers.headers.contentTypeNosniff=true"

      # Service
      - "traefik.http.services.ldap-passwd.loadbalancer.server.port=3000"
    networks:
      - traefik-public
      - ldap-network

networks:
  traefik-public:
    external: true
  ldap-network:
    driver: bridge
```

### Apache

```apache
# /etc/apache2/sites-available/ldap-passwd.conf
<VirtualHost *:80>
    ServerName passwd.example.com
    Redirect permanent / https://passwd.example.com/
</VirtualHost>

<VirtualHost *:443>
    ServerName passwd.example.com

    # SSL Configuration
    SSLEngine on
    SSLCertificateFile /etc/ssl/certs/passwd.example.com.crt
    SSLCertificateKeyFile /etc/ssl/private/passwd.example.com.key
    SSLProtocol all -SSLv3 -TLSv1 -TLSv1.1
    SSLCipherSuite HIGH:!aNULL:!MD5

    # Security Headers
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
    Header always set X-Frame-Options "DENY"
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-XSS-Protection "1; mode=block"

    # Logging
    ErrorLog ${APACHE_LOG_DIR}/ldap-passwd-error.log
    CustomLog ${APACHE_LOG_DIR}/ldap-passwd-access.log combined

    # Proxy Configuration
    ProxyPreserveHost On
    ProxyPass / http://127.0.0.1:3000/
    ProxyPassReverse / http://127.0.0.1:3000/

    # Pass client IP
    RequestHeader set X-Forwarded-Proto "https"
    RequestHeader set X-Forwarded-For "%{REMOTE_ADDR}s"
</VirtualHost>
```

Enable modules and site:

```bash
a2enmod ssl proxy proxy_http headers
a2ensite ldap-passwd
apachectl configtest
systemctl reload apache2
```

---

## 🔐 LDAP Server Setup

### Service Account Permissions

#### Read-Only Account

**Required permissions**:

- Read access to user objects in Base DN
- Read `mail` attribute for email lookup
- Read `userPrincipalName` or `uid` for authentication

**OpenLDAP ACL**:

```ldif
# /etc/ldap/slapd.conf or cn=config
access to dn.subtree="dc=example,dc=com"
    by dn="cn=readonly,dc=example,dc=com" read
    by * none
```

**Active Directory**:

```powershell
# Grant read permissions to service account
$user = Get-ADUser -Identity "readonly"
$acl = Get-Acl "AD:\DC=example,DC=com"
$rule = New-Object System.DirectoryServices.ActiveDirectoryAccessRule(
    $user.SID,
    "GenericRead",
    "Allow"
)
$acl.AddAccessRule($rule)
Set-Acl -Path "AD:\DC=example,DC=com" -AclObject $acl
```

#### Password Reset Account (Optional)

**Required permissions**:

- All permissions of read-only account
- Write access to `userPassword` (OpenLDAP) or `unicodePwd` (AD)
- Reset password permission

**OpenLDAP ACL**:

```ldif
access to attrs=userPassword
    by dn="cn=password-reset,dc=example,dc=com" write
    by * none
```

**Active Directory**:

```powershell
# Grant "Reset Password" permission
$user = Get-ADUser -Identity "password-reset"
$acl = Get-Acl "AD:\DC=example,DC=com"
$rule = New-Object System.DirectoryServices.ActiveDirectoryAccessRule(
    $user.SID,
    "ExtendedRight",
    "Allow",
    [GUID]"00299570-246d-11d0-a768-00aa006e0529"  # Reset Password GUID
)
$acl.AddAccessRule($rule)
Set-Acl -Path "AD:\DC=example,DC=com" -AclObject $acl
```

### Testing LDAP Connection

```bash
# Test LDAPS connection
openssl s_client -connect ldap.example.com:636 -showcerts

# Test authentication
ldapsearch -x -H ldaps://ldap.example.com:636 \
  -D "cn=readonly,dc=example,dc=com" \
  -w "password" \
  -b "dc=example,dc=com" \
  "(mail=user@example.com)"

# Test password change (OpenLDAP)
ldappasswd -x -H ldaps://ldap.example.com:636 \
  -D "cn=password-reset,dc=example,dc=com" \
  -w "reset-password" \
  -s "newpassword" \
  "uid=testuser,ou=users,dc=example,dc=com"
```

---

## 📧 SMTP Setup

### Testing SMTP Connection

```bash
# Test SMTP with openssl
openssl s_client -connect smtp.example.com:587 -starttls smtp

# Send test email
swaks --to user@example.com \
  --from noreply@example.com \
  --server smtp.example.com:587 \
  --auth LOGIN \
  --auth-user noreply@example.com \
  --auth-password smtp-secret \
  --tls
```

### Common SMTP Configurations

#### Gmail

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=app-specific-password
SMTP_FROM_ADDRESS=your-email@gmail.com
```

**Note**: Requires [App Password](https://support.google.com/accounts/answer/185833) if 2FA enabled.

#### Office 365

```bash
SMTP_HOST=smtp.office365.com
SMTP_PORT=587
SMTP_USERNAME=your-email@company.com
SMTP_PASSWORD=your-password
SMTP_FROM_ADDRESS=your-email@company.com
```

#### SendGrid

```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USERNAME=apikey
SMTP_PASSWORD=your-sendgrid-api-key
SMTP_FROM_ADDRESS=noreply@example.com
```

#### Amazon SES

```bash
SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_USERNAME=your-smtp-username
SMTP_PASSWORD=your-smtp-password
SMTP_FROM_ADDRESS=noreply@example.com
```

**Note**: Sender email must be verified in SES.

---

## 📊 Monitoring and Logging

### Health Checks

```bash
# HTTP health check endpoint
curl http://localhost:3000/

# Expected response: 200 OK with HTML content
```

### Application Logs

The application logs to stdout/stderr. Configure log aggregation based on deployment:

#### Docker

```bash
# View logs
docker logs -f ldap-passwd

# Export to file
docker logs ldap-passwd > /var/log/ldap-passwd.log 2>&1
```

#### Systemd (Bare Metal)

```ini
# /etc/systemd/system/ldap-passwd.service
[Unit]
Description=LDAP Selfservice Password Changer
After=network.target

[Service]
Type=simple
User=ldap-passwd
Group=ldap-passwd
WorkingDirectory=/opt/ldap-passwd
EnvironmentFile=/opt/ldap-passwd/.env
ExecStart=/opt/ldap-passwd/ldap-passwd
Restart=on-failure
RestartSec=5s

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=ldap-passwd

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/ldap-passwd

[Install]
WantedBy=multi-user.target
```

View logs:

```bash
journalctl -u ldap-passwd -f
```

#### Kubernetes

```bash
# View pod logs
kubectl logs -n identity deployment/ldap-passwd -f

# Export to logging system (Fluentd/Loki)
# Configure via cluster logging infrastructure
```

### Metrics

Application exposes standard HTTP metrics via reverse proxy logs:

- Request count
- Response times
- Status codes
- Client IPs

**Prometheus + Grafana** (via nginx exporter):

```yaml
# docker-compose.monitoring.yml
services:
  nginx-exporter:
    image: nginx/nginx-prometheus-exporter:latest
    command:
      - "-nginx.scrape-uri=http://nginx:8080/stub_status"
    ports:
      - "9113:9113"
```

### Alerting

**Common alerts to configure**:

- High error rate (5xx responses)
- LDAP connection failures
- SMTP delivery failures
- Rate limit threshold reached
- Certificate expiration (30 days)

---

## 🔄 Updates and Maintenance

### Updating Docker Image

```bash
# Pull latest version
docker pull ghcr.io/netresearch/ldap-selfservice-password-changer:latest

# Restart container
docker compose down
docker compose up -d

# Or rolling update (Kubernetes)
kubectl rollout restart deployment/ldap-passwd -n identity
```

### Database Migrations

**Not applicable** - Application is stateless. Token storage is in-memory only.

### Backup and Disaster Recovery

**No persistent data** - Application state is fully ephemeral:

- Password reset tokens stored in memory (expire automatically)
- Configuration via environment variables
- No database or filesystem persistence

**Recovery steps**:

1. Deploy new instance with same configuration
2. Update DNS if needed
3. Application is immediately ready

### Scaling

**Horizontal scaling** - Application is fully stateless:

```yaml
# Docker Swarm
services:
  ldap-passwd:
    image: ghcr.io/netresearch/ldap-selfservice-password-changer:latest
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
```

```yaml
# Kubernetes
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
```

**Load balancing**: Use reverse proxy or service mesh for distribution.

**Session affinity**: Not required - no session state.

---

## 🔒 Security Hardening

### Network Security

- **LDAPS only**: Never use unencrypted LDAP in production
- **HTTPS only**: Enforce TLS for all web traffic
- **Firewall rules**: Allow only necessary ports (443 inbound, 636/587 outbound)
- **Internal network**: Place LDAP communication on private network if possible

### Access Control

- **IP whitelisting**: Restrict access via firewall or reverse proxy
- **VPN/Zero Trust**: Require VPN or Zero Trust Network Access
- **Rate limiting**: Configure reverse proxy rate limits (e.g., 10 req/s per IP)

### Secrets Management

- **Never commit secrets**: Use environment files or secret managers
- **Rotate credentials**: Regular rotation of LDAP/SMTP passwords
- **Principle of least privilege**: Service accounts with minimal permissions
- **Secret scanning**: Use tools like `git-secrets` or `truffleHog`

### Container Security

```yaml
# Recommended security context (Kubernetes)
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  seccompProfile:
    type: RuntimeDefault
```

### Security Headers

Already configured in reverse proxy examples:

- `Strict-Transport-Security` - Force HTTPS
- `X-Frame-Options: DENY` - Prevent clickjacking
- `X-Content-Type-Options: nosniff` - Prevent MIME sniffing
- `X-XSS-Protection` - XSS filter
- `Referrer-Policy` - Control referrer information

---

## 🐛 Troubleshooting

### Application Won't Start

**Check environment variables**:

```bash
docker logs ldap-passwd 2>&1 | grep -i "error\|fatal\|missing"
```

**Common issues**:

- Missing required env vars (LDAP_SERVER, SMTP_HOST, etc.)
- Invalid LDAP DN format
- Port already in use

### LDAP Connection Errors

**Test LDAP connectivity**:

```bash
# From container
docker exec -it ldap-passwd sh
apk add openldap-clients
ldapsearch -x -H $LDAP_SERVER -D "$LDAP_READONLY_USER" -w "$LDAP_READONLY_PASSWORD" -b "$LDAP_BASE_DN" "(mail=test@example.com)"
```

**Common issues**:

- Certificate validation failure (self-signed certs)
- Firewall blocking port 636
- Incorrect service account credentials
- Wrong Base DN

**Fix certificate issues**:

```bash
# Add custom CA certificate
docker run -v /path/to/ca-cert.crt:/etc/ssl/certs/custom-ca.crt:ro ...
```

### SMTP Delivery Failures

**Test SMTP from container**:

```bash
docker exec -it ldap-passwd sh
apk add curl
curl -v --url "smtp://$SMTP_HOST:$SMTP_PORT" \
  --mail-from "$SMTP_FROM_ADDRESS" \
  --mail-rcpt "test@example.com" \
  --user "$SMTP_USERNAME:$SMTP_PASSWORD"
```

**Common issues**:

- SMTP authentication failure
- Sender address not verified (SES, SendGrid)
- Port 587/25 blocked by firewall
- SPF/DKIM/DMARC configuration

### Rate Limiting Issues

**Reset rate limit** (requires application restart):

```bash
docker restart ldap-passwd
```

**Adjust rate limits**:

```bash
RESET_RATE_LIMIT_REQUESTS=10
RESET_RATE_LIMIT_WINDOW_MINUTES=60
```

### Password Policy Errors

**Verify policy settings**:

```bash
docker exec ldap-passwd env | grep MIN_
```

**Client-side validation** matches server-side - check browser console for details.

---

## 📚 Additional Resources

- **GitHub Repository**: https://github.com/netresearch/ldap-selfservice-password-changer
- **Docker Image**: https://github.com/netresearch/ldap-selfservice-password-changer/pkgs/container/ldap-selfservice-password-changer
- **Development Guide**: [docs/development-guide.md](development-guide.md)
- **Security Documentation**: [docs/security.md](security.md)
- **API Reference**: [docs/api-reference.md](api-reference.md)

---

**Last Updated**: 2025-10-08
**Maintained By**: Development Team
