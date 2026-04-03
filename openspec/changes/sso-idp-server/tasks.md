## 1. Repository & Project Bootstrap

- [x] 1.1 Create GitHub repository `braza-sso` with description and topics
- [x] 1.2 Initialize Go module (`go mod init github.com/<org>/braza-sso`)
- [x] 1.3 Create Vite + React + TypeScript frontend scaffold (`frontend/`)
- [x] 1.4 Write `.gitignore` covering Go binaries, `.env`, `node_modules`, IDE files, Docker volumes
- [x] 1.5 Write `README.md` with project overview, prerequisites, local setup steps, and env var documentation
- [x] 1.6 Create `.env.example` documenting all required environment variables
- [x] 1.7 Make initial atomic commit: "chore: bootstrap project structure"

## 2. Infrastructure & Docker Compose

- [x] 2.1 Write `Dockerfile` for Go API (multi-stage: builder + distroless/alpine runtime)
- [x] 2.2 Write `Dockerfile` for React frontend (builder + Nginx static server)
- [x] 2.3 Write `docker-compose.yml` with services: `api`, `frontend`, `cassandra`, `redis`
- [x] 2.4 Add health checks for Cassandra (`nodetool status`) and Redis (`redis-cli ping`)
- [x] 2.5 Configure `depends_on` with health check conditions so API waits for Cassandra and Redis
- [x] 2.6 Add Caddy or Nginx reverse proxy service for TLS termination (with Let's Encrypt config)
- [x] 2.7 Write Cassandra keyspace + table migration script (`scripts/migrate.sh`)
- [x] 2.8 Verify `docker compose up -d` runs locally with all services healthy

## 3. Go API: Project Structure & Config

- [x] 3.1 Define package layout: `cmd/api/`, `internal/auth/`, `internal/user/`, `internal/oauth/`, `internal/middleware/`, `internal/db/`, `internal/cache/`, `internal/email/`
- [x] 3.2 Implement config loader reading all required env vars at startup with clear fatal errors for missing values
- [x] 3.3 Add `chi` (or `gin`) router with structured logging middleware (`zerolog` or `slog`)
- [x] 3.4 Implement graceful shutdown (SIGTERM handling, connection draining)
- [x] 3.5 Add `gocql` Cassandra driver with connection pool and retry on startup

## 4. Cassandra Schema

- [x] 4.1 Create keyspace `braza_sso` with `SimpleStrategy`, replication factor 1
- [x] 4.2 Create `users` table: `user_id (UUID PK)`, `email`, `password_hash`, `email_verified`, `locked_until`, `failed_attempts`, `created_at`, `updated_at`
- [x] 4.3 Create `email_index` materialized view or secondary index for email lookups
- [x] 4.4 Create `oauth_clients` table: `client_id`, `client_secret_hash`, `redirect_uris`, `scopes`, `name`, `logo_url`
- [x] 4.5 Create `user_consents` table: `user_id`, `client_id`, `scopes`, `granted_at`
- [x] 4.6 Create `federated_identities` table: `user_id`, `provider`, `provider_user_id`, `email`

## 5. Password Hashing & Security Primitives

- [x] 5.1 Implement Argon2id hash function with parameters: memory=65536, time=3, threads=4, pepper from env
- [x] 5.2 Implement constant-time hash verification to prevent timing attacks
- [x] 5.3 Implement password policy validator (min 12 chars, upper, lower, digit, special)
- [x] 5.4 Generate RSA-4096 key pair for JWT RS256 signing; load private key from file path env var
- [x] 5.5 Implement JWKS endpoint (`/oauth/jwks.json`) returning public key in JWK format

## 6. User Registration

- [x] 6.1 Implement `POST /auth/register` endpoint: validate input, check duplicate email, hash password, store user in Cassandra (pending state)
- [x] 6.2 Implement email verification token generation (crypto/rand, 32 bytes, base64url, stored in Redis with 24h TTL)
- [x] 6.3 Implement `GET /auth/verify-email?token=<token>` endpoint: validate token, activate account
- [x] 6.4 Implement `POST /auth/resend-verification` endpoint (rate-limited)
- [x] 6.5 Write unit tests for registration and verification flows

## 7. User Authentication & Session Management

- [x] 7.1 Implement `POST /auth/login` endpoint: look up user by email, verify Argon2id hash + pepper
- [x] 7.2 Implement failed attempt tracking in Redis with 15-minute sliding window
- [x] 7.3 Implement account lockout after 5 failed attempts (30-minute lockout, send unlock email)
- [x] 7.4 Implement JWT RS256 access token issuance (15-minute TTL, claims: sub, iss, aud, exp, iat, jti, email, email_verified)
- [x] 7.5 Implement opaque refresh token generation, storage in Redis (7-day TTL), set as HttpOnly Secure SameSite=Strict cookie
- [x] 7.6 Implement `POST /auth/token/refresh` with refresh token rotation (invalidate old, issue new)
- [x] 7.7 Detect refresh token reuse: invalidate all sessions for user, return 401
- [x] 7.8 Write unit tests for login, lockout, and token refresh flows

## 8. Password Management

- [ ] 8.1 Implement `POST /auth/password/reset-request`: generate reset token (Redis, 1h TTL), send email (always return 200)
- [ ] 8.2 Implement `POST /auth/password/reset`: validate token, update password hash, invalidate all Redis sessions
- [ ] 8.3 Write unit tests for password reset flow

## 9. Two-Factor Authentication (TOTP)

- [ ] 9.1 Add `pquerna/otp` dependency
- [ ] 9.2 Implement `POST /account/2fa/enroll`: generate TOTP secret, return QR code URI + 8 recovery codes
- [ ] 9.3 Store recovery codes hashed with Argon2id in Cassandra (`user_recovery_codes` table)
- [ ] 9.4 Implement `POST /account/2fa/confirm`: verify submitted TOTP code, activate 2FA
- [ ] 9.5 Implement intermediate session token (Redis, 5-minute TTL) issued after credential validation when 2FA is enabled
- [ ] 9.6 Implement `POST /auth/2fa/verify` endpoint to complete login with TOTP code
- [ ] 9.7 Implement `POST /auth/2fa/recovery` endpoint for recovery code usage (invalidate used code, prompt re-enrollment)
- [ ] 9.8 Write unit tests for TOTP enrollment and verification

## 10. Rate Limiting

- [ ] 10.1 Implement Redis sliding window rate limiter middleware (INCR + EXPIRE)
- [ ] 10.2 Apply per-IP limit (20 req/min) to `/auth/login`
- [ ] 10.3 Apply per-account limit (10 req/5min) to `/auth/login`
- [ ] 10.4 Apply per-IP limit (5 req/10min) to `/auth/register`
- [ ] 10.5 Apply per-email silent drop (3 req/15min) to `/auth/password/reset-request`
- [ ] 10.6 Return HTTP 429 with `Retry-After` header for blocked requests

## 11. OAuth2 / OIDC

- [ ] 11.1 Implement `GET /oauth/authorize` endpoint: validate client_id, redirect_uri, response_type, scope, state
- [ ] 11.2 Implement authorization code generation (crypto/rand, stored in Redis with 60-second TTL)
- [ ] 11.3 Implement consent screen logic: check existing consent in Cassandra, skip or show UI
- [ ] 11.4 Implement `POST /oauth/token` endpoint: authorization code exchange, return access token + refresh token + ID token
- [ ] 11.5 Implement `GET /.well-known/openid-configuration` OIDC discovery endpoint
- [ ] 11.6 Implement `GET /oauth/userinfo` endpoint returning OIDC claims from JWT
- [ ] 11.7 Implement client credentials grant for machine-to-machine auth
- [ ] 11.8 Write integration tests for full authorization code flow

## 12. Single Logout (SLO)

- [ ] 12.1 Implement `POST /auth/logout` endpoint: delete refresh token from Redis, clear cookie
- [ ] 12.2 Implement back-channel logout: send signed logout token (JWT) to each SP's logout URI
- [ ] 12.3 Make logout idempotent (return 200 even if session not found)

## 13. External IdP Federation

- [ ] 13.1 Implement Google OAuth2 federation: `/auth/federation/google` redirect and `/auth/federation/google/callback` handler
- [ ] 13.2 Implement state parameter validation (CSRF protection) for federation flows
- [ ] 13.3 Implement account linking: match federated email to existing account or auto-create
- [ ] 13.4 Store federated identity in `federated_identities` Cassandra table
- [ ] 13.5 Implement Keycloak OIDC federation (optional v1): `/auth/federation/keycloak` flow

## 14. Email Service

- [ ] 14.1 Implement email client abstraction (interface) with SMTP implementation
- [ ] 14.2 Configure SMTP relay (SendGrid/Mailgun) via env vars
- [ ] 14.3 Create HTML email templates: verification, password reset, account lockout, unlock
- [ ] 14.4 Write unit tests with mock email client

## 15. React Frontend

- [ ] 15.1 Implement Login page with email/password form and "Continue with Google" button
- [ ] 15.2 Implement Registration page with password strength indicator
- [ ] 15.3 Implement Email Verification page (token from URL param)
- [ ] 15.4 Implement Password Reset request and confirmation pages
- [ ] 15.5 Implement 2FA enrollment page (QR code display + recovery codes)
- [ ] 15.6 Implement 2FA verification page (TOTP input during login)
- [ ] 15.7 Implement OAuth2 Consent screen (client info + requested scopes)
- [ ] 15.8 Implement Account Locked page with resend-unlock option
- [ ] 15.9 Configure Axios/fetch interceptor for automatic token refresh
- [ ] 15.10 Configure Nginx in frontend container to proxy `/api` to Go API

## 16. CI/CD Pipeline

- [ ] 16.1 Write `.github/workflows/ci.yml`: lint (golangci-lint + eslint), test, build Docker images, push to GHCR
- [ ] 16.2 Write `.github/workflows/deploy.yml`: trigger on CI success on `main`, SSH to VPS, `docker compose pull && up -d`
- [ ] 16.3 Configure GitHub repository secrets: `VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, `GHCR_TOKEN`, all app env vars
- [ ] 16.4 Add branch protection rule on `main` requiring CI status checks to pass
- [ ] 16.5 Test full pipeline end-to-end: push to main → CI → deploy

## 17. VPS Deployment

- [ ] 17.1 Provision Hostinger VPS: install Docker, Docker Compose, Git, configure firewall (ports 80, 443, 22)
- [ ] 17.2 Set up DNS A record pointing domain to VPS IP
- [ ] 17.3 Configure Caddy or Certbot for TLS certificate (Let's Encrypt)
- [ ] 17.4 Clone repo to VPS, create `.env` with production values
- [ ] 17.5 Run `docker compose up -d`, execute Cassandra migration script
- [ ] 17.6 Smoke test: register, verify email, login, obtain token, 2FA enrollment, logout

## 18. Notion Project Tasks

- [x] 18.1 Create SSO project in Notion teamspace with all phases as tasks
- [x] 18.2 Save proposal, design, and implementation plans to corresponding Notion tasks
- [ ] 18.3 Track progress per phase in Notion as implementation proceeds
