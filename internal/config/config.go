package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config represents the application configuration
type Config struct {
	Database DatabaseConfig
	Logger   LoggerConfig
	Server   ServerConfig
	Auth     AuthConfig
}

// DatabaseConfig represents the database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// LoggerConfig represents the logger configuration
type LoggerConfig struct {
	Level  string
	Format string // "json" or "text"
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Port int
	Host string
}

// AuthConfig represents API authentication tokens for route groups.
type AuthConfig struct {
	AdminToken  string
	PublicToken string
}

// Load loads the application configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "inventory_reservations"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Server: ServerConfig{
			Port: getEnvInt("SERVER_PORT", 8080),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		Auth: AuthConfig{
			AdminToken:  getEnv("ADMIN_API_TOKEN", ""),
			PublicToken: getEnv("PUBLIC_API_TOKEN", ""),
		},
	}

	return config, nil
}

// getEnv retrieves the value of an environment variable or returns a default value if it's not set'
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves the value of an environment variable as an integer or returns a default value if it's not set'
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetDatabaseDSN returns the database connection DSN string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.User,
		c.Database.Password, c.Database.DBName, c.Database.SSLMode)
}
