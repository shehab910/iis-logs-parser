package processor

import (
	"bufio"
	"fmt"
	"iis-logs-parser/parser"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

type Metrics struct {
	TotalRecords     int64
	SuccessfulWrites int64
	FailedWrites     int64
	StartTime        time.Time
	BatchCount       int64 // For batch operations
	LastError        error
}

func (m *Metrics) LogMetrics(operation string) {
	duration := time.Since(m.StartTime)
	recordsPerSecond := float64(m.TotalRecords) / duration.Seconds()

	log.Info().
		Str("operation", operation).
		Int64("total_records", m.TotalRecords).
		Int64("successful_writes", m.SuccessfulWrites).
		Int64("failed_writes", m.FailedWrites).
		Float64("records_per_second", recordsPerSecond).
		Dur("total_duration", duration).
		Int64("batch_count", m.BatchCount).
		Msg("Operation metrics")
}

func combineBatchInsert(wgCombiner *sync.WaitGroup, results <-chan *parser.LogEntry, db *gorm.DB, uniqueCnt *map[string]int64) {
	defer wgCombiner.Done()

	metrics := &Metrics{StartTime: time.Now()}
	defer metrics.LogMetrics("batch_insert")

	outputFile, err := os.Create("parsed_logs.txt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create output file")
	}
	defer outputFile.Close()

	entriesBatchSize := 3000
	entriesBatch := make([]*parser.LogEntry, 0, entriesBatchSize)

	for entry := range results {
		metrics.TotalRecords++
		(*uniqueCnt)[entry.Status]++

		if _, err := fmt.Fprint(outputFile, entry.String()); err != nil {
			metrics.FailedWrites++
			log.Error().
				Err(err).
				Str("entry", entry.String()).
				Msg("Failed to write to output file")
			continue
		}

		entriesBatch = append(entriesBatch, entry)

		if len(entriesBatch) >= entriesBatchSize {
			if err := insertBatch(db, entriesBatch, metrics); err != nil {
				log.Error().
					Err(err).
					Int("batch_size", len(entriesBatch)).
					Msg("Batch insertion failed")
			}
			metrics.BatchCount++
			entriesBatch = entriesBatch[:0]
		}
	}

	// Handle remaining entries
	if len(entriesBatch) > 0 {
		if err := insertBatch(db, entriesBatch, metrics); err != nil {
			log.Error().
				Err(err).
				Int("batch_size", len(entriesBatch)).
				Msg("Final batch insertion failed")
		}
		metrics.BatchCount++
	}
}

func insertBatch(db *gorm.DB, batch []*parser.LogEntry, metrics *Metrics) error {
	startTime := time.Now()

	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Create(&batch).Error; err != nil {
		tx.Rollback()
		metrics.FailedWrites += int64(len(batch))
		metrics.LastError = err
		return err
	}

	if err := tx.Commit().Error; err != nil {
		metrics.FailedWrites += int64(len(batch))
		metrics.LastError = err
		return err
	}

	metrics.SuccessfulWrites += int64(len(batch))

	log.Debug().
		Int("batch_size", len(batch)).
		Dur("duration", time.Since(startTime)).
		Msg("Batch inserted successfully")

	return nil
}

func combineSequentialInsert(wgCombiner *sync.WaitGroup, results <-chan *parser.LogEntry, db *gorm.DB, uniqueCnt *map[string]int64) {
	defer wgCombiner.Done()

	metrics := &Metrics{StartTime: time.Now()}
	defer metrics.LogMetrics("sequential_insert")

	outputFile, err := os.Create("parsed_logs.txt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create output file")
	}
	defer outputFile.Close()

	for entry := range results {
		metrics.TotalRecords++
		(*uniqueCnt)[entry.Status]++

		startTime := time.Now()

		if _, err := fmt.Fprint(outputFile, entry.String()); err != nil {
			metrics.FailedWrites++
			log.Error().
				Err(err).
				Str("entry", entry.String()).
				Msg("Failed to write to output file")
			continue
		}

		if err := db.Create(entry).Error; err != nil {
			metrics.FailedWrites++
			metrics.LastError = err
			log.Error().
				Err(err).
				Interface("entry", entry).
				Msg("Failed to insert entry")
			continue
		}

		metrics.SuccessfulWrites++

		log.Debug().
			Dur("duration", time.Since(startTime)).
			Msg("Entry inserted successfully")
	}
}

func combineNoDB(wgCombiner *sync.WaitGroup, results <-chan *parser.LogEntry, uniqueCnt *map[string]int64) {
	defer wgCombiner.Done()

	metrics := &Metrics{StartTime: time.Now()}
	defer metrics.LogMetrics("no_db_processing")

	outputFile, err := os.Create("parsed_logs.txt")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create output file")
	}
	defer outputFile.Close()

	for entry := range results {
		metrics.TotalRecords++
		(*uniqueCnt)[entry.Status]++

		if _, err := fmt.Fprint(outputFile, entry.String()); err != nil {
			metrics.FailedWrites++
			log.Error().
				Err(err).
				Str("entry", entry.String()).
				Msg("Failed to write to output file")
			continue
		}
		metrics.SuccessfulWrites++
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
func ProcessLogFile(filename string, numWorkers int, db *gorm.DB, dbInsertionT string) (*map[string]int64, error) {
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
