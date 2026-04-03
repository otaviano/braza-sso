## ADDED Requirements

### Requirement: RS256 JWT access token lifecycle
The system SHALL issue RS256-signed JWTs as access tokens with a configurable TTL (default 15 minutes). Tokens SHALL include standard OIDC claims: `sub`, `iss`, `aud`, `exp`, `iat`, `jti`, `email`, `email_verified`.

#### Scenario: Token issuance
- **WHEN** authentication succeeds
- **THEN** the system signs a JWT with the RSA private key and returns it in the response body

#### Scenario: Token verification by service provider
- **WHEN** a service provider fetches the JWKS endpoint and validates a token signature
- **THEN** the token is verifiable without contacting the SSO server

### Requirement: Opaque refresh token rotation
The system SHALL issue opaque refresh tokens stored in Redis. On each use, the token MUST be rotated (old invalidated, new issued). Reuse of invalidated tokens SHALL trigger full session revocation.

#### Scenario: Successful rotation
- **WHEN** a valid refresh token is presented
- **THEN** a new refresh token is issued, the old one is deleted from Redis, and the new one is set as a HttpOnly Secure SameSite=Strict cookie

### Requirement: Token revocation
The system SHALL provide a token revocation endpoint (`/auth/token/revoke`) that invalidates a refresh token and removes the associated Redis session.

#### Scenario: Logout revocation
- **WHEN** a user calls `/auth/logout` with a valid refresh token cookie
- **THEN** the refresh token is deleted from Redis and the cookie is cleared
