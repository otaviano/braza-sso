## ADDED Requirements

### Requirement: User self-registration with email verification
The system SHALL allow unauthenticated users to register an account by providing an email address and password. Upon submission, the system SHALL send a time-limited verification email before activating the account.

#### Scenario: Successful registration
- **WHEN** a user submits a valid email and password that meets the password policy
- **THEN** the system creates the account in a pending-verification state, sends a verification email with a unique token (expires in 24h), and returns HTTP 201

#### Scenario: Duplicate email
- **WHEN** a user submits an email already associated with an existing account
- **THEN** the system returns HTTP 409 with error code `EMAIL_ALREADY_EXISTS`

#### Scenario: Weak password
- **WHEN** a user submits a password that does not meet policy (min 12 chars, at least 1 upper, 1 lower, 1 digit, 1 special)
- **THEN** the system returns HTTP 422 with error code `WEAK_PASSWORD` and a description of unmet criteria

#### Scenario: Email verification success
- **WHEN** a user clicks the verification link with a valid, non-expired token
- **THEN** the system activates the account and redirects to the login page

#### Scenario: Expired verification token
- **WHEN** a user clicks the verification link with an expired token
- **THEN** the system returns HTTP 400 with error code `TOKEN_EXPIRED` and offers to resend the verification email
