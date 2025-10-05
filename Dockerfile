FROM --platform=amd64 node:24@sha256:4e87fa2c1aa4a31edfa4092cc50428e86bf129e5bb528e2b3bbc8661e2038339 AS frontend-builder
WORKDIR /build

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

FROM golang:1.25-alpine@sha256:b6ed3fd0452c0e9bcdef5597f29cc1418f61672e9d3a2f55bf02e7222c014abd AS backend-builder
WORKDIR /build

# Download dependencies first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy only Go source files
COPY main.go ./
COPY internal/ ./internal/

# Copy frontend build artifacts
COPY --from=frontend-builder /build/internal/web/static/styles.css /build/internal/web/static/styles.css
COPY --from=frontend-builder /build/internal/web/static/js/*.js /build/internal/web/static/js/

# Build with size optimization flags (-w removes DWARF debug info, -s removes symbol table)
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /build/ldap-passwd

FROM scratch AS runner

# Copy CA certificates for LDAPS connections
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the static binary
COPY --from=backend-builder /build/ldap-passwd /ldap-passwd

ENTRYPOINT [ "/ldap-passwd" ]
