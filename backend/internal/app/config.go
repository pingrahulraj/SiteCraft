package app

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	AppName         string
	APIPort         string
	DeliveryPort    string
	DeliveryBaseURL string
	FrontendOrigin  string
	StorageRoot     string
	JWTSecret       string
	DatabaseURL     string
	DBHost          string
	DBPort          string
	DBName          string
	DBUser          string
	DBPassword      string
	DBAdminDB       string
	DBSSLMode       string
}

func LoadConfig() Config {
	return Config{
		AppName:         env("APP_NAME", "WebsiteBuilder"),
		APIPort:         env("API_PORT", "8080"),
		DeliveryPort:    env("DELIVERY_PORT", "8083"),
		DeliveryBaseURL: env("DELIVERY_BASE_URL", ""),
		FrontendOrigin:  env("CORS_ALLOWED_ORIGINS", "http://localhost:5137"),
		StorageRoot:     env("STORAGE_ROOT", "./storage"),
		JWTSecret:       env("JWT_SECRET", "change-me"),
		DatabaseURL:     env("DATABASE_URL", ""),
		DBHost:          env("DB_HOST", "127.0.0.1"),
		DBPort:          env("DB_PORT", "5432"),
		DBName:          env("DB_NAME", "websitebuilder"),
		DBUser:          env("DB_USER", "postgres"),
		DBPassword:      env("DB_PASSWORD", ""),
		DBAdminDB:       env("DB_ADMIN_DB", "postgres"),
		DBSSLMode:       env("DB_SSLMODE", env("PGSSLMODE", "disable")),
	}
}

func (c Config) deliveryURL() string {
	if c.DeliveryBaseURL != "" {
		return strings.TrimRight(c.DeliveryBaseURL, "/")
	}
	return "http://localhost:" + c.DeliveryPort
}

func (c Config) appDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return c.buildDSN(c.DBName)
}

func (c Config) adminDSN() string {
	if c.DatabaseURL != "" {
		parsed, err := url.Parse(c.DatabaseURL)
		if err == nil {
			parsed.Path = "/" + c.DBAdminDB
			return parsed.String()
		}
	}
	return c.buildDSN(c.DBAdminDB)
}

func (c Config) buildDSN(dbName string) string {
	values := url.Values{}
	values.Set("sslmode", c.DBSSLMode)
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?%s",
		url.QueryEscape(c.DBUser),
		url.QueryEscape(c.DBPassword),
		c.DBHost,
		c.DBPort,
		dbName,
		values.Encode(),
	)
}

func openPostgres(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.appDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
