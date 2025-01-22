package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	db "iis-logs-parser/database"
	"iis-logs-parser/parser"
	"iis-logs-parser/processor"
	"iis-logs-parser/utils"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
func setupTestDB(t testing.TB) (*gorm.DB, func()) {

	dbConfig := &db.DBConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "password",
		DBName:   "postgres-dev",
	}

	// Create a custom logger
	customLogger := logger.New(
		// Redirect GORM logs to zerolog
		Writer{Logger: &log.Logger},
		logger.Config{
			SlowThreshold: 1 * time.Second, // Set threshold to 1 second
			LogLevel:      logger.Warn,
			Colorful:      false,
		},
	)

	testDB, err := gorm.Open(postgres.Open(dbConfig.DSN()), &gorm.Config{
		Logger: customLogger,
	})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	cleanup := func() {
		testDB.Exec("DROP TABLE IF EXISTS log_entries")
	}
	cleanup()

	err = testDB.AutoMigrate(&parser.LogEntry{})
	if err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return testDB, cleanup
}

// Helper function to create test log file
func createTestLogFile(t testing.TB, testCase CaseType) (string, []*parser.LogEntry, func()) {
	content := Cases[testCase].input()
	expected := Cases[testCase].expected().([]*parser.LogEntry)

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

func testProcessLogFileBase(t *testing.T, db *gorm.DB, dbInsertionT string) []*parser.LogEntry {
	fileName, expected, cleanup := createTestLogFile(t, ParseCorrectLines)
	defer cleanup()

	err := processor.ProcessLogFile(fileName, 2, db, dbInsertionT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsedLogsFilename := "parsed_logs.txt"
	parsedLogsFile, err := os.Open(parsedLogsFilename)
	if err != nil {
		t.Fatalf("failed to open parsed logs file: %v", err)
	}
	defer parsedLogsFile.Close()
	// defer os.Remove(parsedLogsFilename)

	expectedTempFile, err := os.Create(parsedLogsFilename + ".expected")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	// defer os.Remove(expectedTempFile.Name())
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
	testDB, cleanup := setupTestDB(t)
	defer cleanup()
	expected := testProcessLogFileBase(t, testDB, dbInsertionT)

	var count int64
	if err := testDB.Model(&parser.LogEntry{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count entries: %v", err)
	}

	if count != int64(len(expected)) {
		t.Fatalf("expected %d entries in DB, got %d", len(expected), count)
	}

	var entries []parser.LogEntry
	if err := testDB.Find(&entries).Error; err != nil {
		t.Fatalf("failed to find entries: %v", err)
	}

	for _, entry := range entries {

		found := false
		for _, exp := range expected {
			// Ignoring db fields
			entry.Model.ID = 0
			entry.Model.CreatedAt = exp.Model.CreatedAt
			entry.Model.UpdatedAt = exp.Model.UpdatedAt
			entry.Model.DeletedAt = exp.Model.DeletedAt
			if entry == *exp {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %+v to be found in expected list %+v", entry, expected)
		}

	}
}

func TestParseLogLine(t *testing.T) {
	line := Cases[ParseCorrectLine].input()
	expected := Cases[ParseCorrectLine].expected().(*parser.LogEntry)

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
		{"below_md_file-17MB-no-db", logsDir + "below_med_u_ex190905.log", "none"},
		{"below_md_file-17MB-batch-db", logsDir + "below_med_u_ex190905.log", "batch"},
		{"medium_file-29MB-no-db", logsDir + "u_ex190905.log", "none"},
		{"medium_file-29MB-batch-db", logsDir + "u_ex190905.log", "batch"},
		{"below_lg_file-433MB-no-db", logsDir + "below_lg_u_ex190905.log", "none"},
		{"below_lg_file-433MB-batch-db", logsDir + "below_lg_u_ex190905.log", "batch"},
		// {"large_file-1.7GB-no-db", logsDir + "lg_u_ex190905.log", "none"},
		// {"large_file-1.7GB-batch-db", logsDir + "lg_u_ex190905.log", "batch"},
	}

	for _, c := range cases {
		// Run a sub-benchmark for each case
		b.Run(c.name, func(b *testing.B) {
			testDB, cleanup := setupTestDB(b)
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
