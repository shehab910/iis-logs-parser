package db

import (
	"errors"
	"fmt"
	gormzerolog "iis-logs-parser/gorm-zerolog"
	"iis-logs-parser/models"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var GormDB *gorm.DB

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

func InitGormDB() {
	dbConfig, err := LoadConfigFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load database config")
	}

	dsn := dbConfig.DSN()
	log.Info().Msgf("DB-GORM: Connecting to database: %s", dbConfig.NoPassDSN())
	GormDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormzerolog.Logger{},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("DB-GORM: Failed to connect to database")
	}
	log.Info().Msg("DB-GORM: Connected to database")

	err = errors.Join(
		GormDB.AutoMigrate(&models.LogEntry{}),
		GormDB.AutoMigrate(&models.LogFile{}),
		GormDB.AutoMigrate(&models.User{}),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("DB-GORM: Failed to migrate database")
	}
}
