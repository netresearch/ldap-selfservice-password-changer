# Binary-selector stage (production / CI) — release.yml's binaries matrix
# (build-go-attest.yml) cross-compiles Go with frontend assets embedded
# via go:embed, uploads to the release, and the container job downloads
# them back into bin/. This stage picks the right pre-built binary per
# TARGETARCH/TARGETVARIANT — no `go build` or `bun install` in Docker.
FROM alpine:3.23 AS binary-selector

ARG TARGETARCH
ARG TARGETVARIANT

# CA certificates for LDAPS connections (copied into the scratch
# runtime; alpine ships them already).
RUN apk add --no-cache ca-certificates

COPY bin/ldap-selfservice-password-changer-linux-* /tmp/

RUN set -eux; \
    case "${TARGETARCH}" in \
        arm)              BINARY="ldap-selfservice-password-changer-linux-arm${TARGETVARIANT}" ;; \
        386|amd64|arm64)  BINARY="ldap-selfservice-password-changer-linux-${TARGETARCH}" ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    cp "/tmp/${BINARY}" /usr/bin/ldap-selfservice-password-changer; \
    chmod +x /usr/bin/ldap-selfservice-password-changer

# Runtime stage — scratch: zero packages, zero shell, zero attack surface.
FROM scratch AS runner

LABEL org.opencontainers.image.title="LDAP Self-Service Password Changer" \
      org.opencontainers.image.source="https://github.com/netresearch/ldap-selfservice-password-changer" \
      org.opencontainers.image.vendor="Netresearch DTT GmbH" \
      org.opencontainers.image.licenses="MIT"

# Run as nobody:nogroup (65534:65534) for defense-in-depth.
USER 65534:65534

# LDAPS TLS validation needs the system CA bundle.
COPY --from=binary-selector /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=binary-selector \
     /usr/bin/ldap-selfservice-password-changer \
     /ldap-selfservice-password-changer

# Uses the binary's --health-check flag (works in scratch: no shell
# needed).
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/ldap-selfservice-password-changer", "--health-check"]

ENTRYPOINT ["/ldap-selfservice-password-changer"]
