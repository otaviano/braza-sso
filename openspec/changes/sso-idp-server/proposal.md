## Why

Modern applications require secure, centralized identity management. Building a custom SSO Identity Provider (IdP) removes dependency on third-party auth vendors, provides full control over user data and security policies, and enables seamless authentication across internal and external services — while supporting federation with external providers (Google, Keycloak, etc.).

## What Changes

- New standalone SSO server (Go backend) acting as Identity Provider
- New React frontend for login, registration, consent, and account management UIs
- JWT-based token issuance (RS256) with refresh token rotation
- OAuth2/OIDC-compliant authorization flows (authorization code, implicit, client credentials)
- User credential storage in Apache Cassandra with Argon2id password hashing (memory: 64MB, time: 3, threads: 4) + pepper via env var
- Redis-based session management and caching
- Email verification, password reset, and 2FA (TOTP) flows
- Account lockout and rate limiting on sensitive endpoints
- Single Logout (SLO) across all connected service providers
- Docker Compose provisioning for all infrastructure (Go API, React, Cassandra, Redis)
- GitHub Actions CI/CD pipeline for automated build, test, and deploy to VPS

## Capabilities

### New Capabilities

- `user-registration`: Self-service account creation with email verification flow
- `user-authentication`: Credential validation, JWT issuance, session management via Redis
- `token-management`: Access/refresh token lifecycle, rotation, revocation, RS256 signing
- `oauth2-oidc`: OAuth2 authorization code flow, consent screen, OIDC discovery endpoint
- `password-management`: Password reset via email token, strong password policy enforcement
- `two-factor-auth`: TOTP-based 2FA enrollment, verification, and recovery codes
- `account-lockout`: Failed login tracking, lockout policy, unlock via email
- `rate-limiting`: Per-IP and per-user rate limiting on auth endpoints
- `single-logout`: Centralized SLO invalidating sessions across all service providers
- `external-idp-federation`: Federated login via Google, Keycloak (OAuth2/OIDC upstream)
- `infrastructure`: Docker Compose setup for Go API, React, Cassandra, Redis
- `ci-cd`: GitHub Actions pipeline for build, test, and deploy to Hostinger VPS

### Modified Capabilities

_(none — this is a greenfield project)_

## Impact

- **New services**: SSO Go API server, React SPA, Cassandra cluster, Redis instance
- **Infrastructure**: Docker Compose, Dockerfile for each service, `.env` configuration
- **Security dependencies**: Argon2id (`golang.org/x/crypto`), JWT RS256 (`golang-jwt/jwt`), TOTP (`pquerna/otp`)
- **External integrations**: SMTP for email, Google OAuth2 app, Keycloak realm (optional)
- **Deployment**: Hostinger VPS, GitHub Actions workflow, HTTPS via reverse proxy (Nginx/Caddy)
- **Repository**: New GitHub repo with `.gitignore`, `README.md`, atomic commit discipline
