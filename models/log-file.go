package models

import (
	"time"

	"gorm.io/gorm"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

type LogFile struct {
	gorm.Model
	LogEntries []LogEntry `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	// UserID         uint           `gorm:"not null"`                   // owner of the file
	DomainID       uint       `gorm:"not null"`                   // Domain to which the log file belongs.
	Name           string     `gorm:"type:varchar(255);not null"` // Name of the uploaded file.
	Size           uint       `gorm:"not null"`                   // File size in bytes.
	Status         Status     `gorm:"type:varchar(20);not null"`  // Initially set to Pending
	StartTimestamp *time.Time `gorm:"type:timestamp"`             // The timestamp of the oldest log in the log file.
	EndTimestamp   *time.Time `gorm:"type:timestamp"`             // The timestamp of the most recent log in the log file.
	ParsingTime    int64      ``                                  // The time taken to parse the log file.
}
