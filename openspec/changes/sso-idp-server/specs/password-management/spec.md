## ADDED Requirements

### Requirement: Password reset via email token
The system SHALL allow users to reset their password via a time-limited email token (expires in 1 hour). The reset token MUST be single-use and invalidated after use or expiry.

#### Scenario: Password reset request
- **WHEN** a user submits their email to `/auth/password/reset-request`
- **THEN** the system sends a reset email with a unique token regardless of whether the email exists (to prevent user enumeration) and returns HTTP 200

#### Scenario: Successful password reset
- **WHEN** a user submits a valid, non-expired reset token and a new password meeting policy
- **THEN** the system updates the password hash, invalidates all existing sessions for that user, and returns HTTP 200

#### Scenario: Expired reset token
- **WHEN** a user submits an expired reset token
- **THEN** the system returns HTTP 400 with error code `TOKEN_EXPIRED`

### Requirement: Strong password policy
The system SHALL enforce a password policy: minimum 12 characters, at least one uppercase letter, one lowercase letter, one digit, and one special character.

#### Scenario: Policy enforcement at registration and reset
- **WHEN** a user submits a password not meeting the policy
- **THEN** the system returns HTTP 422 with `WEAK_PASSWORD` and a list of unmet criteria
