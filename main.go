package main

import (
	"context"
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

	// Initialize the high-performance multi-file domain-isolated repository.
	dbRepo, err := repository.NewMultiFileRepository("data")
	if err != nil {
		logger.Error("failed to initialize multi-file repository", "error", err)
		os.Exit(1)
	}

	// Initialize services.
	couponSvc := services.NewCouponService(dbRepo)
	menuSvc := services.NewMenuService(dbRepo)
	authSvc := services.NewAuthService(dbRepo)
	orderSvc := services.NewOrderService(dbRepo, couponSvc)
	auditSvc := services.NewAuditService(dbRepo)
	reportSvc := services.NewReportService(dbRepo, dbRepo, dbRepo)
	inventorySvc := services.NewInventoryService(dbRepo, dbRepo)
	securitySvc := services.NewSecurityService(dbRepo)

	// Create application dependencies container.
	app := &handlers.Application{
		Logger:           logger,
		Config:           cfg,
		MenuService:      menuSvc,
		AuthService:      authSvc,
		OrderService:     orderSvc,
		AuditService:     auditSvc,
		ReportService:    reportSvc,
		CouponService:    couponSvc,
		InventoryService: inventorySvc,
		SecurityService:  securitySvc,
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

	if cfg.EnableTLS {
		if err := services.GenerateSelfSignedCert(cfg.SSLCertFile, cfg.SSLKeyFile); err != nil {
			logger.Warn("failed generating TLS certificate", "error", err)
		}
		logger.Info("starting HTTPS TLS server", "address", srv.Addr, "cert", cfg.SSLCertFile, "env", cfg.Env)
		err = srv.ListenAndServeTLS(cfg.SSLCertFile, cfg.SSLKeyFile)
	} else {
		logger.Info("starting HTTP server", "address", srv.Addr, "env", cfg.Env)
		err = srv.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		logger.Error("server encountered error", "error", err)
		os.Exit(1)
	}
}
