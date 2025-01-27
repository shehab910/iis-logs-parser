package processor

import (
	"bufio"
	"context"
	"fmt"
	"iis-logs-parser/parser"
	"iis-logs-parser/utils"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

type Metrics struct {
	mu                 sync.Mutex
	TotalRecords       int64
	SuccessfulWrites   int64
	FailedWrites       int64
	StartTime          time.Time
	BatchCount         int64 // For batch operations
	LastError          error
	TotalParsingTime   time.Duration
	TotalInsertionTime time.Duration
}

func (m *Metrics) AddParsingTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalParsingTime += duration
}

func (m *Metrics) AddInsertionTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalInsertionTime += duration
}

func (m *Metrics) LogMetrics(operation string) {
	duration := time.Since(m.StartTime)
	recordsPerSecond := float64(atomic.LoadInt64(&m.TotalRecords)) / duration.Seconds()

	log.Info().
		Str("operation", operation).
		Int64("total_records", atomic.LoadInt64(&m.TotalRecords)).
		Int64("successful_writes", atomic.LoadInt64(&m.SuccessfulWrites)).
		Int64("failed_writes", atomic.LoadInt64(&m.FailedWrites)).
		Float64("records_per_second", recordsPerSecond).
		Dur("total_duration", duration).
		Int64("batch_count", atomic.LoadInt64(&m.BatchCount)).
		Msg("Operation metrics")
}

func combineBatchInsert(
	wgCombiner *sync.WaitGroup,
	results <-chan *parser.LogEntry,
	dbPool *pgxpool.Pool,
	writer *utils.SyncWriter,
	metrics *Metrics,
) {
	startTime := time.Now()
	defer wgCombiner.Done()
	defer metrics.LogMetrics("batch_insert")

	entriesBatchSize := 10000
	entriesBatch := make([]*parser.LogEntry, 0, entriesBatchSize)

	for entry := range results {
		atomic.AddInt64(&metrics.TotalRecords, 1)

		// Write to file using synchronized writer
		if err := writer.WriteString(entry.String()); err != nil {
			atomic.AddInt64(&metrics.FailedWrites, 1)
			log.Error().Err(err).Msg("Failed to write to output file")
			continue
		}

		entriesBatch = append(entriesBatch, entry)
		if len(entriesBatch) >= entriesBatchSize {
			if err := insertBatch(dbPool, entriesBatch, metrics); err != nil {
				log.Error().Err(err).Msg("Batch insertion failed")
			}
			atomic.AddInt64(&metrics.BatchCount, 1)
			entriesBatch = entriesBatch[:0]
		}
	}

	// Final batch insert
	if len(entriesBatch) > 0 {
		if err := insertBatch(dbPool, entriesBatch, metrics); err != nil {
			log.Error().Err(err).Msg("Final batch insertion failed")
		}
		atomic.AddInt64(&metrics.BatchCount, 1)
	}
	metrics.AddInsertionTime(time.Since(startTime))
}

func insertBatch(dbPool *pgxpool.Pool, batch []*parser.LogEntry, metrics *Metrics) error {
	startTime := time.Now()
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		return err
	}

	count, err := tx.CopyFrom(
		context.Background(),
		pgx.Identifier{"log_entries"},
		[]string{"date", "time", "server_ip", "method", "uri_stem", "uri_query", "port", "username", "client_ip", "user_agent", "status", "sub_status", "win32_status", "time_taken"},
		pgx.CopyFromSlice(len(batch), func(i int) ([]interface{}, error) {
			return []interface{}{
				batch[i].Date,
				batch[i].Time,
				batch[i].ServerIP,
				batch[i].Method,
				batch[i].URIStem,
				batch[i].URIQuery,
				batch[i].Port,
				batch[i].Username,
				batch[i].ClientIP,
				batch[i].UserAgent,
				batch[i].Status,
				batch[i].SubStatus,
				batch[i].Win32Status,
				batch[i].TimeTaken,
			}, nil
		},
		))

	if err != nil {
		atomic.AddInt64(&metrics.FailedWrites, int64(len(batch)))
		metrics.LastError = err
		tx.Rollback(context.Background())
		return err
	}

	if err := tx.Commit(context.Background()); err != nil {
		atomic.AddInt64(&metrics.FailedWrites, int64(len(batch)))
		metrics.LastError = err
	}

	atomic.AddInt64(&metrics.SuccessfulWrites, count)

	log.Debug().
		Int("batch_size", len(batch)).
		Dur("batch insertion duration", time.Since(startTime)).
		Msg("Batch inserted successfully")

	return nil
}

func combineNoDB(wgCombiner *sync.WaitGroup, results <-chan *parser.LogEntry, writer *utils.SyncWriter) {
	defer wgCombiner.Done()

	metrics := &Metrics{StartTime: time.Now()}
	defer metrics.LogMetrics("no_db_processing")

	for entry := range results {
		atomic.AddInt64(&metrics.TotalRecords, 1)

		if err := writer.WriteString(entry.String()); err != nil {
			atomic.AddInt64(&metrics.FailedWrites, 1)
			log.Error().
				Err(err).
				Str("entry", entry.String()).
				Msg("Failed to write to output file")
			continue
		}
		atomic.AddInt64(&metrics.SuccessfulWrites, 1)
	}
}

func combinerBuilder(
	dbInsertionT string,
	wgCombiner *sync.WaitGroup,
	results <-chan *parser.LogEntry,
	dbPool *pgxpool.Pool,
	writer *utils.SyncWriter,
	metrics *Metrics,
) func() {
	switch dbInsertionT {
	case "batch":
		return func() { combineBatchInsert(wgCombiner, results, dbPool, writer, metrics) }
	case "none":
		return func() { combineNoDB(wgCombiner, results, writer) }
	default:
		log.Fatal().Msg("Invalid combiner type, must be one of 'batch', or 'none'")
		return func() {}
	}
}

func ProcessLogFile(filename string, numWorkers int, dbPool *pgxpool.Pool, dbInsertionT string) error {
	metrics := &Metrics{StartTime: time.Now()}

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

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
			startTime := time.Now()
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
			metrics.AddParsingTime(time.Since(startTime))
		}(i)
	}

	outputFile, err := os.Create(filename + "_" + "parsed_logs.txt")
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()
	syncWriter := utils.NewSyncWriter(outputFile)
	defer syncWriter.Flush()

	// Combiner - Fan-in - Merge
	combine := combinerBuilder(dbInsertionT, &wgCombiner, results, dbPool, syncWriter, metrics)
	for i := 0; i < (numWorkers / 2); i++ {
		wgCombiner.Add(1)
		go combine()
	}

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
		return fmt.Errorf("error reading file: %w", err)
	}

	// Wait
	<-done

	duration := time.Since(metrics.StartTime)
	log.Info().
		Int("total_lines", lineCount).
		Float64("duration_seconds", duration.Seconds()).
		Float64("lines_per_second", float64(lineCount)/duration.Seconds()).
		Msg("Finished processing log file")

	return nil
}
