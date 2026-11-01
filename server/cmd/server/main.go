package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/buricodes/fuck-algo/server/internal/config"
	httpapi "github.com/buricodes/fuck-algo/server/internal/http"
	"github.com/buricodes/fuck-algo/server/internal/service/anthropic"
	"github.com/buricodes/fuck-algo/server/internal/service/checks"
	"github.com/buricodes/fuck-algo/server/internal/service/claudecli"
	"github.com/buricodes/fuck-algo/server/internal/service/elevenlabs"
	"github.com/buricodes/fuck-algo/server/internal/service/jev"
	"github.com/buricodes/fuck-algo/server/internal/service/payments"
	"github.com/buricodes/fuck-algo/server/internal/service/voice"
	"github.com/buricodes/fuck-algo/server/internal/store"
)

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 90 * time.Second
	idleTimeout       = 2 * time.Minute
	shutdownTimeout   = 30 * time.Second
	startupTimeout    = 30 * time.Second
	revealSweepEvery  = time.Hour
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	loaded := loadEnvFiles()
	cfg, err := config.Load(envOr("CONFIG_PATH", "config/config.yaml"))
	if err != nil {
		return err
	}
	slog.Info("starting", "env", cfg.Env, "envFiles", loaded)
	gin.SetMode(gin.ReleaseMode)

	db, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	router, err := httpapi.NewRouter(deps(cfg, db))
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go sweepRevealKeys(ctx, db, cfg.Licence.RevealWindow())
	return serve(ctx, router, cfg.Server.Port)
}

func loadEnvFiles() []string {
	var loaded []string
	try := func(path string) {
		if config.LoadDotEnv(path) {
			loaded = append(loaded, path)
		}
	}
	try(".env")
	if envOr("APP_ENV", config.EnvDevelopment) == config.EnvDevelopment {
		try("../.env.local")
		try("../.env")
	}
	return loaded
}

func openStore(cfg *config.Config) (store.Store, error) {
	if cfg.Secrets.DatabaseURL == "" {
		slog.Warn("DATABASE_URL is empty: using the in memory store, all data is lost on restart (development only)")
		return store.NewMemory(), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()
	return store.OpenPostgres(ctx, cfg.Secrets.DatabaseURL, cfg.DB.MaxConns)
}

func deps(cfg *config.Config, db store.Store) httpapi.Deps {
	s := cfg.Secrets
	d := httpapi.Deps{
		Config: cfg,
		Store:  db,
		STT:    elevenlabs.New(s.ElevenLabsKey, cfg.ElevenLabs),
		Checks: checks.New(checksJudge(cfg),
			time.Duration(cfg.ChecksCache.TTLHours)*time.Hour, cfg.ChecksCache.MaxEntries, time.Now),
		Voice: voice.New(textModel(cfg, "voice-rules", cfg.AI.Rules, cfg.Anthropic.Rules)),
		Payments: payments.NewDodo(payments.Settings{
			APIKey:        s.DodoAPIKey,
			WebhookSecret: s.DodoWebhookSecret,
			ProductID:     s.DodoProductID,
			WebsiteURL:    cfg.WebsiteURL,
			Live:          !cfg.IsDevelopment(),
		}),
		Now: func() time.Time { return time.Now().UTC() },
	}
	warnMissing(d)
	return d
}

func checksJudge(cfg *config.Config) checks.Judge {
	if cfg.AI.Checks == config.ProviderJev {
		return checks.NewJevJudge(jev.New(cfg.Secrets.OpenRouterKey, cfg.Jev))
	}
	return checks.NewModelJudge(textModel(cfg, "checks", cfg.AI.Checks, cfg.Anthropic.Checks))
}

func textModel(cfg *config.Config, label, provider string, api config.Model) checks.Model {
	if provider == config.ProviderCLI {
		return claudecli.New(label, cfg.ClaudeCLI)
	}
	return anthropic.New(label, cfg.Secrets.AnthropicKey, api)
}

func warnMissing(d httpapi.Deps) {
	slog.Info("ai providers", "checks", d.Config.AI.Checks, "rules", d.Config.AI.Rules)
	if !d.Checks.Available() {
		slog.Warn("AI checks provider is not configured: /v1/checks answers unavailable", "provider", d.Config.AI.Checks)
	}
	if !d.Voice.Available() {
		slog.Warn("voice rules provider is not configured: /v1/voice/rules answers unavailable", "provider", d.Config.AI.Rules)
	}
	if !d.STT.Available() {
		slog.Warn("ELEVENLABS_API_KEY is empty: /v1/voice/token and /v1/voice/transcribe answer unavailable")
	}
	if !d.Payments.Available() {
		slog.Warn("Dodo settings are incomplete: /v1/checkout and /v1/webhooks/dodo answer unavailable")
	}
}

func sweepRevealKeys(ctx context.Context, db store.Store, window time.Duration) {
	ticker := time.NewTicker(revealSweepEvery)
	defer ticker.Stop()
	for {
		if err := db.ClearRevealKeys(ctx, time.Now().Add(-window)); err != nil && ctx.Err() == nil {
			slog.Warn("clearing old reveal keys failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func serve(ctx context.Context, handler http.Handler, port int) error {
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	errs := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", server.Addr)
		errs <- server.ListenAndServe()
	}()
	select {
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	slog.Info("shutting down", "timeout", shutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
