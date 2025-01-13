package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

const CHUNK_SIZE = 250 * 1024 * 1024
const FILE_NAME = "fake_log_5120MB.txt"

func LoadFileChunk(file *os.File, buf []byte, offset int64) ([]byte, int64, error) {
	_, err := file.Seek(offset, 0)
	if err != nil {
		return nil, offset, fmt.Errorf("error seeking to offset %d: %w", offset, err)
	}
	reader := bufio.NewReaderSize(file, CHUNK_SIZE)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		return nil, offset, fmt.Errorf("error reading file: %w", err)
	}

	buf = buf[:n]

	if err == io.EOF {
		return buf, offset + int64(n), err
	}

	lastNewLine := bytes.LastIndexByte(buf, '\n')
	if lastNewLine == -1 {
		panic("chunk size too small to contain even one line, increase chunk size")
	}

	buf = buf[:lastNewLine]
	return buf, offset + int64(lastNewLine+1), nil
}

func main() {

	file, err := os.Open(FILE_NAME)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	buf := make([]byte, CHUNK_SIZE)

	newFile, err := os.Create("test.txt")
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer newFile.Close()

	var offset int64 = 0
	for {
		_, newOffset, err := LoadFileChunk(file, buf, offset)
		if err != nil {
			if err.Error() == "EOF" {
				// fmt.Println("End of file reached")
				break
			}
			// fmt.Printf("Error loading file chunk: %v\n", err)
			return
		}

		// fmt.Printf("Read chunk of size: %d bytes\n", len(chunk))

		offset = newOffset
	}
}

func Benchmark(t *testing.B) {
	t.ReportAllocs()

	for i := 0; i < t.N; i++ {

	}
}
