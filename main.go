package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	"iis-logs-parser/parser"
	"iis-logs-parser/utils"

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
}

func processLogFile(filename string, numWorkers int) (*map[string]int64, error) {
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

	// Workers - Fan-out
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
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
	go func() {
		outputFile, err := os.Create("parsed_logs.txt")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create output file")
		}

		for entry := range results {
			// log.Debug().
			// 	Str("timestamp", entry.Timestamp).
			// 	Str("client_ip", entry.ClientIP).
			// 	Str("method", entry.Method).
			// 	Str("uri", entry.URIStem).
			// 	Str("status", entry.StatusCode).
			// 	Msg("Processed log entry")
			uniqueCnt[entry.Status]++
			fmt.Fprintf(outputFile, "%+v\n", *entry)
		}

	}()

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
		wg.Wait()
		close(results)
		close(errorsChan)
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

	res, err := processLogFile(filename, numWorkers)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to process log file")
	}

	str, err := utils.MapToTableLogMsg(res)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert map to table")
	}
	log.Info().Msg(str)
}
