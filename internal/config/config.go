package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port    string
	BaseURL string
	Env     string

	// Security
	Pepper              string
	JWTPrivateKeyPath   string
	JWTIssuer           string
	JWTAccessTokenTTL   time.Duration
	JWTRefreshTokenTTL  time.Duration

	// Cassandra
	CassandraHosts    []string
	CassandraKeyspace string
	CassandraUsername string
	CassandraPassword string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// SMTP
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// Google OAuth2
	GoogleClientID     string
	GoogleClientSecret string

	// Keycloak OIDC (optional)
	KeycloakIssuer       string
	KeycloakClientID     string
	KeycloakClientSecret string
}

func Load() (*Config, error) {
	cfg := &Config{}
	var missing []string

	required := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	optional := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}

	cfg.Port = optional("PORT", "8080")
	cfg.BaseURL = required("BASE_URL")
	cfg.Env = optional("ENV", "production")

	cfg.Pepper = required("PEPPER")
	cfg.JWTPrivateKeyPath = required("JWT_PRIVATE_KEY_PATH")
	cfg.JWTIssuer = required("JWT_ISSUER")

	accessTTL, _ := time.ParseDuration(optional("JWT_ACCESS_TOKEN_TTL", "15m"))
	cfg.JWTAccessTokenTTL = accessTTL
	refreshTTL, _ := time.ParseDuration(optional("JWT_REFRESH_TOKEN_TTL", "168h"))
	cfg.JWTRefreshTokenTTL = refreshTTL

	cfg.CassandraHosts = []string{optional("CASSANDRA_HOSTS", "localhost")}
	cfg.CassandraKeyspace = optional("CASSANDRA_KEYSPACE", "braza_sso")
	cfg.CassandraUsername = optional("CASSANDRA_USERNAME", "")
	cfg.CassandraPassword = optional("CASSANDRA_PASSWORD", "")

	cfg.RedisAddr = optional("REDIS_ADDR", "localhost:6379")
	cfg.RedisPassword = optional("REDIS_PASSWORD", "")
	redisDB, _ := strconv.Atoi(optional("REDIS_DB", "0"))
	cfg.RedisDB = redisDB

	cfg.SMTPHost = required("SMTP_HOST")
	smtpPort, _ := strconv.Atoi(optional("SMTP_PORT", "587"))
	cfg.SMTPPort = smtpPort
	cfg.SMTPUser = required("SMTP_USER")
	cfg.SMTPPass = required("SMTP_PASS")
	cfg.SMTPFrom = required("SMTP_FROM")

	cfg.GoogleClientID = optional("GOOGLE_CLIENT_ID", "")
	cfg.GoogleClientSecret = optional("GOOGLE_CLIENT_SECRET", "")

	cfg.KeycloakIssuer = optional("KEYCLOAK_ISSUER", "")
	cfg.KeycloakClientID = optional("KEYCLOAK_CLIENT_ID", "")
	cfg.KeycloakClientSecret = optional("KEYCLOAK_CLIENT_SECRET", "")

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}
