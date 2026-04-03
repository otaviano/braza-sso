package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/otaviano/braza-sso/internal/auth"
	"github.com/otaviano/braza-sso/internal/cache"
	"github.com/otaviano/braza-sso/internal/config"
	"github.com/otaviano/braza-sso/internal/db"
	"github.com/otaviano/braza-sso/internal/email"
	"github.com/otaviano/braza-sso/internal/oauth"
	"github.com/otaviano/braza-sso/internal/user"
)

func main() {
	// Structured JSON logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Load config — fatal on missing required vars
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config error")
	}

	// Cassandra
	cassSession, err := db.NewCassandraSession(
		cfg.CassandraHosts,
		cfg.CassandraKeyspace,
		cfg.CassandraUsername,
		cfg.CassandraPassword,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("cassandra connection failed")
	}
	defer cassSession.Close()

	// Redis
	redisClient, err := cache.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatal().Err(err).Msg("redis connection failed")
	}
	defer redisClient.Close()

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log.Info().
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Str("request_id", chimiddleware.GetReqID(req.Context())).
				Msg("request")
			next.ServeHTTP(w, req)
		})
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// JWT token service + JWKS
	tokenSvc, err := auth.NewTokenService(cfg.JWTPrivateKeyPath, cfg.JWTIssuer, cfg.JWTAccessTokenTTL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load JWT private key")
	}
	r.Get("/oauth/jwks.json", oauth.JWKSHandler(tokenSvc))

	// Repositories & services
	userRepo := user.NewRepository(cassSession)
	tokenStore := auth.NewTokenStore(redisClient)
	mailer := email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)

	// Registration & email verification
	regHandler := auth.NewRegistrationHandler(userRepo, tokenStore, mailer, cfg.Pepper, cfg.BaseURL)
	r.Post("/auth/register", regHandler.Register)
	r.Get("/auth/verify-email", regHandler.VerifyEmail)
	r.Post("/auth/resend-verification", regHandler.ResendVerification)

	// Login & token refresh
	loginHandler := auth.NewLoginHandler(userRepo, tokenStore, tokenSvc, mailer, cfg.Pepper, cfg.BaseURL, cfg.JWTIssuer)
	r.Post("/auth/login", loginHandler.Login)
	r.Post("/auth/token/refresh", loginHandler.Refresh)

	// Password reset
	pwdResetHandler := auth.NewPasswordResetHandler(userRepo, tokenStore, mailer, cfg.Pepper, cfg.BaseURL)
	r.Post("/auth/password/reset-request", pwdResetHandler.ResetRequest)
	r.Post("/auth/password/reset", pwdResetHandler.Reset)

	// HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.Port).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// Wait for SIGTERM / SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("forced shutdown")
	}
	log.Info().Msg("server stopped")
}
