## ADDED Requirements

### Requirement: Federated login via external OAuth2/OIDC providers
The system SHALL support federated authentication via Google (OAuth2) and Keycloak (OIDC) as upstream identity providers. Upon successful upstream authentication, the system SHALL create or link a local user account and issue its own JWT.

#### Scenario: Google federated login
- **WHEN** a user clicks "Continue with Google" and completes Google's OAuth2 flow
- **THEN** the system exchanges the Google authorization code for an ID token, extracts the email, creates or retrieves a local account linked to that Google identity, and issues an SSO JWT

#### Scenario: Account linking on first federation
- **WHEN** a federated identity's email matches an existing local account
- **THEN** the system links the federated identity to the existing account (no duplicate created)

#### Scenario: New account from federation
- **WHEN** a federated identity's email does not match any existing account
- **THEN** the system auto-creates a verified account (no email verification required, as the upstream provider is trusted) and issues an SSO JWT

#### Scenario: Federated provider error
- **WHEN** the upstream provider returns an error or the state parameter does not match
- **THEN** the system returns HTTP 400 with error code `FEDERATION_ERROR` and logs the event
