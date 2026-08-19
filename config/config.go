package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds the application configuration.
type Config struct {
	AppName                string `json:"app_name"`
	Host                   string `json:"host"`
	Port                   string `json:"port"`
	PublicHost             string `json:"public_host"`
	PublicPort             string `json:"public_port"`
	AdminHost              string `json:"admin_host"`
	AdminPort              string `json:"admin_port"`
	Env                    string `json:"env"`
	EnableTLS              bool   `json:"enable_tls"`
	SSLCertFile            string `json:"ssl_cert_file"`
	SSLKeyFile             string `json:"ssl_key_file"`
	SSLPort                string `json:"ssl_port"`
	EnableRateLimit        bool   `json:"enable_rate_limit"`
	RateLimitRequests      int    `json:"rate_limit_requests"`
	RateLimitWindowSeconds int    `json:"rate_limit_window_seconds"`
	DBPath                 string `json:"db_path"`
	LogDir                 string `json:"log_dir"`
	LogFile                string `json:"log_file"`
}

// New loads configuration from server config.json file, environment variables, or sensible defaults using 100% Go Standard Library.
func New() (*Config, error) {
	return LoadConfig("config.json")
}

// LoadConfig reads the specified JSON config file and overrides with environment variables if present.
func LoadConfig(configPath string) (*Config, error) {
	cfg := &Config{
		EnableRateLimit:        true,
		RateLimitRequests:      60,
		RateLimitWindowSeconds: 60,
	}

	// 1. Try reading from JSON config file if present
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	// 2. Override or populate defaults using standard library os.Getenv
	if envAppName := os.Getenv("APP_NAME"); envAppName != "" {
		cfg.AppName = envAppName
	}
	if cfg.AppName == "" {
		cfg.AppName = "teachar.in"
	}

	if envHost := os.Getenv("APP_HOST"); envHost != "" {
		cfg.Host = envHost
	}
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}

	if envPort := os.Getenv("APP_PORT"); envPort != "" {
		cfg.Port = envPort
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if envPublicHost := os.Getenv("PUBLIC_HOST"); envPublicHost != "" {
		cfg.PublicHost = envPublicHost
	}
	if cfg.PublicHost == "" {
		cfg.PublicHost = cfg.Host
	}

	if envPublicPort := os.Getenv("PUBLIC_PORT"); envPublicPort != "" {
		cfg.PublicPort = envPublicPort
	}
	if cfg.PublicPort == "" {
		cfg.PublicPort = cfg.Port
	}

	if envAdminHost := os.Getenv("ADMIN_HOST"); envAdminHost != "" {
		cfg.AdminHost = envAdminHost
	}
	if cfg.AdminHost == "" {
		cfg.AdminHost = cfg.Host
	}

	if envAdminPort := os.Getenv("ADMIN_PORT"); envAdminPort != "" {
		cfg.AdminPort = envAdminPort
	}
	if cfg.AdminPort == "" {
		cfg.AdminPort = "8081"
	}

	if envEnv := os.Getenv("APP_ENV"); envEnv != "" {
		cfg.Env = envEnv
	}
	if cfg.Env == "" {
		cfg.Env = "development"
	}

	if envTLS := os.Getenv("ENABLE_TLS"); envTLS != "" {
		cfg.EnableTLS = envTLS == "true" || envTLS == "1"
	}

	if envCert := os.Getenv("SSL_CERT_FILE"); envCert != "" {
		cfg.SSLCertFile = envCert
	}
	if cfg.SSLCertFile == "" {
		cfg.SSLCertFile = filepath.Join("data", "certs", "cert.pem")
	}

	if envKey := os.Getenv("SSL_KEY_FILE"); envKey != "" {
		cfg.SSLKeyFile = envKey
	}
	if cfg.SSLKeyFile == "" {
		cfg.SSLKeyFile = filepath.Join("data", "certs", "key.pem")
	}

	if envSSLPort := os.Getenv("SSL_PORT"); envSSLPort != "" {
		cfg.SSLPort = envSSLPort
	}
	if cfg.SSLPort == "" {
		cfg.SSLPort = "8443"
	}

	if envRateLimit := os.Getenv("ENABLE_RATE_LIMIT"); envRateLimit != "" {
		cfg.EnableRateLimit = envRateLimit == "true" || envRateLimit == "1"
	}

	if envRateReq := os.Getenv("RATE_LIMIT_REQUESTS"); envRateReq != "" {
		if v, err := strconv.Atoi(envRateReq); err == nil && v > 0 {
			cfg.RateLimitRequests = v
		}
	}
	if cfg.RateLimitRequests <= 0 {
		cfg.RateLimitRequests = 60
	}

	if envRateWin := os.Getenv("RATE_LIMIT_WINDOW_SECONDS"); envRateWin != "" {
		if v, err := strconv.Atoi(envRateWin); err == nil && v > 0 {
			cfg.RateLimitWindowSeconds = v
		}
	}
	if cfg.RateLimitWindowSeconds <= 0 {
		cfg.RateLimitWindowSeconds = 60
	}

	if envDBPath := os.Getenv("DB_PATH"); envDBPath != "" {
		cfg.DBPath = envDBPath
	}
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join("data", "teachar.db")
	}

	if envLogDir := os.Getenv("LOG_DIR"); envLogDir != "" {
		cfg.LogDir = envLogDir
	}
	if cfg.LogDir == "" {
		cfg.LogDir = "logs"
	}

	if envLogFile := os.Getenv("LOG_FILE"); envLogFile != "" {
		cfg.LogFile = envLogFile
	}
	if cfg.LogFile == "" {
		cfg.LogFile = filepath.Join(cfg.LogDir, "app.log")
	}

	return cfg, nil
}
