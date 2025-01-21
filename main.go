package main

import (
	db "iis-logs-parser/database"
	"iis-logs-parser/parser"
	"iis-logs-parser/processor"
	"iis-logs-parser/utils"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	db.AutoMigrate(&parser.LogEntry{})

	log.Info().Msg("Connected to database")

	res, err := processor.ProcessLogFile(filename, numWorkers, db, "batch")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to process log file")
	}

	str, err := utils.MapToTableLogMsg(res)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert map to table")
	}
	log.Info().Msg(str)
}
