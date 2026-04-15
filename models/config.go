package models

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config menyimpan seluruh konfigurasi aplikasi yang dimuat dari .env.
type Config struct {
	AppName  string
	Port     string
	APIPrefix string
	Debug    bool

	// Auth
	JWTSecret     string
	JWTExpireHour int

	// Databases — kosongkan jika tidak dipakai
	SQLiteDSN   string
	PostgresDSN string
	MsSQLDSN    string

	// Logging
	LogDir        string
	LogRetainDays int
}

// LoadConfig memuat konfigurasi dari file .env (default: ".env").
// Variabel yang sudah ada di environment OS lebih diprioritaskan.
func LoadConfig(envFiles ...string) *Config {
	if len(envFiles) == 0 {
		envFiles = []string{".env"}
	}
	// Muat .env ke environment, tidak menimpa variabel OS yang sudah ada
	_ = godotenv.Load(envFiles...)

	return &Config{
		AppName:   getEnv("APP_NAME", "bifeldy-app"),
		Port:      getEnv("PORT", "8080"),
		APIPrefix: getEnv("API_PREFIX", "api"),
		Debug:     getEnvBool("DEBUG", false),

		JWTSecret:     getEnv("JWT_SECRET", "ganti-secret-ini"),
		JWTExpireHour: getEnvInt("JWT_EXPIRE_HOUR", 24),

		SQLiteDSN:   getEnv("DB_SQLITE", ""),
		PostgresDSN: getEnv("DB_POSTGRES", ""),
		MsSQLDSN:    getEnv("DB_MSSQL", ""),

		LogDir:        getEnv("LOG_DIR", "_data/logs"),
		LogRetainDays: getEnvInt("LOG_RETAIN_DAYS", 30),
	}
}

func getEnv(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
