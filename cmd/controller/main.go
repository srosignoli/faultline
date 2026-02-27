package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/srosignoli/faultline/pkg/api"
	"github.com/srosignoli/faultline/pkg/k8s"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	k8sClient, err := k8s.New("") // attempt in-cluster config
	if err != nil {
		slog.Info("in-cluster config unavailable, falling back to kubeconfig", "reason", err.Error())
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				slog.Error("cannot determine home directory", "err", err)
				os.Exit(1)
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		k8sClient, err = k8s.New(kubeconfig)
		if err != nil {
			slog.Error("failed to build k8s client", "kubeconfig", kubeconfig, "err", err)
			os.Exit(1)
		}
	}
	slog.Info("k8s client ready")

	handler := api.NewHandler(k8sClient)
	mux := api.NewRouter(handler)

	httpSrv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		slog.Info("controller listening", "addr", httpSrv.Addr)
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
