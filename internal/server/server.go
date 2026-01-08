package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Addr              string
	Handler           http.Handler
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	HealthCheckPath   string
}

type Runner struct {
	cfg    Config
	server *http.Server
	logger *zap.Logger
}

func NewRunner(cfg Config, logger *zap.Logger) *Runner {
	return &Runner{
		cfg: cfg,
		server: &http.Server{
			Addr:              cfg.Addr,
			Handler:           cfg.Handler,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
		logger: logger,
	}
}

func (r *Runner) Run(ctx context.Context) error {

	serverErr := make(chan error, 1)
	go func() {
		r.logger.Info("Server is ready", zap.String("address", r.cfg.Addr))
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.logger.Error("Server error", zap.Error(err))
			serverErr <- err
		}
	}()

	if r.cfg.HealthCheckPath != "" {
		go r.healthCheck()
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case err := <-serverErr:
		return err
	case sig := <-shutdown:
		r.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		r.logger.Info("Received context cancellation")
	}

	return r.gracefulShutdown()
}

func (r *Runner) healthCheck() {
	time.Sleep(500 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	url := "http://" + r.cfg.Addr + r.cfg.HealthCheckPath

	if r.cfg.Addr[0] == ':' {
		url = "http://localhost" + r.cfg.Addr + r.cfg.HealthCheckPath
	}

	resp, err := client.Get(url)
	if err != nil {
		r.logger.Warn("Health check failed", zap.Error(err), zap.String("url", url))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		r.logger.Info("Server health check passed")
	} else {
		r.logger.Warn("Health check returned non-200 status", zap.Int("status", resp.StatusCode))
	}
}

func (r *Runner) gracefulShutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := r.server.Shutdown(shutdownCtx); err != nil {
		r.logger.Error("Failed to gracefully shutdown server", zap.Error(err))
		return err
	}

	r.logger.Info("Server stopped gracefully")
	return nil
}
