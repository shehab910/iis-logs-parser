package main

import (
	"context"
	"errors"
	db "iis-logs-parser/database"
	"iis-logs-parser/processor"
	"os"

	pgxZerolog "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	log.Info().Msgf("Connecting to database: %s", dbConfig.NoPassDSN())
	pgxConfig, err := pgxpool.ParseConfig(dbConfig.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse connection string")
	}

	logger := pgxZerolog.NewLogger(log.Logger)

	pgxConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	dbPool, err := pgxpool.NewWithConfig(context.Background(), pgxConfig)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			log.Fatal().Err(err).Msg("Unable to create connection pool\n")
		}
	}
	defer dbPool.Close()

	log.Info().Msg("Connected to database")

	dbPool.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS log_entries (ID INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,date DATE, time TIME, server_ip TEXT, method TEXT, uri_stem TEXT, uri_query TEXT, port TEXT, username TEXT, client_ip TEXT, user_agent TEXT, status TEXT, sub_status TEXT, win32_status TEXT, time_taken TEXT)")

	err = processor.ProcessLogFile(filename, numWorkers, dbPool, "batch")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to process log file")
	}
}
