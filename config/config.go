package config

import "os"

// Config holds all application configuration.
type Config struct {
	// Primary PostgreSQL (writes)
	PrimaryDBHost     string
	PrimaryDBPort     string
	PrimaryDBUser     string
	PrimaryDBPassword string
	PrimaryDBName     string

	// Replica PostgreSQL (reads)
	ReplicaDBHost string
	ReplicaDBPort string

	// Redis
	RedisAddr string

	// Server
	ServerPort string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		PrimaryDBHost:     getEnv("PRIMARY_DB_HOST", "localhost"),
		PrimaryDBPort:     getEnv("PRIMARY_DB_PORT", "5432"),
		PrimaryDBUser:     getEnv("DB_USER", "wallet_user"),
		PrimaryDBPassword: getEnv("DB_PASSWORD", "wallet_secret"),
		PrimaryDBName:     getEnv("DB_NAME", "wallet_db"),

		ReplicaDBHost: getEnv("REPLICA_DB_HOST", "localhost"),
		ReplicaDBPort: getEnv("REPLICA_DB_PORT", "5433"),

		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),

		ServerPort: getEnv("SERVER_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
