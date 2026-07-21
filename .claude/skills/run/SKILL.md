---
name: run
description: Launch and drive GopherPass locally for review/testing — build the embedded binary, bring up the OpenLDAP + Mailpit dev stack, and exercise the password change/reset flows against seeded users.
---

# Running GopherPass locally

The production `Dockerfile` is a **binary-selector**: it `COPY`s pre-built
binaries from `bin/` (produced by the release CI's cross-compile matrix) and
does **no** `go build` / `bun install`. So `docker compose --build` fails from a
clean checkout with `COPY bin/… lstat /bin: no such file or directory` — there is
nothing in `bin/` yet.

Two ways to get a running instance. Prefer **A** (everything on the Compose
network, so the app reaches Mailpit's internal SMTP).

## A. Build the binary, then Compose (recommended)

The frontend assets are embedded via `go:embed`, so build assets _before_ the Go
binary. Cross-compile into the exact name the Dockerfile expects
(`bin/<repo>-linux-<arch>`):

```bash
ARCH=$(go env GOARCH)                       # amd64 on WSL2/x86_64
bun install --frozen-lockfile
bun run build:assets                        # writes internal/web/static/{styles.css,js/*.js}
CGO_ENABLED=0 GOOS=linux GOARCH=$ARCH go build -trimpath \
  -ldflags="-w -s -X main.version=vX.Y.Z-rc -X main.build=$(git rev-parse --short HEAD)" \
  -o bin/ldap-selfservice-password-changer-linux-$ARCH .

# Pick non-default host ports to avoid collisions; the app container listens on 3000.
APP_PORT=3140 MAILPIT_WEB_PORT=8125 docker compose --profile dev up --build -d
```

The `dev` profile brings up `openldap` → `openldap-init` (seeds users, exits 0) →
`mailpit` → `app`.

- App: `http://localhost:3140` (change password) and `/forgot-password`
- Mailpit (catches reset emails): `http://localhost:8125`
- Health: `curl -s -o /dev/null -w '%{http_code}' http://localhost:3140/health/live` → `200`

**Fixed container names** (`gopherpass-openldap` etc.) are NOT project-scoped, so a
stale container from an earlier run collides:
`docker rm -f gopherpass-openldap gopherpass-mailpit gopherpass-app` then retry.

## B. Native binary against Compose infra

Only if you don't want to rebuild the image. Note Mailpit's SMTP (1025) is
**internal-only** (not host-mapped), so a native app cannot send reset mail —
use A if you need the reset email. LDAP (389) and the Mailpit web UI (8125) are
host-mapped.

```bash
docker compose up -d openldap openldap-init mailpit
go run . -ldap-server ldap://127.0.0.1:389 -base-dn dc=netresearch,dc=local \
  -readonly-user cn=admin,dc=netresearch,dc=local -readonly-password admin -port 39443
```

## Seeded users (dev/seed.ldif)

Password policy: ≥10 chars, 1 number, 1 symbol, 1 upper, 1 lower.

| uid                             | mail                         | password         |
| ------------------------------- | ---------------------------- | ---------------- |
| `jdoe`                          | john.doe@netresearch.local   | `password`       |
| `jsmith`                        | jane.smith@netresearch.local | `password`       |
| `password-reset` (service acct) | —                            | `reset-password` |

## Drive it — reset by username (proves RESET_IDENTIFIER_MODE)

Set `RESET_IDENTIFIER_MODE=both` (via `.env.local` in the worktree — Compose loads
it after the inline env) and recreate `app`. Then submit a **username** and confirm
Mailpit received the mail addressed to the account's **registered** address, not
the typed identifier:

```bash
printf 'RESET_IDENTIFIER_MODE=both\n' > .env.local
APP_PORT=3140 MAILPIT_WEB_PORT=8125 docker compose --profile dev up -d --force-recreate app
curl -s -X DELETE http://localhost:8125/api/v1/messages
curl -s -X POST http://localhost:3140/api/rpc -H 'Content-Type: application/json' \
  -d '{"method":"request-password-reset","params":["jdoe"]}'
curl -s http://localhost:8125/api/v1/messages | \
  python3 -c "import json,sys;[print(m['To'][0]['Address'],'|',m['Subject']) for m in json.load(sys.stdin)['messages']]"
# → john.doe@netresearch.local | Password Reset Request   (NOT "jdoe")
```

## Teardown

```bash
docker compose --profile dev down -v      # or: docker rm -f gopherpass-{app,mailpit,openldap}
```
