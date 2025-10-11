FROM node:24 AS frontend-builder
WORKDIR /build

# Disable Husky git hooks in Docker build (no .git directory in build context)
ENV HUSKY=0

# Use Corepack instead of npm global install for better performance
RUN corepack enable

# Copy dependency files first for better layer caching
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy only necessary files for frontend build
COPY postcss.config.js tailwind.config.js tsconfig.json ./
COPY scripts/ ./scripts/
COPY internal/web/ ./internal/web/

RUN pnpm build:assets

FROM golang:1.25-alpine AS backend-builder
WORKDIR /build

# Copy dependency files
COPY go.mod go.sum ./

# Copy only Go source files
COPY main.go ./
COPY internal/ ./internal/

# Copy frontend build artifacts
COPY --from=frontend-builder /build/internal/web/static/styles.css /build/internal/web/static/styles.css
COPY --from=frontend-builder /build/internal/web/static/js/*.js /build/internal/web/static/js/

# Download dependencies and build with size optimization flags
RUN go mod download && \
    CGO_ENABLED=0 go build -ldflags="-w -s" -o /build/ldap-passwd

FROM scratch AS runner

# Run as non-root user for defense-in-depth (nobody:nogroup = 65534:65534)
USER 65534:65534

# Copy CA certificates for LDAPS connections
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the static binary
COPY --from=backend-builder /build/ldap-passwd /ldap-passwd

ENTRYPOINT [ "/ldap-passwd" ]
