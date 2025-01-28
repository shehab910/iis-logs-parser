package main

import (
	"context"
	"errors"
	db "iis-logs-parser/database"
	"iis-logs-parser/models"
	"iis-logs-parser/processor"
	"iis-logs-parser/routes"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	pgxZerolog "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5"
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

func processLogsCron(ctx context.Context) {
	dbConfig, err := db.LoadConfigFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load database config")
	}

	log.Info().Msgf("DB-PGX: Connecting to database: %s", dbConfig.NoPassDSN())
	pgxConfig, err := pgxpool.ParseConfig(dbConfig.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("DB-PGX: Failed to connect to database")
	}

	logger := pgxZerolog.NewLogger(log.Logger)

	pgxConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			log.Fatal().Err(err).Msg("DB-PGX: Unable to create connection pool")
		}
	}
	defer dbPool.Close()
	log.Info().Msg("DB-PGX: Connected to database")

	log.Info().Msg("Searching for files to parse")
	var pendingFilesCount int
	err = dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM log_files WHERE status = 'pending'").Scan(&pendingFilesCount)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get pending files count")
	}
	if pendingFilesCount == 0 {
		log.Info().Msg("No files to process")
		return
	}
	log.Info().Msgf("Found %d files to process", pendingFilesCount)

	rows, err := dbPool.Query(ctx, "SELECT id, name FROM log_files WHERE status = 'pending'")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to query log files")
	}

	var fileId uint
	var fileName string
	pgx.ForEachRow(
		rows,
		[]any{&fileId, &fileName},
		func() error {
			log.Info().Msgf("Starting to process file: %s with id: %d", fileName, fileId)

			_, err := dbPool.Exec(ctx, "UPDATE log_files SET status = $1 WHERE id = $2", models.StatusProcessing, fileId)
			if err != nil {
				log.Err(err).Msgf("Failed to update file status to processing: %s with id: %d", fileName, fileId)
				log.Info().Msgf("Skipping file: %s with id: %d", fileName, fileId)
				return nil
			}

			savedFileName := filepath.Join("uploaded_logs", fileName+"-"+strconv.FormatUint(uint64(fileId), 10))
			err = processor.ProcessLogFile(savedFileName, 12, dbPool, "batch")
			if err != nil {
				log.Err(err).Msgf("Failed to process file: %s with id: %d", fileName, fileId)
				_, err := dbPool.Exec(ctx, "UPDATE log_files SET status = $1 WHERE id = $2", models.StatusFailed, fileId)
				if err != nil {
					log.Err(err).Msgf("Failed to update file status to failed: %s with id: %d", fileName, fileId)
				}
				return nil
			}
			log.Info().Msgf("Finished processing file: %s with id: %d", fileName, fileId)
			_, err = dbPool.Exec(ctx, "UPDATE log_files SET status = $1 WHERE id = $2", models.StatusCompleted, fileId)
			if err != nil {
				log.Err(err).Msgf("Failed to update file status to completed: %s with id: %d", fileName, fileId)
			}
			return nil
		},
	)
}

func main() {
	if _, err := os.Stat("uploaded_logs"); os.IsNotExist(err) {
		os.Mkdir("uploaded_logs", 0755)
	}

	db.InitGormDB()

	var scheduleNext func() // Declare a function variable for self-referencing

	task := func(ctx context.Context) {
		processLogsCron(ctx)
		time.AfterFunc(20*time.Second, scheduleNext)
	}

	// Define scheduleNext to trigger the task in a goroutine
	scheduleNext = func() {
		go task(context.Background())
	}

	scheduleNext()

	r := gin.Default()
	routes.RegisterRoutes(r)
	r.Run()
}
