-- Braza SSO — Cassandra schema
-- Run once against a fresh keyspace:
--   docker exec -i braza-sso-cassandra-1 cqlsh < db/schema.cql

CREATE KEYSPACE IF NOT EXISTS braza_sso
  WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE braza_sso;

-- ─── Users ────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS users (
    user_id         UUID PRIMARY KEY,
    email           TEXT,
    password_hash   TEXT,
    email_verified  BOOLEAN,
    totp_enabled    BOOLEAN,
    totp_secret     TEXT,
    locked_until    TIMESTAMP,
    failed_attempts INT,
    created_at      TIMESTAMP,
    updated_at      TIMESTAMP
);

-- Email → user_id lookup table (avoids ALLOW FILTERING on users.email)
CREATE TABLE IF NOT EXISTS users_by_email (
    email   TEXT PRIMARY KEY,
    user_id UUID
);

-- ─── Federated Identities ─────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS federated_identities (
    user_id          UUID,
    provider         TEXT,
    provider_user_id TEXT,
    email            TEXT,
    linked_at        TIMESTAMP,
    PRIMARY KEY (user_id, provider)
);

-- ─── Recovery Codes ───────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS user_recovery_codes (
    user_id    UUID,
    code_id    UUID,
    code_hash  TEXT,
    used       BOOLEAN,
    created_at TIMESTAMP,
    PRIMARY KEY (user_id, code_id)
);

-- ─── OAuth2 Clients ───────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS oauth_clients (
    client_id                UUID PRIMARY KEY,
    client_secret_hash       TEXT,
    redirect_uris            LIST<TEXT>,
    scopes                   LIST<TEXT>,
    name                     TEXT,
    logo_url                 TEXT,
    backchannel_logout_uri   TEXT
);

-- ─── User Consents ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS user_consents (
    user_id    UUID,
    client_id  TEXT,
    scopes     LIST<TEXT>,
    granted_at TIMESTAMP,
    PRIMARY KEY (user_id, client_id)
);
