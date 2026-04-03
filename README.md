# Braza SSO

Self-hosted Identity Provider (IdP/SSO) with OAuth2/OIDC, built in Go + React. Supports Google federation, TOTP 2FA, Argon2id password hashing, RS256 JWTs, and GitHub Actions CI/CD to Hostinger VPS.

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22+ (chi router, zerolog) |
| Frontend | React 18 + Vite + TypeScript |
| Database | Apache Cassandra |
| Cache / Sessions | Redis |
| Password hashing | Argon2id (memory: 64MB, time: 3, threads: 4) + pepper |
| Tokens | RS256 JWT (access) + opaque refresh tokens |
| Infrastructure | Docker Compose + Caddy (TLS) |
| CI/CD | GitHub Actions → GHCR → Hostinger VPS |

## Prerequisites

- Go 1.22+
- Node 18+ / npm 9+
- Docker + Docker Compose v2
- `gh` CLI (optional, for repo management)

## Local Setup

### 1. Clone and configure

```bash
git clone git@github.com:otaviano/braza-sso.git
cd braza-sso
cp .env.example .env
# Edit .env with your values
```

### 2. Generate RSA key pair

```bash
mkdir -p keys
openssl genrsa -out keys/private.pem 4096
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
```

### 3. Start all services

```bash
docker compose up -d
```

### 4. Run Cassandra migrations

```bash
./scripts/migrate.sh
```

### 5. Start development servers

```bash
# Backend
go run ./cmd/api

# Frontend (separate terminal)
cd frontend && npm install && npm run dev
```

API: http://localhost:8080  
Frontend: http://localhost:5173

## Environment Variables

See [.env.example](.env.example) for all required variables.

| Variable | Description |
|---|---|
| `PEPPER` | Server-side pepper for Argon2id hashing |
| `JWT_PRIVATE_KEY_PATH` | Path to RSA private key PEM file |
| `JWT_ISSUER` | JWT issuer URL (e.g. `https://sso.yourdomain.com`) |
| `CASSANDRA_HOSTS` | Comma-separated Cassandra hosts |
| `CASSANDRA_KEYSPACE` | Cassandra keyspace name (`braza_sso`) |
| `REDIS_ADDR` | Redis address (e.g. `localhost:6379`) |
| `SMTP_HOST` | SMTP relay host |
| `SMTP_PORT` | SMTP relay port |
| `SMTP_USER` | SMTP username |
| `SMTP_PASS` | SMTP password |
| `SMTP_FROM` | From email address |
| `GOOGLE_CLIENT_ID` | Google OAuth2 client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth2 client secret |
| `BASE_URL` | Public base URL of the SSO server |
| `PORT` | API server port (default: `8080`) |

## Project Structure

```
braza-sso/
├── cmd/api/              # Application entrypoint
├── internal/
│   ├── auth/             # Login, register, JWT, sessions
│   ├── user/             # User model and Cassandra repository
│   ├── oauth/            # OAuth2/OIDC flows
│   ├── middleware/        # Rate limiting, auth middleware
│   ├── db/               # Cassandra driver setup
│   ├── cache/            # Redis client
│   └── email/            # Email templates and SMTP client
├── frontend/             # Vite + React + TypeScript SPA
├── scripts/              # Migration and utility scripts
├── keys/                 # RSA key pair (not committed)
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

## Deployment

Deployed via GitHub Actions on push to `main`:
1. Lint + test
2. Build Docker images → push to GHCR
3. SSH to Hostinger VPS → `docker compose pull && docker compose up -d`

## License

MIT
