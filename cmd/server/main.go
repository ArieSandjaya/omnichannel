package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ariesandjaya/omnichannel/internal/config"
	"github.com/ariesandjaya/omnichannel/internal/handler"
	"github.com/ariesandjaya/omnichannel/internal/middleware"
	"github.com/ariesandjaya/omnichannel/internal/repository"
	"github.com/ariesandjaya/omnichannel/internal/service"
	"github.com/ariesandjaya/omnichannel/pkg/database"
	"github.com/ariesandjaya/omnichannel/pkg/jwt"
	"github.com/ariesandjaya/omnichannel/pkg/response"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database connected")

	// --- Repositories ---
	productRepo := repository.NewProductRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	inventoryRepo := repository.NewInventoryRepository(db)

	// --- Services ---
	checkoutSvc := service.NewCheckoutService(db, productRepo, orderRepo, inventoryRepo)
	posSvc := service.NewPOSService(db, productRepo, orderRepo, inventoryRepo)

	// --- Infrastructure ---
	jwtProvider := jwt.NewProvider(cfg.JWTSecret)
	authMW := middleware.NewAuthMiddleware(jwtProvider)
	resp := response.NewResponder()

	// --- Handlers ---
	checkoutHandler := handler.NewCheckoutHandler(checkoutSvc, resp)
	posHandler := handler.NewPOSHandler(posSvc, resp)

	// --- Router ---
	router := handler.NewRouter(handler.RouterDeps{
		Auth:     authMW,
		Checkout: checkoutHandler,
		POS:      posHandler,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("server stopped")
}
