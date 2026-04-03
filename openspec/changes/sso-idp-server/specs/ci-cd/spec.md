## ADDED Requirements

### Requirement: GitHub Actions CI pipeline
The system SHALL have a GitHub Actions workflow that runs on every push to `main` and on pull requests: lint (Go + React), unit tests, build Docker images, and push to GHCR.

#### Scenario: Successful CI run on push to main
- **WHEN** code is pushed to `main`
- **THEN** the CI pipeline runs lint, tests, builds Docker images tagged with the commit SHA and `latest`, and pushes to GHCR

#### Scenario: Failed tests block merge
- **WHEN** a pull request has failing tests
- **THEN** the CI status check fails and the PR cannot be merged (branch protection rule)

### Requirement: Automated deployment to VPS
The system SHALL have a GitHub Actions deployment job that SSH-deploys to the Hostinger VPS after a successful CI run on `main`. Deployment SHALL use `docker compose pull && docker compose up -d`.

#### Scenario: Successful deploy
- **WHEN** CI passes on `main`
- **THEN** the deployment job SSHs into the VPS, pulls the new images, and restarts containers with zero manual intervention

#### Scenario: Deployment failure notification
- **WHEN** the deployment job fails
- **THEN** GitHub Actions marks the workflow as failed and notifies the repository owner
