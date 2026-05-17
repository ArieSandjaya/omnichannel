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
	"github.com/go-resty/resty/v2"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/ariesandjaya/omnichannel/internal/broker"
	"github.com/ariesandjaya/omnichannel/internal/config"
	"github.com/ariesandjaya/omnichannel/internal/gateway"
	"github.com/ariesandjaya/omnichannel/internal/handler"
	"github.com/ariesandjaya/omnichannel/internal/repository"
	"github.com/ariesandjaya/omnichannel/internal/service"
	"github.com/ariesandjaya/omnichannel/internal/worker"
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
	logLevel := slog.LevelInfo
	if cfg.AppEnv == "development" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))

	// ── PostgreSQL ────────────────────────────────────────────────────────
	poolCtx, poolCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer poolCancel()

	pool, err := pgxpool.New(poolCtx, cfg.DBConnString())
	if err != nil {
		slog.Error("database connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(poolCtx); err != nil {
		slog.Error("database ping failed", "err", err)
		os.Exit(1)
	}
	slog.Info("database connected", "host", cfg.DB.Host, "db", cfg.DB.Name)

	// ── Redis ─────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Warn("redis ping failed", "err", err)
	} else {
		slog.Info("redis connected", "addr", cfg.Redis.Addr)
	}
	defer rdb.Close()

	// ── Asynq ─────────────────────────────────────────────────────────────
	asynqRedisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	asynqClient := asynq.NewClient(asynqRedisOpt)
	defer asynqClient.Close()

	// ── External clients ──────────────────────────────────────────────────
	xenditGw := gateway.NewXenditGateway(cfg.Xendit.SecretKey)

	biteshipClient := resty.New().
		SetBaseURL(cfg.Biteship.APIURL).
		SetHeader("Authorization", "Bearer "+cfg.Biteship.APIKey).
		SetTimeout(15 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(500 * time.Millisecond)
	if cfg.AppEnv == "development" {
		biteshipClient.SetDebug(true)
	}

	// ── SSE Broker ────────────────────────────────────────────────────────
	sseBroker := broker.NewSSEBroker()
	go sseBroker.Run()
	defer sseBroker.Stop()

	// ── Repository adapter ────────────────────────────────────────────────
	// RawAdapter uses pgxpool with raw SQL queries.
	// After running `make sqlc`, replace with the generated *db.Queries wrapper.
	dbAdapter := repository.NewRawAdapter(pool)

	// ── Services ──────────────────────────────────────────────────────────
	paymentSvc := service.NewPaymentService(dbAdapter, xenditGw, sseBroker, asynqClient, cfg)
	shippingSvc := service.NewShippingService(dbAdapter, biteshipClient, cfg)

	// ── Asynq worker server ───────────────────────────────────────────────
	asynqServer := asynq.NewServer(asynqRedisOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"webhooks":   8,
			"stock_sync": 2,
		},
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(worker.TypeWebhookXenditQRIS,
		worker.NewXenditQRISWebhookHandler(paymentSvc).ProcessTask)
	mux.HandleFunc(worker.TypeWebhookXenditVA,
		worker.NewXenditVAWebhookHandler(paymentSvc).ProcessTask)
	mux.HandleFunc(worker.TypeWebhookBiteship,
		worker.NewBiteshipWebhookHandler(shippingSvc).ProcessTask)
	mux.HandleFunc(worker.TypeStockSync,
		worker.NewStockSyncHandler().ProcessTask)

	go func() {
		if err := asynqServer.Run(mux); err != nil {
			slog.Error("asynq server error", "err", err)
		}
	}()

	// ── HTTP server ───────────────────────────────────────────────────────
	r := handler.SetupRouter(handler.RouterDeps{
		Config:      cfg,
		PaymentSvc:  paymentSvc,
		ShippingSvc: shippingSvc,
		SSEBroker:   sseBroker,
		AsynqClient: asynqClient,
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
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.AppPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
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
	// ── Graceful shutdown ─────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	srv.Shutdown(shutCtx)
	asynqServer.Shutdown()
	slog.Info("server stopped")
}
