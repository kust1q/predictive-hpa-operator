package config

import (
	"os"
	"strconv"
)

type Config struct {
	MetricsAddr          string
	ProbeAddr            string
	EnableLeaderElection bool
}

var cfg *Config

func InitConfig() error {
	cfg = &Config{
		MetricsAddr:          getEnv("METRICS_ADDR", ":8080"),
		ProbeAddr:            getEnv("PROBE_ADDR", ":8081"),
		EnableLeaderElection: getEnvBool("ENABLE_LEADER_ELECTION", false),
	}
	return nil
}

func Get() *Config {
	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fallback
		}
		return b
	}
	return fallback
}
