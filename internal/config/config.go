package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Environment string         `env:"APP_ENV"     envDefault:"development"`
	Server      ServerConfig   `envPrefix:"SERVER_"`
	Database    DatabaseConfig `envPrefix:"DATABASE_"`
	Auth        AuthConfig     `envPrefix:"AUTH_"`
	Logging     LoggingConfig  `envPrefix:"LOGGING_"`
	CORS        CORSConfig     `envPrefix:"CORS_"`
}

type ServerConfig struct {
	Port         int           `env:"PORT"          envDefault:"8070"`
	Host         string        `env:"HOST"          envDefault:"0.0.0.0"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT"  envDefault:"15s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"15s"`
	SwaggerHost  string        `env:"SWAGGER_HOST"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `envPrefix:"POSTGRES_"`
}

type PostgresConfig struct {
	Host         string `env:"HOST"           envDefault:"localhost"`
	Port         int    `env:"PORT"           envDefault:"5432"`
	Database     string `env:"DATABASE"       envDefault:"edugo"`
	User         string `env:"USER"           envDefault:"edugo"`
	Password     string `env:"PASSWORD,required"`
	MaxOpenConns int    `env:"MAX_OPEN_CONNS" envDefault:"25"`
	MaxIdleConns int    `env:"MAX_IDLE_CONNS" envDefault:"10"`
	SSLMode      string `env:"SSL_MODE"       envDefault:"disable"`
}

type LoggingConfig struct {
	Level  string `env:"LEVEL"  envDefault:"info"`
	Format string `env:"FORMAT" envDefault:"json"`
}

type AuthConfig struct {
	JWT JWTConfig `envPrefix:"JWT_"`
}

type JWTConfig struct {
	Secret               string        `env:"SECRET,required"`
	Issuer               string        `env:"ISSUER"                 envDefault:"edugo-central"`
	AccessTokenDuration  time.Duration `env:"ACCESS_TOKEN_DURATION"  envDefault:"15m"`
	RefreshTokenDuration time.Duration `env:"REFRESH_TOKEN_DURATION" envDefault:"168h"`
}

type CORSConfig struct {
	AllowedOrigins string `env:"ALLOWED_ORIGINS" envDefault:"*"`
	AllowedMethods string `env:"ALLOWED_METHODS" envDefault:"GET,POST,PUT,PATCH,DELETE,OPTIONS"`
	AllowedHeaders string `env:"ALLOWED_HEADERS" envDefault:"Origin,Content-Type,Accept,Authorization,X-Request-ID"`
}

func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("error parsing config from environment: %w", err)
	}
	return &cfg, nil
}
