package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	db "iis-logs-parser/database"
	"iis-logs-parser/parser"
	"iis-logs-parser/utils"

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

func combineBatchInsert(wgCombiner *sync.WaitGroup, results <-chan *parser.LogEntry, db *gorm.DB, uniqueCnt *map[string]int64) {
	defer wgCombiner.Done()
	outputFile, err := os.Create("parsed_logs.txt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create output file")
	}

	entriesBatchSize := 1000
	entriesBatch := make([]*parser.LogEntry, 0, entriesBatchSize)

	for entry := range results {
		// log.Debug().
		// 	Str("timestamp", entry.Timestamp).
		// 	Str("client_ip", entry.ClientIP).
		// 	Str("method", entry.Method).
		// 	Str("uri", entry.URIStem).
		// 	Str("status", entry.StatusCode).
		// 	Msg("Processed log entry")
		(*uniqueCnt)[entry.Status]++
		_, err := fmt.Fprintf(outputFile, "%+v\n", *entry)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to write entry to output file\nEntry: \n%+v", *entry)
		}

		entriesBatch = append(entriesBatch, entry)

		if len(entriesBatch) >= entriesBatchSize {

			res := db.Create(&entriesBatch)

			if res.Error != nil {
				log.Error().Err(res.Error).Msgf("Failed to save log batch to database\nEntries: \n%+v", *entry)
			}
			entriesBatch = entriesBatch[:0]
		}
	}
	if len(entriesBatch) >= 0 {

		res := db.Create(&entriesBatch)

		if res.Error != nil {
			log.Error().Err(res.Error).Msgf("Failed to save log batch to database\nEntries: \n%+v", entriesBatch)

		}
		entriesBatch = entriesBatch[:0]
	}
}

func combineSequentialInsert(wgCombiner *sync.WaitGroup, results <-chan *parser.LogEntry, db *gorm.DB, uniqueCnt *map[string]int64) {
	defer wgCombiner.Done()
	outputFile, err := os.Create("parsed_logs.txt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create output file")
	}
	for entry := range results {
		(*uniqueCnt)[entry.Status]++
		_, err := fmt.Fprintf(outputFile, "%+v\n", *entry)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to write entry to output file\nEntry: \n%+v", *entry)
		}

		res := db.Create(entry)

		if res.Error != nil {
			log.Error().Err(res.Error).Msgf("Failed to save entry to database\nEntries: \n%+v", *entry)
		}
	}
}

func combineNoDB(wgCombiner *sync.WaitGroup, results <-chan *parser.LogEntry, uniqueCnt *map[string]int64) {
	defer wgCombiner.Done()
	outputFile, err := os.Create("parsed_logs.txt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create output file")
	}
	for entry := range results {
		(*uniqueCnt)[entry.Status]++
		_, err := fmt.Fprintf(outputFile, "%+v\n", *entry)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to write entry to output file\nEntry: \n%+v", *entry)
		}
	}
}

func combinerBuilder(dbInsertionT string, wgCombiner *sync.WaitGroup, results <-chan *parser.LogEntry, db *gorm.DB, uniqueCnt *map[string]int64) func() {
	switch dbInsertionT {
	case "batch":
		return func() { combineBatchInsert(wgCombiner, results, db, uniqueCnt) }
	case "sequential":
		return func() { combineSequentialInsert(wgCombiner, results, db, uniqueCnt) }
	case "none":
		return func() { combineNoDB(wgCombiner, results, uniqueCnt) }
	default:
		log.Fatal().Msg("Invalid combiner type, must be one of 'batch', 'sequential', or 'none'")
		return func() {}
	}
}

// PGPASSWORD=password psql -U postgres -h localhost -d postgres-dev -c "SELECT COUNT(*) FROM log_entries;"
// PGPASSWORD=password psql -U postgres -h localhost -d postgres-dev -c "drop table if exists log_entries;"
func processLogFile(filename string, numWorkers int, db *gorm.DB, dbInsertionT string) (*map[string]int64, error) {
	startTime := time.Now()

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	uniqueCnt := make(map[string]int64)

	lines := make(chan string)
	results := make(chan *parser.LogEntry)
	errorsChan := make(chan error)
	done := make(chan bool)

	var wgWorkers sync.WaitGroup
	var wgCombiner sync.WaitGroup

	// Workers - Fan-out
	for i := 0; i < numWorkers; i++ {
		wgWorkers.Add(1)
		go func(id int) {
			defer wgWorkers.Done()
			for line := range lines {
				entry, err := parser.ParseLogLine(line)
				if err != nil {
					errorsChan <- err
					continue
				}
				if entry != nil {
					results <- entry
				}
			}
		}(i)
	}

	// Combiner - Fan-in - Merge
	wgCombiner.Add(1)
	combine := combinerBuilder(dbInsertionT, &wgCombiner, results, db, &uniqueCnt)
	go combine()

	go func() {
		for err := range errorsChan {
			if parseErr, ok := err.(*parser.ParseError); ok {
				log.Warn().
					Str("error", parseErr.Message).
					Str("line", parseErr.Line).
					Msg("Failed to parse log line")
			} else {
				log.Error().Err(err).Msg("Unexpected error during parsing")
			}
		}
	}()

	go func() {
		wgWorkers.Wait()
		close(results)
		close(errorsChan)
		wgCombiner.Wait()
		done <- true
	}()

	// Read and distribute lines to workers
	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lines <- scanner.Text()
		lineCount++
	}
	close(lines)

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Wait
	<-done

	duration := time.Since(startTime)
	log.Info().
		Int("total_lines", lineCount).
		Float64("duration_seconds", duration.Seconds()).
		Float64("lines_per_second", float64(lineCount)/duration.Seconds()).
		Msg("Finished processing log file")

	return &uniqueCnt, nil
}

func main() {
	log.Info().Msg("Starting IIS log parser")

	if len(os.Args) != 2 {
		log.Fatal().Msg("Usage: ./iis-parser <logfile>")
	}

	filename := os.Args[1]
	numWorkers := 4

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

	res, err := processLogFile(filename, numWorkers, db, "batch")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to process log file")
	}

	str, err := utils.MapToTableLogMsg(res)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert map to table")
	}
	log.Info().Msg(str)
}
