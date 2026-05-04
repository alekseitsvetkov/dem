// Package service provides shared lifecycle infrastructure for pipeline services.
//
// The Service interface defines the contract every pipeline service implements.
// Runner orchestrates multiple services with signal-aware graceful shutdown.
// Services are started in the order added and stopped in reverse order.
package service

import (
	"context"
	"log/slog"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Service defines the contract every pipeline service implements.
// The ctx passed to Run is cancelled when the Runner receives SIGTERM or
// SIGINT, allowing services to use it for graceful shutdown of their own
// internal loops.
type Service interface {
	Run(ctx context.Context) error
}

// Runner manages the lifecycle of multiple Service implementations.
// It provides signal-aware graceful shutdown: services are started in the
// order they were added and stopped in reverse order on SIGTERM or SIGINT.
type Runner struct {
	services        []Service
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// RunnerOption is a functional option for configuring a Runner.
type RunnerOption func(*Runner)

// NewRunner creates a new Runner with the given options.
// Defaults: logger=slog.Default(), shutdownTimeout=30s.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{
		logger:          slog.Default(),
		shutdownTimeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithLogger sets the structured logger used by the Runner.
func WithLogger(logger *slog.Logger) RunnerOption {
	return func(r *Runner) {
		r.logger = logger
	}
}

// WithShutdownTimeout sets the maximum time to wait for each service to
// stop during shutdown. Default is 30 seconds.
func WithShutdownTimeout(d time.Duration) RunnerOption {
	return func(r *Runner) {
		r.shutdownTimeout = d
	}
}

// AddService appends a service to the runner. Services are started in the
// order added and stopped in reverse order.
func (r *Runner) AddService(svc Service) {
	r.services = append(r.services, svc)
}

// Run starts all services and blocks until a signal (SIGTERM/SIGINT) is
// received or a service returns an error. On shutdown, services are
// stopped in reverse order, each given shutdownTimeout to complete.
//
// Returns the first error encountered, or nil if all services started and
// stopped cleanly.
func (r *Runner) Run(ctx context.Context) error {
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	r.logger.Info("starting services", slog.Int("count", len(r.services)))

	errCh := make(chan error, len(r.services))
	var wg sync.WaitGroup

	for i, svc := range r.services {
		wg.Add(1)
		go func(idx int, s Service) {
			defer wg.Done()
			if err := s.Run(sigCtx); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(i, svc)
	}

	// Wait for all service goroutines to finish in the background.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	var firstErr error

	// Block until a service error, a signal, or all services complete
	// cleanly.
	select {
	case err := <-errCh:
		firstErr = err
		r.logger.Error("service error, initiating shutdown", slog.Any("error", err))
	case <-sigCtx.Done():
		r.logger.Info("received signal, shutting down", slog.String("signal", sigCtx.Err().Error()))
	case <-done:
		r.logger.Info("all services completed without signal")
	}

	stop()

	// Shut down each service in reverse order with a bounded timeout.
	r.logger.Info("shutting down services", slog.Int("count", len(r.services)))
	for i := len(r.services) - 1; i >= 0; i-- {
		r.logger.Info("stopping service", slog.Int("index", i))
		shutdownCtx, cancel := context.WithTimeout(context.Background(), r.shutdownTimeout)
		err := r.services[i].Run(shutdownCtx)
		cancel()
		if err != nil {
			r.logger.Error("service shutdown error", slog.Int("index", i), slog.Any("error", err))
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}
