package utils

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"sort"
	"strings"
	"sync"

	tableStr "iis-logs-parser/table_string"
)

func MapToTableLogMsg(mp *map[string]int64) (string, error) {
	rows := [][]string{}
	for k, v := range *mp {
		rows = append(rows, []string{k, fmt.Sprintf("%v", v)})
	}

	t := tableStr.New()
	t.SetHeaders([]string{"Status Code", "Number of Occurrences"})
	t.SetRows(rows)

	resStr, err := t.String()

	if err != nil {
		return "", err
	}
	return resStr, nil
}

func MapToStr(mp *map[string]int64) string {
	sb := strings.Builder{}
	sb.WriteString("\n")

	for k, v := range *mp {
		sb.WriteString(fmt.Sprintf("%v: %v\n", k, v))
	}
	return sb.String()
}

// CompareUnsortedFiles ensures that two files have the same lines
// even if the order of the lines are different in the two files
// NOTE (unsafe): This functions uses hashing, so it is not 100% accurate though it's highly unlikely in practice
func CompareUnsortedLgFilesUnsafe(file1Content, file2Content *os.File) (bool, error) {
	// TODO: improve performance by reading files in chunks,
	// using go routines for f1 and f2

	file1ContentStat, err := file1Content.Stat()
	if err != nil {

		return false, err
	}

	file2ContentStat, err := file2Content.Stat()
	if err != nil {
		return false, err
	}

	if file1ContentStat.Size() != file2ContentStat.Size() {
		return false, nil
	}

	wg := &sync.WaitGroup{}
	file1ContentMap := make(map[string]int64)
	file2ContentMap := make(map[string]int64)

	loadFileLinesInMap := func(file *os.File, fileMap *map[string]int64) {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			hashedLine := fmt.Sprintf("%x", md5.Sum([]byte(scanner.Text())))
			(*fileMap)[hashedLine]++
		}
	}

	wg.Add(2)
	go func() {
		loadFileLinesInMap(file1Content, &file1ContentMap)
		wg.Done()
	}()

	go func() {
		loadFileLinesInMap(file2Content, &file2ContentMap)
		wg.Done()
	}()

	wg.Wait()

	if len(file1ContentMap) != len(file2ContentMap) {
		return false, nil
	}

	if maps.Equal(file1ContentMap, file2ContentMap) {
		return true, nil
	}

	return false, errors.New("not implemented yet")
}

func CompareUnsortedFiles(file1, file2 *os.File) (bool, error) {
	file1Stat, err := file1.Stat()
	if err != nil {

		return false, err
	}

	file2Stat, err := file2.Stat()
	if err != nil {
		return false, err
	}

	if file1Stat.Size() != file2Stat.Size() {
		return false, nil
	}

	wg := &sync.WaitGroup{}
	file1Content := make([]byte, file1Stat.Size())
	file2Content := make([]byte, file2Stat.Size())

	errsChan := make(chan error, 2)
	bytesLenChan := make(chan int, 2)
	readFile := func(file *os.File, fileContent []byte, wg *sync.WaitGroup) {
		bytesLen, err := file.Read(fileContent)
		bytesLenChan <- bytesLen
		errsChan <- err
		wg.Done()
	}
	//"{Date:2023-10-10 Time:12:00:00 ServerIP:192.168.1.1 Method:GET URIStem:/index.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.100 UserAgent:Mozilla/5.0 Status:200 SubStatus:0 Win32Status:0 TimeTaken:123 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}"
	//"{Date:2023-10-10 Time:12:00:00 ServerIP:192.168.1.1 Method:GET URIStem:/index.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.100 UserAgent:Mozilla/5.0 Status:200 SubStatus:0 Win32Status:0 TimeTaken:123 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}"
	wg.Add(2)
	go readFile(file1, file1Content, wg)
	go readFile(file2, file2Content, wg)

	wg.Wait()
	close(errsChan)
	close(bytesLenChan)

	for err := range errsChan {
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}
	}

	bytes1Len := <-bytesLenChan
	bytes2Len := <-bytesLenChan
	if bytes1Len != bytes2Len {
		return false, nil
	}

	file1Lines := strings.Split(string(file1Content), "\n")
	file2Lines := strings.Split(string(file2Content), "\n")

	if len(file1Lines) != len(file2Lines) {
		return false, nil
	}
	
	sort.Slice(file1Lines, func(i, j int) bool {
		return file1Lines[i] < file1Lines[j]
	})
	
	sort.Slice(file2Lines, func(i, j int) bool {
		return file2Lines[i] < file2Lines[j]
	})

	for i := 0; i < len(file1Lines); i++ {
		if file1Lines[i] != file2Lines[i] {
			return false, nil
		}
	}

	return true, nil
}

