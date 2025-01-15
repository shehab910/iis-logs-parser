package parser

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
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
	gorm.Model
}

type ParseError struct {
	Line    string
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("Parse error: %s in line: %s", e.Message, e.Line)
}

func ParseLogLine(line string) (*LogEntry, error) {
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
