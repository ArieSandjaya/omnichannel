package main

import (
	"log"
	"log/slog"

	"github.com/ariesandjaya/omnichannel/internal/broker"
	"github.com/ariesandjaya/omnichannel/internal/config"
	"github.com/ariesandjaya/omnichannel/internal/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config error:", err)
	}

	sseBroker := broker.NewSSEBroker()
	r := handler.SetupRouter(cfg, sseBroker)

	addr := ":" + cfg.AppPort
	slog.Info("server starting",
		"addr", addr,
		"env", cfg.AppEnv,
		"mock", cfg.MockMode,
	)

	if err := r.Run(addr); err != nil {
		log.Fatal("server error:", err)
	}
}
