package db

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func LoadConfigFromEnv() (*DBConfig, error) {

	if os.Getenv("GO_ENV") != "production" {
		log.Info().Msg("loading .env file")
		err := godotenv.Load(".env.local")
		if err != nil {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	cfg := &DBConfig{}

	vars := map[string]*string{
		"DB_USER": &cfg.User,
		"DB_PASS": &cfg.Password,
		"DB_HOST": &cfg.Host,
		"DB_PORT": &cfg.Port,
		"DB_NAME": &cfg.DBName,
	}

	var missingVars []string
	for env, ptr := range vars {
		if value, found := os.LookupEnv(env); found {
			*ptr = value
		} else {
			missingVars = append(missingVars, env)
		}
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	return cfg, nil
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		url.QueryEscape(c.User),
		url.QueryEscape(c.Password),
		url.QueryEscape(c.Host),
		url.QueryEscape(c.Port),
		url.QueryEscape(c.DBName),
	)
}

func (c *DBConfig) NoPassDSN() string {
	return strings.Replace(c.DSN(), url.QueryEscape(c.Password), "****", 1)
}
