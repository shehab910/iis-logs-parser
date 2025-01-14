package main

import (
	"bufio"
	"fmt"
	tableStr "iis-logs-parser/table_string"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	FIELDS_DEF = "#Fields: date time s-ip cs-method cs-uri-stem cs-uri-query s-port cs-username c-ip cs(User-Agent) sc-status sc-substatus sc-win32-status time-taken"
)

var FIELDS_LEN = len(strings.Split(strings.Replace(FIELDS_DEF, "#Fields: ", "", 1), " "))

type LogEntry struct {
	Date        string
	Time        string
	ServerIP    string
	Method      string
	URIStem     string
	URIQuery    string
	Port        string
	Username    string
	ClientIP    string
	UserAgent   string
	Status      string
	SubStatus   string
	Win32Status string
	TimeTaken   string
}

type Counter struct {
	count int64
}

func (c *Counter) Increment(by int64) {
	atomic.AddInt64(&c.count, by)
}

func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.count)
}

type ParseError struct {
	Line    string
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("Parse error: %s in line: %s", e.Message, e.Line)
}

func init() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
}

func parseLogLine(line string) (*LogEntry, error) {
	if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
		if strings.HasPrefix(line, "#Fields:") {
			if line != FIELDS_DEF {
				log.Fatal().Msgf("Incorrect fields format\nMust be: %v", FIELDS_DEF)
			}
		}
		return nil, nil
	}

	fields := strings.Split(line, " ")

	if len(fields) < FIELDS_LEN {
		return nil, &ParseError{
			Line:    line,
			Message: fmt.Sprintf("Expected %d fields, got %d", FIELDS_LEN, len(fields)),
		}
	}

	entry := &LogEntry{
		Date:        fields[0],
		Time:        fields[1],
		ServerIP:    fields[2],
		Method:      fields[3],
		URIStem:     fields[4],
		URIQuery:    fields[5],
		Port:        fields[6],
		Username:    fields[7],
		ClientIP:    fields[8],
		UserAgent:   fields[9],
		Status:      fields[10],
		SubStatus:   fields[11],
		Win32Status: fields[12],
		TimeTaken:   fields[13],
	}

	return entry, nil
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
	results := make(chan *LogEntry)
	errorsChan := make(chan error)
	done := make(chan bool)

	// Workers - Fan-out
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for line := range lines {
				entry, err := parseLogLine(line)
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
		for entry := range results {
			// log.Debug().
			// 	Str("timestamp", entry.Timestamp).
			// 	Str("client_ip", entry.ClientIP).
			// 	Str("method", entry.Method).
			// 	Str("uri", entry.URIStem).
			// 	Str("status", entry.StatusCode).
			// 	Msg("Processed log entry")
			uniqueCnt[entry.Status]++
		}

	}()

	go func() {
		for err := range errorsChan {
			if parseErr, ok := err.(*ParseError); ok {
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
	log.Info().MsgFunc(func() string {
		return MapLogMsg(res)
	})
}

func SPrintMap(mp *map[string]int64) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	for k, v := range *mp {
		sb.WriteString(fmt.Sprintf("%v: %v\n", k, v))
	}
	return sb.String()
}

func MapLogMsg(mp *map[string]int64) string {
	rows := [][]string{}
	for k, v := range *mp {
		rows = append(rows, []string{k, fmt.Sprintf("%v", v)})
	}

	t := tableStr.New()
	t.SetHeaders([]string{"Status Code", "Number of Occurrences"})
	t.SetRows(rows)

	resStr, err := t.String()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to generate table")
	}
	return resStr
}
