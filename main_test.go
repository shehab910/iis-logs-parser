package main

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestParseLogLine(t *testing.T) {
	line := "2023-10-10 12:00:00 192.168.1.1 GET /index.html - 80 - 192.168.1.100 Mozilla/5.0 200 0 0 123"
	expected := &LogEntry{
		Date:        "2023-10-10",
		Time:        "12:00:00",
		ServerIP:    "192.168.1.1",
		Method:      "GET",
		URIStem:     "/index.html",
		URIQuery:    "-",
		Port:        "80",
		Username:    "-",
		ClientIP:    "192.168.1.100",
		UserAgent:   "Mozilla/5.0",
		Status:      "200",
		SubStatus:   "0",
		Win32Status: "0",
		TimeTaken:   "123",
	}

	entry, err := parseLogLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *entry != *expected {
		t.Fatalf("expected %v, got %v", expected, entry)
	}
}

func TestProcessLogFile(t *testing.T) {
	content := `#Fields: date time s-ip cs-method cs-uri-stem cs-uri-query s-port cs-username c-ip cs(User-Agent) sc-status sc-substatus sc-win32-status time-taken
2023-10-10 12:00:00 192.168.1.1 GET /index.html - 80 - 192.168.1.100 Mozilla/5.0 200 0 0 123
2023-10-10 12:00:01 192.168.1.1 GET /about.html - 80 - 192.168.1.101 Mozilla/5.0 404 0 0 456
2023-10-10 12:00:02 192.168.1.1 GET /contact.html - 80 - 192.168.1.102 Mozilla/5.0 500 0 0 789`

	tmpfile, err := os.CreateTemp("", "logfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	expected := map[string]int64{
		"200": 1,
		"404": 1,
		"500": 1,
	}

	result, err := processLogFile(tmpfile.Name(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*result) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}

	for k, v := range expected {
		if (*result)[k] != v {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestProcessLogLgFile(t *testing.T) {
	fileName := "../u_ex190905.log"
	expected := map[string]int64{
		"404": 14589,
		"304": 583,
		"302": 472,
		"301": 30,
		"500": 1,
		"400": 1,
		"200": 111065,
		"206": 2599,
		"403": 4,
		"406": 2,
	}

	result, err := processLogFile(fileName, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*result) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}

	for k, v := range expected {
		if (*result)[k] != v {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func BenchmarkProcessLogFile(b *testing.B) {
	content := `#Fields: date time s-ip cs-method cs-uri-stem cs-uri-query s-port cs-username c-ip cs(User-Agent) sc-status sc-substatus sc-win32-status time-taken
2023-10-10 12:00:00 192.168.1.1 GET /index.html - 80 - 192.168.1.100 Mozilla/5.0 200 0 0 123
2023-10-10 12:00:01 192.168.1.1 GET /about.html - 80 - 192.168.1.101 Mozilla/5.0 404 0 0 456
2023-10-10 12:00:02 192.168.1.1 GET /contact.html - 80 - 192.168.1.102 Mozilla/5.0 500 0 0 789`

	tmpfile, err := os.CreateTemp("", "logfile")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		b.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		_, err := processLogFile(tmpfile.Name(), 2)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkProcessLogLgFile(b *testing.B) {
	fileName := "../lg_u_ex190905.log"
	for i := 0; i < b.N; i++ {
		_, err := processLogFile(fileName, 1000)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
