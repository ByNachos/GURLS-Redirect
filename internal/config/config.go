package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Config holds all the configuration for the application.
type Config struct {
	Env        string `yaml:"env" env:"ENV" env-default:"production"`
	HTTPServer `yaml:"http_server"`
	GRPCClient `yaml:"grpc_client"`
}

// HTTPServer holds HTTP server specific configuration.
type HTTPServer struct {
	Address     string        `yaml:"address" env:"HTTP_SERVER_ADDRESS" env-default:"0.0.0.0:8080"`
	BaseURL     string        `yaml:"base_url" env:"BASE_URL" env-default:"http://localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env:"HTTP_SERVER_TIMEOUT" env-default:"5s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env:"HTTP_SERVER_IDLE_TIMEOUT" env-default:"60s"`
}

// GRPCClient holds gRPC client specific configuration.
type GRPCClient struct {
	BackendAddress string        `yaml:"backend_address" env:"GRPC_BACKEND_ADDRESS" env-default:"localhost:50051"`
	Timeout        time.Duration `yaml:"timeout" env:"GRPC_CLIENT_TIMEOUT" env-default:"5s"`
}

// MustLoad loads the application configuration.
func MustLoad() *Config {
	// Try to load .env file (ignore error in production)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	var cfg Config

	// Check if config file path is specified
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/local.yml" // default path
	}

	// Try to load config file
	if _, err := os.Stat(configPath); err == nil {
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			log.Fatalf("cannot read config: %s", err)
		}
	} else {
		// If config file doesn't exist, use environment variables only
		log.Println("Config file not found, using environment variables only")
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			log.Fatalf("cannot read config from environment: %s", err)
		}
	}

	return &cfg
}