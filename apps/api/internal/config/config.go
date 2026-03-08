package config

import "os"

type Config struct {
	AppEnv      string
	DatabaseURL string
	RedisAddr   string
	S3Endpoint  string
	S3Bucket    string
}

func Load() Config {
	return Config{
		AppEnv:      getEnv("APP_ENV", "dev"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://acp:acp@localhost:5432/acp?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		S3Endpoint:  getEnv("S3_ENDPOINT", "localhost:9000"),
		S3Bucket:    getEnv("S3_BUCKET", "acp-artifacts"),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
