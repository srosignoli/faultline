package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/srosignoli/faultline/pkg/config"
	"github.com/srosignoli/faultline/pkg/parser"
	"github.com/srosignoli/faultline/pkg/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	dumpPath := envOr("FAULTLINE_DUMP_PATH", "/etc/faultline/dump.txt")
	rulesPath := envOr("FAULTLINE_RULES_PATH", "/etc/faultline/rules.yaml")

	// Parse the Prometheus dump.
	f, err := os.Open(dumpPath)
	if err != nil {
		slog.Error("failed to open dump file", "path", dumpPath, "err", err)
		os.Exit(1)
	}
	metrics, err := parser.ParseDump(f)
	f.Close()
	if err != nil {
		slog.Error("failed to parse dump", "path", dumpPath, "err", err)
		os.Exit(1)
	}
	slog.Info("dump parsed", "metrics", len(metrics))

	// Load mutation rules.
	cfg, err := config.LoadConfig(rulesPath)
	if err != nil {
		slog.Error("failed to load rules", "path", rulesPath, "err", err)
		os.Exit(1)
	}
	rules, err := config.BuildRules(cfg)
	if err != nil {
		slog.Error("failed to build rules", "err", err)
		os.Exit(1)
	}
	slog.Info("rules loaded", "count", len(rules))

	// Wire up the HTTP server.
	sim := server.New(metrics, rules)
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", sim.MetricsHandler)

	httpSrv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		slog.Info("simulator listening", "addr", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
		os.Exit(1)
	}
	slog.Info("stopped")
}

func envOr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
