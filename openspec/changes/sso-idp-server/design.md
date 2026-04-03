## Context

Braza SSO is a greenfield Identity Provider built to provide centralized authentication across internal and external services. There are no legacy systems to migrate from. The system must be self-hosted on a Hostinger VPS, support federation with external providers (Google, Keycloak), and expose an OAuth2/OIDC-compliant API. All infrastructure is provisioned via Docker Compose, and delivery is automated through GitHub Actions CI/CD.

## Goals / Non-Goals

**Goals:**
- Implement a fully functional OAuth2/OIDC-compliant Identity Provider in Go
- Provide a React SPA for login, registration, consent, 2FA, and account management
- Store user credentials in Cassandra with Argon2id hashing (memory: 64MB, time: 3, threads: 4, pepper from env)
- Manage sessions and token caching in Redis
- Support TOTP-based 2FA, email verification, password reset, and account lockout
- Rate-limit sensitive endpoints per IP and per user
- Enable Single Logout (SLO) across service providers
- Federate with upstream IdPs (Google OAuth2, Keycloak OIDC)
- Deploy to VPS via GitHub Actions with Docker Compose
- Ship with README, `.gitignore`, and atomic git commits

**Non-Goals:**
- SAML support (out of scope for v1)
- Native mobile SDK
- Multi-tenancy / organization isolation in v1
- Billing or subscription management

## Decisions

### 1. Language: Go for backend

**Decision:** Go (`net/http` + `chi` router or `gin`).
**Rationale:** Excellent concurrency model for auth workloads, strong standard crypto library, small binary, easy Dockerization. Avoids Node.js runtime overhead for a security-critical service.
**Alternatives considered:** Node.js/Express — familiar but weaker typing and larger attack surface; Python FastAPI — slower cold start, GIL concerns.

### 2. Database: Apache Cassandra

**Decision:** Cassandra for user credentials and session-adjacent data.
**Rationale:** Wide-column store scales horizontally, tunable consistency; user credential lookups are single-partition reads (by user ID or email index). Fits requirement.
**Alternatives considered:** PostgreSQL — simpler operationally, but single-node unless PgBouncer+replicas; MongoDB — document model less suited to tabular auth data.
**Note:** Use `gocql` driver. Design keyspace with replication factor 1 for single-node VPS, upgradeable later.

### 3. Password Hashing: Argon2id

**Decision:** `golang.org/x/crypto/argon2` with parameters `memory=65536 (64MB)`, `time=3`, `threads=4`, and a server-side pepper stored as env var.
**Rationale:** Argon2id is memory-hard, resistant to GPU cracking, and recommended by OWASP. Pepper adds a second factor attackers need even if DB is compromised.
**Alternatives considered:** bcrypt — CPU-only, less resistant to GPU attacks; scrypt — no parallelism parameter.

### 4. Token Strategy: RS256 JWTs

**Decision:** RS256 (asymmetric) for access tokens; opaque refresh tokens stored in Redis.
**Rationale:** Service providers can verify access tokens without contacting SSO server (public key). Refresh tokens are short-lived, Redis-backed, and rotated on use. This limits blast radius of token theft.
**Alternatives considered:** HS256 — requires sharing secret with every SP; opaque access tokens — require introspection endpoint on every request.

### 5. Session Management: Redis

**Decision:** Redis for session store, refresh token store, rate limit counters, and TOTP enrollment state.
**Rationale:** Sub-millisecond reads, TTL-native, atomic operations for rate limiting (`INCR`/`EXPIRE`). Aligns with stated requirement.

### 6. Frontend: React SPA

**Decision:** React (Vite + TypeScript) served as static files via Nginx, communicating with Go API over HTTPS.
**Rationale:** Meets requirement; Vite gives fast DX; TypeScript reduces auth-flow bugs. SPA is served from same domain to avoid cross-origin cookie issues with HttpOnly JWT cookies.

### 7. Infrastructure: Docker Compose

**Decision:** Single `docker-compose.yml` with services: `api` (Go), `frontend` (Nginx + React build), `cassandra`, `redis`.
**Rationale:** Reproducible local dev and production. VPS runs Compose directly. GitHub Actions builds and pushes images, then SSH-deploys via `docker compose pull && docker compose up -d`.

### 8. OAuth2/OIDC: Custom implementation over library

**Decision:** Implement OAuth2 authorization code flow manually with `golang-jwt/jwt` and standard Go crypto. Use `zitadel/oidc` library for OIDC discovery/token introspection helpers if needed.
**Rationale:** Full visibility into security-critical flows. Libraries like `ory/fosite` add complexity; custom implementation is more auditable for this scope.
**Alternatives considered:** `ory/hydra` — full-featured but heavyweight, black-box for learning; `dex` — federation-ready but less customizable.

### 9. Rate Limiting

**Decision:** Redis-backed sliding window rate limiter on `/login`, `/register`, `/reset-password`, `/token` endpoints. Limits per IP and per account.
**Rationale:** Prevents brute-force and credential stuffing. Redis `INCR`+`EXPIRE` pattern is atomic and horizontally scalable.

### 10. CI/CD: GitHub Actions

**Decision:** GitHub Actions workflow: lint → test → build Docker images → push to GHCR → SSH deploy to Hostinger VPS.
**Rationale:** Native to GitHub, free for public repos, supports secrets management. VPS deploy via `appleboy/ssh-action`.

## Risks / Trade-offs

- **Cassandra complexity on single node** → Use replication factor 1 initially; document upgrade path to 3-node cluster. Monitor with `nodetool status`.
- **JWT RS256 key rotation** → Generate RSA key pair at deploy time, store private key as Docker secret/env. Plan key rotation procedure in runbook before v1 launch.
- **Email deliverability** → Use SMTP relay (SendGrid/Mailgun) rather than raw SMTP from VPS IP (likely blocklisted). Configure SPF/DKIM.
- **TOTP recovery codes** → Must be hashed at rest (Argon2id). Risk of user lockout if codes lost → provide re-enrollment via email verification.
- **Cassandra cold start in Docker Compose** → API must implement retry/backoff on startup; use healthcheck in Compose.
- **Single VPS = SPOF** → Acceptable for v1; document HA upgrade path.

## Migration Plan

1. Provision VPS: install Docker + Docker Compose, open ports 80/443/8080
2. Set up DNS, obtain TLS cert (Let's Encrypt via Caddy or Certbot)
3. Create GitHub repo, push initial code with `.gitignore` and `README.md`
4. Configure GitHub secrets: `VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, `GHCR_TOKEN`, env vars for Cassandra/Redis/SMTP/pepper
5. First deploy: `docker compose up -d` on VPS
6. Run DB migrations (Cassandra keyspace + table creation) via migration job container
7. Smoke test: register user, verify email, login, obtain token, logout
8. Enable GitHub Actions CI/CD for subsequent deploys

**Rollback:** Keep previous Docker image tag; `docker compose up -d --no-build` with previous tag.

## Open Questions

- Which Nginx vs Caddy for reverse proxy/TLS termination? (Caddy preferred for auto-HTTPS simplicity)
- External IdP federation scope for v1: Google only, or also Keycloak?
- Email provider choice: SendGrid, Mailgun, or Resend?
- TOTP enforcement: optional or mandatory for all users?
