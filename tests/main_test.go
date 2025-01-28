package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	db "iis-logs-parser/database"
	"iis-logs-parser/models"
	"iis-logs-parser/parser"
	"iis-logs-parser/processor"
	"iis-logs-parser/utils"

	pgxZerolog "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// Writer implements the gormlogger.Writer interface for zerolog
type Writer struct {
	Logger *zerolog.Logger
}

func (w Writer) Printf(format string, args ...interface{}) {
	w.Logger.Debug().Msgf(format, args...)
}

// setupTestDB creates a test database connection and returns cleanup function
func setupTestDB() (*pgxpool.Pool, func()) {

	dbConfig := &db.DBConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "password",
		DBName:   "postgres-dev",
	}

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

	cleanup := func() {
		dbPool.Exec(context.Background(), "DROP TABLE IF EXISTS log_entries")
	}
	cleanup()

	dbPool.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS log_entries (ID INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,date DATE, time TIME, server_ip TEXT, method TEXT, uri_stem TEXT, uri_query TEXT, port TEXT, username TEXT, client_ip TEXT, user_agent TEXT, status TEXT, sub_status TEXT, win32_status TEXT, time_taken TEXT)")

	return dbPool, cleanup
}

// Helper function to create test log file
func createTestLogFile(t testing.TB, testCase CaseType) (string, []*models.LogEntry, func()) {
	content := Cases[testCase].input()
	expected := Cases[testCase].expected().([]*models.LogEntry)

	tmpfile, err := os.CreateTemp("", "logfile")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name(), expected, func() {
		os.Remove(tmpfile.Name())
	}
}

func testProcessLogFileBase(t *testing.T, db *pgxpool.Pool, dbInsertionT string) []*models.LogEntry {
	fileName, expected, cleanup := createTestLogFile(t, ParseCorrectLines)
	defer cleanup()

	err := processor.ProcessLogFile(fileName, 2, db, dbInsertionT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsedLogsFilename := fileName + "_" + "parsed_logs.txt"
	parsedLogsFile, err := os.Open(parsedLogsFilename)
	if err != nil {
		t.Fatalf("failed to open parsed logs file: %v", err)
	}
	defer parsedLogsFile.Close()
	defer os.Remove(parsedLogsFilename)

	expectedTempFile, err := os.Create(parsedLogsFilename + ".expected")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(expectedTempFile.Name())
	fmt.Println(expectedTempFile.Name())

	for _, entry := range expected {
		_, err := expectedTempFile.WriteString(entry.String())
		if err != nil {
			t.Fatalf("failed to write to temp file: %v", err)
		}
	}

	expectedTempFile.Close()
	expectedTempFile, err = os.Open(expectedTempFile.Name())
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer expectedTempFile.Close()

	isSameLogs, err := utils.CompareUnsortedFiles(parsedLogsFile, expectedTempFile)
	if err != nil {
		t.Fatalf("failed to compare files: %v", err)
	}

	if !isSameLogs {
		t.Fatalf("parsed logs are different from expected")
	}

	return expected
}

func testProcessLogFileBaseWithDB(t *testing.T, dbInsertionT string) {
	testDBPool, _ := setupTestDB()
	// defer cleanup()
	expected := testProcessLogFileBase(t, testDBPool, dbInsertionT)

	var count int64
	testDBPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM log_entries").Scan(&count)

	if count != int64(len(expected)) {
		t.Fatalf("expected %d entries in DB, got %d", len(expected), count)
	}

	for i, entry := range expected {
		var count int64
		testDBPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM log_entries WHERE date = $1 AND time = $2 AND server_ip = $3 AND method = $4 AND uri_stem = $5 AND uri_query = $6 AND port = $7 AND username = $8 AND client_ip = $9 AND user_agent = $10 AND status = $11 AND sub_status = $12 AND win32_status = $13 AND time_taken = $14",
			entry.Date, entry.Time, entry.ServerIP, entry.Method, entry.URIStem, entry.URIQuery, entry.Port, entry.Username, entry.ClientIP, entry.UserAgent, entry.Status, entry.SubStatus, entry.Win32Status, entry.TimeTaken).Scan(&count)

		if count != 1 {
			t.Fatalf("expected 1 entry in DB for entry %d, got %d", i, count)
		}
	}

	testDBPool.Close()
}

func TestParseLogLine(t *testing.T) {
	line := Cases[ParseCorrectLine].input()
	expected := Cases[ParseCorrectLine].expected().(*models.LogEntry)

	entry, err := parser.ParseLogLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *entry != *expected {
		t.Fatalf("expected %+v, got %+v", expected, entry)
	}
}

func TestProcessLogFileNoDB(t *testing.T) {
	testProcessLogFileBase(t, nil, "none")
}

func TestProcessLogFileBatchDBInsert(t *testing.T) {
	testProcessLogFileBaseWithDB(t, "batch")
}

func BenchmarkProcessLogFile(b *testing.B) {
	const logsDir = "../"

	cases := []struct {
		name         string
		file         string
		dbInsertionT string
	}{
		// {"mini_file-1.7MB-no-db", logsDir + "mini_u_ex190905.log", "none"},
		// {"mini_file-1.7MB-batch-db", logsDir + "mini_u_ex190905.log", "batch"},
		// {"below_md_file-17MB-no-db", logsDir + "below_med_u_ex190905.log", "none"},
		{"below_md_file-17MB-batch-db", logsDir + "below_med_u_ex190905.log", "batch"},
		// {"medium_file-29MB-no-db", logsDir + "u_ex190905.log", "none"},
		{"medium_file-29MB-batch-db", logsDir + "u_ex190905.log", "batch"},
		// {"below_lg_file-433MB-no-db", logsDir + "below_lg_u_ex190905.log", "none"},
		{"below_lg_file-433MB-batch-db", logsDir + "below_lg_u_ex190905.log", "batch"},
		// {"large_file-1.7GB-no-db", logsDir + "lg_u_ex190905.log", "none"},
		{"large_file-1.7GB-batch-db", logsDir + "lg_u_ex190905.log", "batch"},
	}

	for _, c := range cases {
		// Run a sub-benchmark for each case
		b.Run(c.name, func(b *testing.B) {
			testDB, cleanup := setupTestDB()
			defer cleanup()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := processor.ProcessLogFile(c.file, 8, testDB, c.dbInsertionT)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
