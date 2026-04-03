## ADDED Requirements

### Requirement: Progressive account lockout on failed logins
The system SHALL lock an account after 5 consecutive failed login attempts within a 15-minute window. Lockout duration SHALL be 30 minutes. After lockout, the user SHALL be able to unlock via an email link.

#### Scenario: Failed attempt tracking
- **WHEN** a user submits invalid credentials
- **THEN** the system increments the failed attempt counter in Redis with a 15-minute TTL

#### Scenario: Account lockout trigger
- **WHEN** a user reaches 5 failed login attempts
- **THEN** the system locks the account, stores the lockout expiry, and sends an unlock email to the registered address

#### Scenario: Automatic unlock after timeout
- **WHEN** a locked account's lockout period (30 minutes) expires
- **THEN** the system automatically allows login attempts again and resets the failed counter

#### Scenario: Email unlock
- **WHEN** a user clicks the unlock link in the lockout email with a valid token
- **THEN** the system unlocks the account immediately and resets the failed counter
