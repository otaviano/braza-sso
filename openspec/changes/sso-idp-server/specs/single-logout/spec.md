## ADDED Requirements

### Requirement: Centralized Single Logout (SLO)
The system SHALL provide a `/auth/logout` endpoint that invalidates the user's SSO session, revokes all associated refresh tokens in Redis, and redirects to registered service providers' logout URIs.

#### Scenario: SLO initiation
- **WHEN** a user calls `/auth/logout` with a valid session
- **THEN** the system invalidates the SSO session in Redis, clears the refresh token cookie, and initiates back-channel logout notifications to registered SPs

#### Scenario: Back-channel SP logout notification
- **WHEN** SLO is initiated for a user
- **THEN** the system sends a logout token (JWT) to each registered SP's back-channel logout URI

#### Scenario: Logout without active session
- **WHEN** a user calls `/auth/logout` without a valid session cookie
- **THEN** the system returns HTTP 200 (idempotent — already logged out)
