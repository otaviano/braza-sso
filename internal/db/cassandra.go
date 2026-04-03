package db

import (
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
)

func NewCassandraSession(hosts []string, keyspace, username, password string) (*gocql.Session, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 10 * time.Second
	cluster.ConnectTimeout = 10 * time.Second
	cluster.NumConns = 2

	if username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: username,
			Password: password,
		}
	}

	// Retry on startup — Cassandra can be slow to start
	var session *gocql.Session
	var err error
	for attempt := 1; attempt <= 10; attempt++ {
		session, err = cluster.CreateSession()
		if err == nil {
			log.Info().Msg("connected to Cassandra")
			return session, nil
		}
		log.Warn().Err(err).Int("attempt", attempt).Msg("waiting for Cassandra...")
		time.Sleep(time.Duration(attempt) * 3 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to Cassandra after 10 attempts: %w", err)
}
