## ADDED Requirements

### Requirement: Docker Compose provisioning
The system SHALL be fully provisioned via Docker Compose with services for the Go API, React frontend (Nginx), Cassandra, and Redis. All services SHALL have health checks and restart policies.

#### Scenario: Fresh environment startup
- **WHEN** a developer runs `docker compose up -d` in the repo root
- **THEN** all four services start, pass health checks, and the API is reachable at `http://localhost:8080`

#### Scenario: Cassandra startup dependency
- **WHEN** Cassandra has not yet passed its health check
- **THEN** the Go API container waits (via `depends_on` + health check condition) before starting

### Requirement: Environment-based configuration
The system SHALL load all sensitive configuration from environment variables (no hardcoded secrets). An `.env.example` file SHALL document all required variables.

#### Scenario: Missing required env var
- **WHEN** the API starts without a required env var (e.g., `PEPPER`, `JWT_PRIVATE_KEY_PATH`)
- **THEN** the application exits with a clear error message listing the missing variable

### Requirement: Repository structure
The system SHALL include a `.gitignore` excluding secrets, build artifacts, and IDE files, and a `README.md` documenting local setup, environment variables, and deployment steps.

#### Scenario: README completeness
- **WHEN** a developer follows the README
- **THEN** they can run the full stack locally without additional guidance
