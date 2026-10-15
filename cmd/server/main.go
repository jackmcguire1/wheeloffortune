// Command server runs the wheel of fortune API.
//
// It serves Connect, gRPC and gRPC-Web on a single port, so browsers talk to
// it directly without a proxy.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"

	"github.com/jackmcguire1/wheeloffortune/gen/wheel/v1/wheelv1connect"
	"github.com/jackmcguire1/wheeloffortune/internal/auth"
	"github.com/jackmcguire1/wheeloffortune/internal/server"
	"github.com/jackmcguire1/wheeloffortune/internal/wheel"
	"github.com/jackmcguire1/wheeloffortune/internal/wheel/redisstore"
)

type config struct {
	addr        string
	redisURL    string
	authMode    string
	projectID   string
	corsOrigins string
	logLevel    string
	logJSON     bool
}

func main() {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", env("ADDR", ":7755"), "listen address")
	flag.StringVar(&cfg.redisURL, "redis-url", env("REDIS_URL", "redis://localhost:6379/0"), "Redis connection URL")
	flag.StringVar(&cfg.authMode, "auth", env("AUTH_MODE", "firebase"), `"firebase" verifies Firebase ID tokens, "none" treats every caller as one local user`)
	flag.StringVar(&cfg.projectID, "firebase-project", env("FIREBASE_PROJECT_ID", "demo-wheeloffortune"), "Firebase project id")
	flag.StringVar(&cfg.corsOrigins, "cors-origins", env("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"), "comma separated allowed browser origins")
	flag.StringVar(&cfg.logLevel, "log-level", env("LOG_LEVEL", "info"), "debug, info, warn or error")
	flag.BoolVar(&cfg.logJSON, "log-json", env("LOG_FORMAT", "text") == "json", "log as JSON")
	flag.Parse()

	log := newLogger(cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, cfg, log)
	stop()
	if err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config, log *slog.Logger) error {
	redisOpts, err := redis.ParseURL(cfg.redisURL)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(redisOpts)
	defer func() { _ = rdb.Close() }()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("connect to redis at %s: %w", redisOpts.Addr, err)
	}

	verifier, err := newVerifier(ctx, cfg, log)
	if err != nil {
		return err
	}

	svc := wheel.NewService(redisstore.New(rdb))
	mux := http.NewServeMux()
	mux.Handle(wheelv1connect.NewWheelServiceHandler(
		server.New(svc, log),
		connect.WithInterceptors(auth.NewInterceptor(verifier)),
	))
	checker := grpchealth.NewStaticChecker(wheelv1connect.WheelServiceName)
	mux.Handle(grpchealth.NewHandler(checker))
	reflector := grpcreflect.NewStaticReflector(wheelv1connect.WheelServiceName)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true) // gRPC clients without TLS
	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           withCORS(cfg.corsOrigins, mux),
		Protocols:         protocols,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", cfg.addr, "auth", cfg.authMode)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newVerifier(ctx context.Context, cfg config, log *slog.Logger) (auth.Verifier, error) {
	switch cfg.authMode {
	case "firebase":
		if host := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"); host != "" {
			log.Info("using firebase auth emulator", "host", host, "project", cfg.projectID)
		}
		return auth.NewFirebase(ctx, cfg.projectID)
	case "none":
		log.Warn("auth disabled: every request acts as the local user")
		return auth.Static{User: wheel.User{UID: "local", Name: "Local user"}}, nil
	}
	return nil, fmt.Errorf("unknown auth mode %q", cfg.authMode)
}

func withCORS(origins string, h http.Handler) http.Handler {
	var allowed []string
	for o := range strings.SplitSeq(origins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowed = append(allowed, o)
		}
	}
	return cors.New(cors.Options{
		AllowedOrigins: allowed,
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: append(connectcors.AllowedHeaders(), "Authorization"),
		ExposedHeaders: connectcors.ExposedHeaders(),
		MaxAge:         7200,
	}).Handler(h)
}

func newLogger(cfg config) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.logLevel)); err != nil {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	if cfg.logJSON {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
