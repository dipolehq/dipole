package main

import (
	"context"
	"encoding/hex"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dipolehq/dipole/cloud/internal/config"
)

func main() {
	logger := setupLogger()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	encKey, err := hex.DecodeString(cfg.EncryptionKey)
	if err != nil || len(encKey) != 32 {
		logger.Error("DIPOLE_ENCRYPTION_KEY must be valid 64-char hex (32 bytes)")
		os.Exit(1)
	}

	logger.Info("starting dipole cloud",
		"region", cfg.Region,
		"grpc_port", cfg.GRPCPort,
		"http_port", cfg.HTTPPort,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	if err := run(ctx, cfg, encKey, logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config, encKey []byte, logger *slog.Logger) error {
	_ = cfg
	_ = encKey

	logger.Info("dipole cloud ready")
	<-ctx.Done()
	logger.Info("shutting down")
	return nil
}

func setupLogger() *slog.Logger {
	level := slog.LevelInfo
	switch os.Getenv("DIPOLE_LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
