package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

// generateRandomString creates a random string of a given length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result = make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// generateRandomDate generates a random date within the last year
func generateRandomDate() string {
	// Set the base date (e.g., 1 year ago)
	start := time.Now().AddDate(-1, 0, 0) // One year ago
	// Generate a random number of days from the start date
	randomDuration := time.Duration(rand.Int63n(int64(time.Hour * 24 * 365)))
	randomDate := start.Add(randomDuration)
	// Return the formatted random date
	return randomDate.UTC().Format("2006-01-02T15:04:05.9999Z")
}

// generateCsvLogLine creates a log line with random data and random date
func generateCsvLogLine() string {
	// Generate random timestamp
	timestamp := generateRandomDate()

	// Generate random fields for the log line
	field1 := generateRandomString(10)                 // Random string field 1
	field2 := generateRandomString(15)                 // Random string field 2
	field3 := rand.Intn(1000)                          // Random integer field
	field4 := generateRandomString(20)                 // Random string field 3
	field5 := fmt.Sprintf("%.2f", rand.Float64()*1000) // Random float field

	// Combine the timestamp and the fields into a single log line
	logLine := fmt.Sprintf("%s, %s, %s, %d, %s, %s\n", timestamp, field1, field2, field3, field4, field5)
	return logLine
}

// generateCsvLogFile generates a log file of the requested size in megabytes
func generateCsvLogFile(filename string, sizeMB int) error {
	// Open the file for writing (or create it)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create file: %v", err)
	}
	defer file.Close()

	// Target size in bytes
	targetSize := sizeMB * 1024 * 1024

	// Generate log lines until the file reaches the target size
	var totalSize int
	for totalSize < targetSize {
		logLine := generateCsvLogLine()
		lineSize := len(logLine)

		// Write the log line to the file
		_, err := file.WriteString(logLine)
		if err != nil {
			return fmt.Errorf("could not write to file: %v", err)
		}

		totalSize += lineSize
	}

	fmt.Printf("Generated a log file of %d MB at %s\n", sizeMB, filename)
	return nil
}

func main() {
	// Seed the random number generator for random data generation
	rand.Seed(time.Now().UnixNano())

	// Set the desired size in MB (e.g., 5 MB)
	sizeMB := 1024 * 5
	filename := fmt.Sprintf("fake_log_%dMB.txt", sizeMB)

	// Generate the log file
	if err := generateCsvLogFile(filename, sizeMB); err != nil {
		fmt.Println("Error:", err)
	}
}
