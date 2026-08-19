package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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
	// Parse CLI flags for config file path (defaults to config.json)
	configPathFlag := flag.String("config", "config.json", "Path to server JSON configuration file")
	flag.Parse()

	// Load server configuration from config.json file using 100% standard library
	cfg, err := config.LoadConfig(*configPathFlag)
	if err != nil {
		fmt.Printf("Error loading configuration from %s: %v\n", *configPathFlag, err)
		os.Exit(1)
	}

	// Initialize log directory and multi-writer file logging (stdout + logs/app.log)
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		fmt.Printf("Error creating log directory %s: %v\n", cfg.LogDir, err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file %s: %v\n", cfg.LogFile, err)
		os.Exit(1)
	}
	defer logFile.Close()

	// MultiWriter sends logs to stdout AND to logs/app.log simultaneously
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(slog.NewJSONHandler(multiWriter, nil))

	logger.Info("server config loaded successfully",
		"config_file", *configPathFlag,
		"app_name", cfg.AppName,
		"host", cfg.Host,
		"port", cfg.Port,
		"enable_tls", cfg.EnableTLS,
		"enable_rate_limit", cfg.EnableRateLimit,
		"rate_limit_requests", cfg.RateLimitRequests,
		"rate_limit_window_seconds", cfg.RateLimitWindowSeconds,
		"log_file", cfg.LogFile,
	)

	// Initialize SQLite database repository with auto-migration from existing data files.
	dbRepo, err := repository.NewSQLiteRepository(cfg.DBPath, "data")
	if err != nil {
		logger.Error("failed to initialize sqlite repository", "error", err, "db_path", cfg.DBPath)
		os.Exit(1)
	}
	defer dbRepo.Close()

	// Initialize services.
	couponSvc := services.NewCouponService(dbRepo)
	menuSvc := services.NewMenuService(dbRepo)
	authSvc := services.NewAuthService(dbRepo)
	membershipSvc := services.NewMembershipService(dbRepo)
	orderSvc := services.NewOrderService(dbRepo, couponSvc, membershipSvc)
	auditSvc := services.NewAuditService(dbRepo)
	reportSvc := services.NewReportService(dbRepo, dbRepo, dbRepo)
	inventorySvc := services.NewInventoryService(dbRepo, dbRepo)
	securitySvc := services.NewSecurityService(dbRepo)

	// Create application dependencies container.
	app := &handlers.Application{
		Logger:            logger,
		Config:            cfg,
		MenuService:       menuSvc,
		AuthService:       authSvc,
		OrderService:      orderSvc,
		AuditService:      auditSvc,
		ReportService:     reportSvc,
		CouponService:     couponSvc,
		InventoryService:  inventorySvc,
		SecurityService:   securitySvc,
		MembershipService: membershipSvc,
		SettingsRepo:      dbRepo,
	}

	// Initialize Routers: Public Customer Storefront & Private Staff/Admin Portal
	publicMux := http.NewServeMux()
	app.RegisterPublicRoutes(publicMux)

	adminMux := http.NewServeMux()
	app.RegisterAdminRoutes(adminMux)

	// Public Customer Web Server (Default :8080)
	publicSrv := &http.Server{
		Addr:              fmt.Sprintf("%s:%s", cfg.PublicHost, cfg.PublicPort),
		Handler:           publicMux,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Private Staff, Admin & Superadmin Portal Web Server (Default :8081)
	adminSrv := &http.Server{
		Addr:              fmt.Sprintf("%s:%s", cfg.AdminHost, cfg.AdminPort),
		Handler:           adminMux,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown goroutine for both servers.
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

		s := <-quit
		logger.Info("received signal, shutting down servers...", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := publicSrv.Shutdown(ctx); err != nil {
			logger.Error("public server shutdown failed", "error", err)
		}
		if err := adminSrv.Shutdown(ctx); err != nil {
			logger.Error("admin server shutdown failed", "error", err)
		}

		logger.Info("all servers gracefully stopped")
	}()

	if cfg.EnableTLS {
		if err := services.GenerateSelfSignedCert(cfg.SSLCertFile, cfg.SSLKeyFile); err != nil {
			logger.Warn("failed generating TLS certificate", "error", err)
		}

		// Start Admin Server in background
		go func() {
			logger.Info("starting Private Staff & Admin HTTPS portal",
				"domain_url", fmt.Sprintf("https://admin.teachar.in:%s", cfg.AdminPort),
				"localhost_url", fmt.Sprintf("https://localhost:%s", cfg.AdminPort),
				"cert", cfg.SSLCertFile,
				"key", cfg.SSLKeyFile,
			)
			if err := adminSrv.ListenAndServeTLS(cfg.SSLCertFile, cfg.SSLKeyFile); err != nil && err != http.ErrServerClosed {
				logger.Error("admin server encountered error", "error", err)
			}
		}()

		// Start Public Server in foreground
		logger.Info("starting Public Customer HTTPS storefront",
			"domain", "teachar.in",
			"domain_url", fmt.Sprintf("https://teachar.in:%s", cfg.PublicPort),
			"localhost_url", fmt.Sprintf("https://localhost:%s", cfg.PublicPort),
			"cert", cfg.SSLCertFile,
			"key", cfg.SSLKeyFile,
			"log_file", cfg.LogFile,
		)
		err = publicSrv.ListenAndServeTLS(cfg.SSLCertFile, cfg.SSLKeyFile)
	} else {
		// Start Admin Server in background
		go func() {
			logger.Info("starting Private Staff & Admin HTTP portal",
				"address", fmt.Sprintf("http://%s:%s", cfg.AdminHost, cfg.AdminPort),
			)
			if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("admin server encountered error", "error", err)
			}
		}()

		// Start Public Server in foreground
		logger.Info("starting Public Customer HTTP storefront",
			"address", fmt.Sprintf("http://%s:%s", cfg.PublicHost, cfg.PublicPort),
			"log_file", cfg.LogFile,
			"env", cfg.Env,
		)
		err = publicSrv.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		logger.Error("public server encountered error", "error", err)
		os.Exit(1)
	}
}
