## ADDED Requirements

### Requirement: TOTP-based 2FA enrollment
The system SHALL allow users to enroll in TOTP-based 2FA. Upon enrollment, the system SHALL provide a QR code (base32 secret) and a set of one-time recovery codes (hashed at rest with Argon2id).

#### Scenario: Enrollment initiation
- **WHEN** an authenticated user initiates 2FA setup at `/account/2fa/enroll`
- **THEN** the system generates a TOTP secret, returns a QR code URI, and a list of 8 recovery codes

#### Scenario: Enrollment confirmation
- **WHEN** a user submits a valid TOTP code to confirm enrollment
- **THEN** 2FA is activated for the account and recovery codes are stored hashed

### Requirement: TOTP verification on login
The system SHALL require TOTP verification as a second factor when 2FA is enabled on an account.

#### Scenario: Login with 2FA
- **WHEN** a user passes credential validation and 2FA is enabled
- **THEN** the system issues a short-lived intermediate session token and prompts for TOTP code before issuing the final JWT

#### Scenario: Invalid TOTP code
- **WHEN** a user submits an incorrect TOTP code
- **THEN** the system returns HTTP 401 with error code `INVALID_TOTP`

### Requirement: Recovery code usage
The system SHALL allow users to authenticate using a one-time recovery code if they lose access to their TOTP device.

#### Scenario: Valid recovery code
- **WHEN** a user submits a valid, unused recovery code
- **THEN** the system completes authentication, invalidates the used code, and prompts the user to re-enroll 2FA
