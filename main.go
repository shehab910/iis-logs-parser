package main

import (
	"bufio"
	"fmt"
	"os"
	"testing"
)

const CHUNK_SIZE = 250 * 1024 * 1024
const FILE_NAME = "fake_log_1024MB.txt"

func LoadFileChunk(file *os.File, reader *bufio.Reader, buf *[]byte, offset int64) (*[]byte, int64, error) {

	_, err := file.Seek(offset, 0)
	if err != nil {
		return nil, offset, fmt.Errorf("error seeking to offset %d: %w", offset, err)
	}

	n, err := reader.Read(*buf)
	if err != nil {
		if err.Error() == "EOF" {
			return nil, offset, err
		}
		return nil, offset, fmt.Errorf("error reading file: %w", err)
	}

	(*buf) = (*buf)[:n]

	return buf, offset + int64(n), nil
}

func main() {

	file, err := os.Open(FILE_NAME)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, CHUNK_SIZE)

	buf := make([]byte, CHUNK_SIZE)

	// newFile, err := os.Create("test.txt")
	// if err != nil {
	// 	fmt.Printf("Error opening file: %v\n", err)
	// 	return
	// }
	// defer newFile.Close()

	var offset int64 = 0
	for {
		_, newOffset, err := LoadFileChunk(file, reader, &buf, offset)
		if err != nil {
			if err.Error() == "EOF" {
				// End of file reached, stop reading
				// fmt.Println("End of file reached")
				break
			}
			fmt.Printf("Error loading file chunk: %v\n", err)
			return
		}

		// fmt.Printf("Read chunk of size: %d bytes\n", len(*chunk))
		// newFile.Write(chunk)

		offset = newOffset
	}
}

func Benchmark(t *testing.B) {
	t.ReportAllocs()

	for i := 0; i < t.N; i++ {

	}
}
