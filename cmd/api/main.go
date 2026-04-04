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
	"github.com/otaviano/braza-sso/internal/middleware"
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
		_, _ = w.Write([]byte("ok"))
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
	rl := middleware.NewSlidingWindowLimiter(redisClient)

	// Registration & email verification
	regHandler := auth.NewRegistrationHandler(userRepo, tokenStore, mailer, cfg.Pepper, cfg.BaseURL)
	r.With(rl.PerIP(5, 10*time.Minute)).Post("/auth/register", regHandler.Register)
	r.Get("/auth/verify-email", regHandler.VerifyEmail)
	r.Post("/auth/resend-verification", regHandler.ResendVerification)

	// Login & token refresh
	loginHandler := auth.NewLoginHandler(userRepo, tokenStore, tokenSvc, mailer, cfg.Pepper, cfg.BaseURL, cfg.JWTIssuer)
	r.With(rl.PerIP(20, time.Minute)).Post("/auth/login", loginHandler.Login)
	r.Post("/auth/token/refresh", loginHandler.Refresh)

	// Password reset (silent drop on rate limit)
	pwdResetHandler := auth.NewPasswordResetHandler(userRepo, tokenStore, mailer, cfg.Pepper, cfg.BaseURL)
	r.With(rl.PerEmailSilent(3, 15*time.Minute, "email")).Post("/auth/password/reset-request", pwdResetHandler.ResetRequest)
	r.Post("/auth/password/reset", pwdResetHandler.Reset)

	// TOTP 2FA
	recoveryCodeRepo := user.NewRecoveryCodeRepository(cassSession)
	totpHandler := auth.NewTOTPHandler(userRepo, recoveryCodeRepo, tokenStore, tokenSvc, mailer, cfg.Pepper, cfg.JWTIssuer)
	r.Post("/auth/2fa/verify", totpHandler.Verify)
	r.Post("/auth/2fa/recovery", totpHandler.Recovery)

	// OAuth2/OIDC (Phase 11) — declared before logout so notifier can reference it
	oauthClients := oauth.NewClientRepository(cassSession)
	oauthConsents := oauth.NewConsentRepository(cassSession)
	oauthHandlers := oauth.NewOAuthHandlers(oauthClients, oauthConsents, userRepo, redisClient, tokenSvc, cfg.JWTIssuer, cfg.BaseURL)
	r.Get("/.well-known/openid-configuration", oauthHandlers.Discovery)
	r.Post("/oauth/token", oauthHandlers.Token)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(tokenSvc))
		r.Get("/oauth/authorize", oauthHandlers.Authorize)
		r.Post("/oauth/consent", oauthHandlers.Consent)
		r.Get("/oauth/userinfo", oauthHandlers.Userinfo)
	})

	// Logout (Phase 12)
	backChannelNotifier := oauth.NewBackChannelLogoutService(oauthClients, oauthConsents)
	logoutHandler := auth.NewLogoutHandler(tokenStore, tokenSvc, backChannelNotifier)
	r.Post("/auth/logout", logoutHandler.Logout)
	r.Post("/auth/backchannel-logout", auth.BackChannelLogoutReceiver(tokenStore))
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(tokenSvc))
		r.Post("/auth/logout/all", logoutHandler.RevokeAll)
		r.Post("/account/2fa/enroll", totpHandler.Enroll)
		r.Post("/account/2fa/confirm", totpHandler.Confirm)
	})

	// Google federation (Phase 13)
	if cfg.GoogleClientID != "" {
		federatedIdentityRepo := user.NewFederatedIdentityRepository(cassSession)
		fedHandler, err := auth.NewFederationHandler(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.BaseURL,
			userRepo, federatedIdentityRepo, tokenStore, tokenSvc, cfg.Pepper, cfg.JWTIssuer)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to initialize Google federation")
		}
		r.Get("/auth/federation/google", fedHandler.GoogleRedirect)
		r.Get("/auth/federation/google/callback", fedHandler.GoogleCallback)
	}

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
