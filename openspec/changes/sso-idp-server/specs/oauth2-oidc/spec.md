## ADDED Requirements

### Requirement: OAuth2 Authorization Code Flow
The system SHALL implement the OAuth2 authorization code flow. Registered client applications SHALL be able to redirect users to the SSO authorization endpoint, receive an authorization code, and exchange it for tokens.

#### Scenario: Authorization request
- **WHEN** a client redirects a user to `/oauth/authorize` with valid `client_id`, `redirect_uri`, `response_type=code`, `scope`, and `state`
- **THEN** the system authenticates the user (if not already), shows a consent screen, and redirects to `redirect_uri` with `code` and `state`

#### Scenario: Invalid redirect_uri
- **WHEN** the `redirect_uri` does not match a pre-registered URI for the client
- **THEN** the system returns HTTP 400 with error `INVALID_REDIRECT_URI` (no redirect to client)

#### Scenario: Code exchange
- **WHEN** a client POSTs to `/oauth/token` with `grant_type=authorization_code`, valid `code`, `client_id`, and `client_secret`
- **THEN** the system returns an access token, refresh token, and ID token (OIDC)

#### Scenario: Expired authorization code
- **WHEN** a client presents an authorization code older than 60 seconds
- **THEN** the system returns HTTP 400 with error `invalid_grant`

### Requirement: OIDC Discovery and JWKS
The system SHALL expose `/.well-known/openid-configuration` and `/oauth/jwks.json` endpoints for service provider auto-configuration.

#### Scenario: Discovery endpoint
- **WHEN** a service provider fetches `/.well-known/openid-configuration`
- **THEN** the system returns a valid OIDC discovery document including `issuer`, `authorization_endpoint`, `token_endpoint`, `userinfo_endpoint`, and `jwks_uri`

### Requirement: Consent screen
The system SHALL display a consent screen to users before authorizing a client application, listing requested scopes.

#### Scenario: First-time consent
- **WHEN** a user has not previously consented to a client's requested scopes
- **THEN** the system displays a consent screen listing the client name, logo, and requested scopes

#### Scenario: Pre-consented scope
- **WHEN** a user has previously consented to the exact scopes requested
- **THEN** the system skips the consent screen and proceeds with authorization
