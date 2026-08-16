package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"teachar.in/config"
	"teachar.in/handlers"
	"teachar.in/repository"
	"teachar.in/services"
)

func main() {
	// Initialize structured logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Load application configuration.
	cfg, err := config.New()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize the persistent standard library JSON repository.
	dbRepo, err := repository.NewJSONRepository("data/db.json")
	if err != nil {
		logger.Error("failed to initialize persistent repository", "error", err)
		os.Exit(1)
	}

	// Initialize services.
	menuSvc := services.NewMenuService(dbRepo)
	authSvc := services.NewAuthService(dbRepo)
	orderSvc := services.NewOrderService(dbRepo)
	auditSvc := services.NewAuditService(dbRepo)
	reportSvc := services.NewReportService(dbRepo, dbRepo)

	// Create application dependencies container.
	app := &handlers.Application{
		Logger:        logger,
		Config:        cfg,
		MenuService:   menuSvc,
		AuthService:   authSvc,
		OrderService:  orderSvc,
		AuditService:  auditSvc,
		ReportService: reportSvc,
	}

	// Initialize router.
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful shutdown goroutine.
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

		s := <-quit
		logger.Info("received signal, shutting down server...", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("server shutdown failed", "error", err)
		}

		logger.Info("server gracefully stopped")
	}()

	logger.Info("starting server", "address", srv.Addr, "env", cfg.Env)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server startup failed", "error", err)
		os.Exit(1)
	}
}
