## ADDED Requirements

### Requirement: IP-based and account-based rate limiting
The system SHALL enforce rate limits on authentication endpoints using a Redis sliding window counter. Limits SHALL apply per IP address and per account independently.

#### Scenario: IP rate limit on login
- **WHEN** more than 20 login requests from the same IP occur within 1 minute
- **THEN** the system returns HTTP 429 with `Retry-After` header indicating when requests can resume

#### Scenario: Account rate limit on login
- **WHEN** more than 10 login attempts for the same account occur within 5 minutes
- **THEN** the system returns HTTP 429 with `Retry-After` header

#### Scenario: Rate limit on registration
- **WHEN** more than 5 registration requests from the same IP occur within 10 minutes
- **THEN** the system returns HTTP 429

#### Scenario: Rate limit on password reset
- **WHEN** more than 3 password reset requests for the same email occur within 15 minutes
- **THEN** the system silently discards subsequent requests (to prevent timing attacks) but does not send additional emails
