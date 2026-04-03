## ADDED Requirements

### Requirement: Credential validation and JWT issuance
The system SHALL authenticate users by validating their email/password against stored Argon2id hashes and, upon success, issue a signed RS256 JWT access token and an opaque refresh token stored in Redis.

#### Scenario: Successful login
- **WHEN** a user submits valid credentials for an active, verified account
- **THEN** the system returns an RS256 JWT access token (15-minute TTL) and sets a HttpOnly cookie containing the opaque refresh token (7-day TTL)

#### Scenario: Invalid credentials
- **WHEN** a user submits an incorrect password or non-existent email
- **THEN** the system returns HTTP 401 with error code `INVALID_CREDENTIALS` (no distinction between wrong email vs wrong password)

#### Scenario: Unverified account login
- **WHEN** a user with a pending-verification account attempts to log in
- **THEN** the system returns HTTP 403 with error code `EMAIL_NOT_VERIFIED`

#### Scenario: Locked account login
- **WHEN** a user whose account is locked attempts to log in
- **THEN** the system returns HTTP 403 with error code `ACCOUNT_LOCKED` and the unlock timestamp

### Requirement: Session management via Redis
The system SHALL store refresh tokens and session metadata in Redis with TTL enforcement.

#### Scenario: Refresh token reuse
- **WHEN** a previously invalidated refresh token is presented
- **THEN** the system invalidates ALL sessions for that user and returns HTTP 401 with error code `TOKEN_REUSE_DETECTED`

#### Scenario: Access token refresh
- **WHEN** a client presents a valid refresh token cookie to `/auth/token/refresh`
- **THEN** the system issues a new access token and rotates the refresh token (old token invalidated, new one set as HttpOnly cookie)
