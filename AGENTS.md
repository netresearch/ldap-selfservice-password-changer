# AGENTS.md

<!-- Managed by agent: keep sections & order; edit content, not structure. Last updated: 2026-04-18 -->

**Precedence:** the **closest `AGENTS.md`** to the files you're changing wins. This root holds global defaults only; scoped files override.

## Index of scoped AGENTS.md

| Path                                             | Scope                                               |
| ------------------------------------------------ | --------------------------------------------------- |
| [internal/AGENTS.md](./internal/AGENTS.md)       | Go backend packages — LDAP, rate limit, RPC, tokens |
| [internal/web/AGENTS.md](./internal/web/AGENTS.md) | Frontend — TypeScript, Tailwind CSS, WCAG 2.2 AAA |

## Project

LDAP self-service password changer (hybrid Go + TypeScript web app). Email-based reset, rate limiting, WCAG 2.2 AAA compliance. Single binary deployment with embedded assets.

**Stack**: Go 1.26 + Fiber v3, TypeScript (ultra-strict), Tailwind CSS 4, Docker multi-stage, **Bun** (migrated from pnpm in [`8f69f47`](https://github.com/netresearch/ldap-selfservice-password-changer/commit/8f69f47)).

## Commands

Source: `package.json` scripts + `go test`. All commands runnable from repo root.

| Task            | Command                                       |
| --------------- | --------------------------------------------- |
| Install deps    | `bun install --frozen-lockfile`               |
| Dev (all watch) | `bun run dev`                                 |
| Build all       | `bun run build`                               |
| Build assets    | `bun run build:assets`                        |
| TS watch        | `bun run js:dev`                              |
| TS build/check  | `bun run js:build`                            |
| CSS watch       | `bun run css:dev`                             |
| CSS build       | `bun run css:build`                           |
| Go test         | `go test -v -race ./...`                      |
| Go build        | `go build -v ./...`                           |
| Format          | `bunx prettier --write .`                     |
| Format check    | `bunx prettier --check .`                     |
| Lint TS         | `bun run lint` (or `bun run lint:fix`)        |
| Lint Go         | `golangci-lint run` (CI: `golangci/golangci-lint-action`) |

**Docker-first**: `docker compose --profile dev up` is the canonical dev path; native Bun/Go is optional convenience.

## Workflow

1. Before coding, read the nearest `AGENTS.md` for the area you're touching.
2. After a change: smallest relevant check (`bun run js:build` for TS, `go test ./pkg/...` for Go).
3. Before committing: `bunx prettier --write .` + full `go test ./...` if ≥2 files or shared code.
4. Before claiming done: run verification, **show output as evidence** — never "try again" or "should work now" without proof.

## Security (global)

- **No secrets in git.** Use `.env.local` (gitignored). Never commit LDAP/SMTP credentials.
- **LDAPS required** in production (`ldaps://` URLs).
- **No PII logging** — passwords, tokens, session IDs never reach logs.
- **Rate limiting** 3 req/hour/IP (configurable via `RATE_LIMIT_*`).
- **Cryptographic random tokens** with configurable expiry; single-use, server-side.
- **Strict input validation** at boundaries — see `internal/validators/`.
- **Container runs as UID 65534** (nobody), not root.
- **Renovate** handles dependency updates; review major-version changelogs.

## PR/Commit Checklist

Before commit:

- [ ] `bunx prettier --write .`
- [ ] `bun run js:build` (TypeScript strict)
- [ ] `go test ./...`
- [ ] `go build`
- [ ] No secrets in staged files
- [ ] Docs updated if behavior changed
- [ ] WCAG 2.2 AAA maintained (if UI changed — see [docs/accessibility.md](docs/accessibility.md))

Commit format: [Conventional Commits](https://www.conventionalcommits.org/). Examples: `feat(auth): add reset via email`, `fix(validators): correct regex`, `chore(deps): bump bun`. **No AI attribution** in messages.

PR:

- [ ] CI green (types, formatting, tests, security scans)
- [ ] Keep small (~≤300 net LOC when possible)
- [ ] Prefix with ticket ID when applicable
- [ ] Updated docs land in same PR

## House Rules

- **Docker-first.** Native setup is convenience, not requirement. Use compose profiles: `dev`, `test`.
- **YAGNI.** Build only what's requested. No speculative features.
- **Type safety.** TS: no `any`, all strict flags on. Go: `any` (not `interface{}`), `errors.AsType[T]`, wrap errors with context.
- **Accessibility non-negotiable.** 7:1 contrast, full keyboard nav, screen-reader tested. See [docs/accessibility.md](docs/accessibility.md).
- **Dependencies.** Keep `bun.lock` and `go.sum` committed. Use top-level `overrides` in `package.json` for transitive CVE fixes when upstream hasn't patched.
- **CI/CD.** `pr-quality.yml` auto-approves collaborator PRs. `auto-merge-deps.yml` handles Dependabot/Renovate. See [docs/development-guide.md](docs/development-guide.md) for bootstrap.

## Releases

Unified with the org's release pipeline ([`netresearch/.github`](https://github.com/netresearch/.github) reusables).

```bash
git tag -s vX.Y.Z -m "vX.Y.Z"    # annotated + signed tag (required)
git push origin vX.Y.Z            # triggers release.yml
```

Pipeline (see [.github/workflows/release.yml](.github/workflows/release.yml)):

1. **`create-release`** — creates GitHub Release; computes `make_latest` from semver vs existing releases (backfills and older-major bugfixes don't steal the Latest badge).
2. **`goreleaser`** — builds binaries, archives, per-archive Syft SBOMs, cosign-signed `checksums.txt`; attests archives+SBOMs via `actions/attest-build-provenance`.
3. **`container`** — multi-arch ghcr.io image, cosign keyless-signed + SLSA provenance.
4. **`verify-notes`** — appends standardized "Verify your download" block.

**Backfill**: `gh workflow run release.yml --ref main -f tag=vX.Y.Z`.

**GoReleaser config notes** ([`.goreleaser.yml`](.goreleaser.yml)):

- `mode: keep-existing` (preserves create-release's notes; don't change to `replace`)
- `changelog.use: git` (not `github` — `github` ignores filters)
- No 32-bit ARM: Fiber v3 `math.MaxUint32` overflows `int` on 32-bit → `linux/arm` excluded

## When Stuck

1. Scoped `AGENTS.md` for the area → code in `internal/` → [docs/](docs/).
2. Similar patterns: search git history / existing tests.
3. `go test -v ./...` surfaces many problems.
4. Docker weirdness: `docker compose down -v && docker compose --profile dev up --build`.
5. Env config: `.env.local` vs `.env.local.example`.
