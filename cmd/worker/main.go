package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/config"
	"github.com/Rahmannugar/consumel-server/internal/infra/cache"
	"github.com/Rahmannugar/consumel-server/internal/infra/clients"
	"github.com/Rahmannugar/consumel-server/internal/infra/database"
	"github.com/Rahmannugar/consumel-server/internal/infra/emaildelivery"
	"github.com/Rahmannugar/consumel-server/internal/infra/events"
	"github.com/Rahmannugar/consumel-server/internal/infra/telemetry"
	"github.com/google/uuid"
)

const (
	databaseTimeout = 10 * time.Second
	redisTimeout    = 5 * time.Second
	resendTimeout   = 10 * time.Second
	shutdownTimeout = 15 * time.Second
	// Email calls are slow external I/O; eight slots keep one V1 worker busy
	// while the Resend client independently enforces provider request pacing.
	emailConcurrency = 8
	// Redis publication is shorter I/O and can safely use a wider pool than
	// provider delivery without increasing the PostgreSQL claim batch.
	outboxConcurrency = 16
)

type workerResult struct {
	name string
	err  error
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := run(); err != nil {
		logger.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run() (runError error) {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	telemetryRuntime, err := telemetry.New(context.Background(), "consumel-worker", string(cfg.Environment))
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	logger := telemetryRuntime.Logger()
	slog.SetDefault(logger)
	// Telemetry starts before infrastructure and stops after it so shutdown
	// failures retain their trace and log correlation.
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := telemetryRuntime.Shutdown(ctx); err != nil {
			runError = errors.Join(runError, fmt.Errorf("shutdown telemetry: %w", err))
		}
	}()

	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), databaseTimeout)
	databasePool, err := database.Open(databaseContext, cfg.Database.ConnectionString())
	cancelDatabase()
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer databasePool.Close()

	redisClient, err := cache.Open(cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("connect Redis: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			runError = errors.Join(runError, fmt.Errorf("close Redis client: %w", err))
		}
	}()
	redisContext, cancelRedis := context.WithTimeout(context.Background(), redisTimeout)
	err = redisClient.Ping(redisContext).Err()
	cancelRedis()
	if err != nil {
		return fmt.Errorf("ping Redis: %w", err)
	}

	emailQueue, err := emaildelivery.NewQueue(databasePool, cfg.Auth.OTPHMACSecret)
	if err != nil {
		return fmt.Errorf("configure email delivery queue: %w", err)
	}
	emailSender, err := clients.NewResendEmailClient(
		telemetry.NewHTTPClient(resendTimeout), cfg.Resend.APIKey, cfg.Resend.NoReplyFrom,
	)
	if err != nil {
		return fmt.Errorf("configure Resend email client: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "worker"
	}
	consumerName := hostname + "-" + uuid.NewString()
	relay, err := events.NewRelay(databasePool, redisClient, outboxConcurrency, logger)
	if err != nil {
		return fmt.Errorf("configure outbox relay: %w", err)
	}
	emailConsumer, err := emaildelivery.NewConsumer(
		databasePool, redisClient, emailQueue, emailSender, consumerName, emailConcurrency, logger,
	)
	if err != nil {
		return fmt.Errorf("configure email delivery consumer: %w", err)
	}

	signalContext, stopSignals := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM,
	)
	defer stopSignals()
	workerContext, stopWorkers := context.WithCancel(signalContext)
	defer stopWorkers()
	results := make(chan workerResult, 2)
	go func() { results <- workerResult{name: "outbox-relay", err: relay.Run(workerContext)} }()
	go func() { results <- workerResult{name: "email-deliveries", err: emailConsumer.Run(workerContext)} }()

	logger.Info("worker consumption started", "consumer", consumerName)
	completed := 0
	select {
	case <-signalContext.Done():
		logger.Info("worker shutdown started")
	case result := <-results:
		completed++
		if result.err != nil {
			runError = fmt.Errorf("%s stopped: %w", result.name, result.err)
		} else {
			runError = fmt.Errorf("%s stopped unexpectedly", result.name)
		}
	}
	stopWorkers()

	shutdownTimer := time.NewTimer(shutdownTimeout)
	defer shutdownTimer.Stop()
	for completed < 2 {
		select {
		case result := <-results:
			completed++
			if result.err != nil && !errors.Is(result.err, context.Canceled) {
				runError = errors.Join(runError, fmt.Errorf("%s stopped: %w", result.name, result.err))
			}
		case <-shutdownTimer.C:
			return errors.Join(runError, fmt.Errorf("worker shutdown timed out"))
		}
	}
	logger.Info("worker shutdown completed")
	return runError
}
