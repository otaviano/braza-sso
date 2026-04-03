#!/usr/bin/env bash
# Braza SSO — Cassandra schema migration
# Run after Cassandra is healthy: ./scripts/migrate.sh
set -euo pipefail

CASSANDRA_HOST="${CASSANDRA_HOSTS:-localhost}"
CQLSH="docker compose exec -T cassandra cqlsh $CASSANDRA_HOST"

echo "⏳ Waiting for Cassandra to be ready..."
until docker compose exec -T cassandra nodetool status 2>/dev/null | grep -q "UN"; do
  sleep 5
done
echo "✅ Cassandra is ready."

echo "📦 Applying schema migrations..."

$CQLSH <<'CQL'

-- ── Keyspace ───────────────────────────────────────────────────────────────
CREATE KEYSPACE IF NOT EXISTS braza_sso
  WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}
  AND durable_writes = true;

USE braza_sso;

-- ── Users ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
  user_id       UUID PRIMARY KEY,
  email         TEXT,
  password_hash TEXT,
  email_verified BOOLEAN,
  totp_enabled  BOOLEAN,
  totp_secret   TEXT,
  locked_until  TIMESTAMP,
  failed_attempts INT,
  created_at    TIMESTAMP,
  updated_at    TIMESTAMP
);

-- Email lookup index
CREATE INDEX IF NOT EXISTS users_email_idx ON users (email);

-- ── User recovery codes (2FA) ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_recovery_codes (
  user_id UUID,
  code_hash TEXT,
  used BOOLEAN,
  PRIMARY KEY (user_id, code_hash)
);

-- ── OAuth clients ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS oauth_clients (
  client_id          TEXT PRIMARY KEY,
  client_secret_hash TEXT,
  redirect_uris      LIST<TEXT>,
  scopes             LIST<TEXT>,
  name               TEXT,
  logo_url           TEXT,
  back_channel_logout_uri TEXT,
  created_at         TIMESTAMP
);

-- ── User consents ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_consents (
  user_id    UUID,
  client_id  TEXT,
  scopes     LIST<TEXT>,
  granted_at TIMESTAMP,
  PRIMARY KEY (user_id, client_id)
);

-- ── Federated identities ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS federated_identities (
  user_id          UUID,
  provider         TEXT,
  provider_user_id TEXT,
  email            TEXT,
  PRIMARY KEY (user_id, provider)
);

-- Provider → user lookup
CREATE INDEX IF NOT EXISTS fed_provider_user_idx ON federated_identities (provider_user_id);

CQL

echo "✅ Schema migrations applied successfully."
