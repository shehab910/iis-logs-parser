package main

import (
	db "iis-logs-parser/database"
	"iis-logs-parser/parser"
	"iis-logs-parser/processor"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormWriter implements the gormlogger.GormWriter interface for zerolog
type GormWriter struct {
	Logger *zerolog.Logger
}

func (w GormWriter) Printf(format string, args ...interface{}) {
	w.Logger.Debug().Msgf(format, args...)
}

func init() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open log file")
	}
	log.Logger = log.Output(logFile)
	// log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	log.Info().Msg("Starting IIS log parser")

	if len(os.Args) != 2 {
		log.Fatal().Msg("Usage: ./iis-parser <logfile>")
	}

	filename := os.Args[1]
	numWorkers := 12

	dbConfig, err := db.LoadConfigFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load database config")
	}

	dsn := dbConfig.DSN()
	log.Info().Msgf("Connecting to database: %s", dbConfig.NoPassDSN())

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			// Redirect GORM logs to zerolog
			GormWriter{Logger: &log.Logger},
			logger.Config{
				SlowThreshold: 1 * time.Second, // Set threshold to 1 second
				LogLevel:      logger.Warn,
				Colorful:      false,
			},
		),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	db.AutoMigrate(&parser.LogEntry{})

	log.Info().Msg("Connected to database")

	err = processor.ProcessLogFile(filename, numWorkers, db, "batch")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to process log file")
	}
}
