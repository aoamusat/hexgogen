package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/aoamusat/hexgogen/api"
	"github.com/aoamusat/hexgogen/configs"
	"github.com/aoamusat/hexgogen/domain/services"
	"github.com/aoamusat/hexgogen/pkg/logger"
)

func main() {
	log := logger.NewZerolog()
	if err := run(log); err != nil {
		log.Error("Application stopped", err)
		os.Exit(1)
	}
}

func run(log services.Logger) error {
	cfg, err := configs.LoadConfig()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return api.Start(ctx, cfg, log)
}
