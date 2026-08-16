package config

import "os"

// Config holds the application configuration.
type Config struct {
	AppName     string
	Host        string
	Port        string
	Env         string
	EnableTLS   bool
	SSLCertFile string
	SSLKeyFile  string
	SSLPort     string
}

// New creates a new Config instance from environment variables with defaults.
func New() (*Config, error) {
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "teachar.in"
	}
	host := os.Getenv("APP_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	enableTLS := os.Getenv("ENABLE_TLS") == "true" || os.Getenv("ENABLE_TLS") == "1"
	sslCertFile := os.Getenv("SSL_CERT_FILE")
	if sslCertFile == "" {
		sslCertFile = "data/certs/cert.pem"
	}
	sslKeyFile := os.Getenv("SSL_KEY_FILE")
	if sslKeyFile == "" {
		sslKeyFile = "data/certs/key.pem"
	}
	sslPort := os.Getenv("SSL_PORT")
	if sslPort == "" {
		sslPort = "8443"
	}

	return &Config{
		AppName:     appName,
		Host:        host,
		Port:        port,
		Env:         env,
		EnableTLS:   enableTLS,
		SSLCertFile: sslCertFile,
		SSLKeyFile:  sslKeyFile,
		SSLPort:     sslPort,
	}, nil
}
