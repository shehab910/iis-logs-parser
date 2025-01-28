package models

import "fmt"

type LogEntry struct {
	// IMPORTANT: This struct is used by both pgx (parser) and gorm (api)
	// and MUST be auto migrated with gorm
	ID          uint `gorm:"primarykey"`
	LogFileID   uint // Foreign key to the owner file
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

func (entry *LogEntry) String() string {
	return fmt.Sprintf("%+v\n", *entry)
}
