package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/db"
	"github.com/otaviano/braza-sso/internal/oauth"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	name := flag.String("name", "", "Client application name (required)")
	redirectURIs := flag.String("redirect-uris", "", "Comma-separated redirect URIs (required)")
	scopes := flag.String("scopes", "openid profile email", "Comma-separated scopes")
	logoURL := flag.String("logo-url", "", "Logo URL (optional)")
	backchannelURI := flag.String("backchannel-logout-uri", "", "Back-channel logout URI (optional)")
	flag.Parse()

	if *name == "" || *redirectURIs == "" {
		fmt.Fprintln(os.Stderr, "usage: register-client --name <name> --redirect-uris <uri1,uri2> [--scopes <scopes>] [--logo-url <url>] [--backchannel-logout-uri <uri>]")
		os.Exit(1)
	}

	secret, err := generateSecret()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate secret: %v\n", err)
		os.Exit(1)
	}

	secretHash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to hash secret: %v\n", err)
		os.Exit(1)
	}

	hosts := strings.Split(getenv("CASSANDRA_HOSTS", "localhost"), ",")
	session, err := db.NewCassandraSession(
		hosts,
		getenv("CASSANDRA_KEYSPACE", "braza_sso"),
		getenv("CASSANDRA_USERNAME", ""),
		getenv("CASSANDRA_PASSWORD", ""),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cassandra connection failed: %v\n", err)
		os.Exit(1)
	}
	defer session.Close()

	clientID := uuid.New().String()
	client := &oauth.Client{
		ID:                   clientID,
		SecretHash:           string(secretHash),
		RedirectURIs:         splitTrimmed(*redirectURIs),
		Scopes:               splitTrimmed(*scopes),
		Name:                 *name,
		LogoURL:              *logoURL,
		BackChannelLogoutURI: *backchannelURI,
	}

	repo := oauth.NewClientRepository(session)
	if err := repo.Create(context.Background(), client); err != nil {
		fmt.Fprintf(os.Stderr, "failed to register client: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("client_id:     %s\n", clientID)
	fmt.Printf("client_secret: %s\n", secret)
	fmt.Println("\nSave the client_secret — it will not be shown again.")
}

func generateSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func splitTrimmed(s string) []string {
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
