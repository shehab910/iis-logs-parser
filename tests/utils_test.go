package tests

import (
	"iis-logs-parser/utils"
	"os"
	"testing"
)

func testCompareUnsortedFilesBase(t *testing.T, testVariant string, funcToTest func(*os.File, *os.File) (bool, error)) {
	baseLogs := `{Date:2023-10-10 Time:12:00:00 ServerIP:192.168.1.1 Method:GET URIStem:/index.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.100 UserAgent:Mozilla/5.0 Status:200 SubStatus:0 Win32Status:0 TimeTaken:123 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2023-10-10 Time:12:00:01 ServerIP:192.168.1.1 Method:GET URIStem:/about.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.101 UserAgent:Mozilla/5.0 Status:404 SubStatus:0 Win32Status:0 TimeTaken:456 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2023-10-10 Time:12:00:02 ServerIP:192.168.1.1 Method:GET URIStem:/contact.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.102 UserAgent:Mozilla/5.0 Status:500 SubStatus:0 Win32Status:0 TimeTaken:789 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
`
	baseLogsRandomized := `{Date:2023-10-10 Time:12:00:00 ServerIP:192.168.1.1 Method:GET URIStem:/index.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.100 UserAgent:Mozilla/5.0 Status:200 SubStatus:0 Win32Status:0 TimeTaken:123 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2023-10-10 Time:12:00:02 ServerIP:192.168.1.1 Method:GET URIStem:/contact.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.102 UserAgent:Mozilla/5.0 Status:500 SubStatus:0 Win32Status:0 TimeTaken:789 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2023-10-10 Time:12:00:01 ServerIP:192.168.1.1 Method:GET URIStem:/about.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.101 UserAgent:Mozilla/5.0 Status:404 SubStatus:0 Win32Status:0 TimeTaken:456 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
`
	baseLogsPlusOne := `{Date:2023-10-10 Time:12:00:00 ServerIP:192.168.1.1 Method:GET URIStem:/index.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.100 UserAgent:Mozilla/5.0 Status:200 SubStatus:0 Win32Status:0 TimeTaken:123 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2023-10-10 Time:12:00:02 ServerIP:192.168.1.1 Method:GET URIStem:/contact.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.102 UserAgent:Mozilla/5.0 Status:500 SubStatus:0 Win32Status:0 TimeTaken:789 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2023-10-10 Time:12:00:01 ServerIP:192.168.1.1 Method:GET URIStem:/about.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.101 UserAgent:Mozilla/5.0 Status:404 SubStatus:0 Win32Status:0 TimeTaken:456 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2023-10-10 Time:12:00:01 ServerIP:192.168.1.1 Method:GET URIStem:/about.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.101 UserAgent:Mozilla/5.0 Status:404 SubStatus:0 Win32Status:0 TimeTaken:456 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
`
	baseLogsMinusOne := `{Date:2023-10-10 Time:12:00:00 ServerIP:192.168.1.1 Method:GET URIStem:/index.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.100 UserAgent:Mozilla/5.0 Status:200 SubStatus:0 Win32Status:0 TimeTaken:123 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2023-10-10 Time:12:00:01 ServerIP:192.168.1.1 Method:GET URIStem:/about.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.101 UserAgent:Mozilla/5.0 Status:404 SubStatus:0 Win32Status:0 TimeTaken:456 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
`
	wrongLogs := `{Date:2012-10-10 Time:12:00:00 ServerIP:192.168.1.1 Method:GET URIStem:/index.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.100 UserAgent:Mozilla/5.0 Status:200 SubStatus:0 Win32Status:0 TimeTaken:123 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2213-10-10 Time:12:00:01 ServerIP:192.168.1.1 Method:GET URIStem:/about.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.101 UserAgent:Mozilla/5.0 Status:404 SubStatus:0 Win32Status:0 TimeTaken:456 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
{Date:2022-10-21 Time:12:00:02 ServerIP:192.168.1.1 Method:GET URIStem:/contact.html URIQuery:- Port:80 Username:- ClientIP:192.168.1.102 UserAgent:Mozilla/5.0 Status:500 SubStatus:0 Win32Status:0 TimeTaken:789 Model:{ID:0 CreatedAt:0001-01-01 00:00:00 +0000 UTC UpdatedAt:0001-01-01 00:00:00 +0000 UTC DeletedAt:{Time:0001-01-01 00:00:00 +0000 UTC Valid:false}}}
`

	testCases := []struct {
		name     string
		file1    string
		file2    string
		expected bool
	}{
		{testVariant + "baseLogs vs baseLogs", baseLogs, baseLogs, true},
		{testVariant + "baseLogs vs sameButRandomized", baseLogs, baseLogsRandomized, true},
		{testVariant + "baseLogs vs baseLogsPlusOne", baseLogs, baseLogsPlusOne, false},
		{testVariant + "baseLogs vs baseLogsMinusOne", baseLogs, baseLogsMinusOne, false},
		{testVariant + "baseLogs vs wrongLogs", baseLogs, wrongLogs, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log1File, err := os.CreateTemp("", "log1*")
			if err != nil {
				t.Fatal(err)
			}

			log2File, err := os.CreateTemp("", "log2*")
			if err != nil {
				t.Fatal(err)
			}

			_, err = log1File.WriteString(tc.file1)
			if err != nil {
				t.Fatal(err)
			}

			_, err = log2File.WriteString(tc.file2)
			if err != nil {
				t.Fatal(err)
			}

			log1File.Close()
			log2File.Close()
			defer os.Remove(log1File.Name())
			defer os.Remove(log2File.Name())

			log1File, err = os.Open(log1File.Name())
			if err != nil {
				t.Fatal(err)
			}
			log2File, err = os.Open(log2File.Name())
			if err != nil {
				t.Fatal(err)
			}

			actual, err := funcToTest(log1File, log2File)
			if err != nil {
				t.Fatal(err)
			}

			if actual != tc.expected {
				t.Errorf("expected %v but got %v", tc.expected, actual)
			}
		})
	}

}

func TestCompareUnsortedFiles(t *testing.T) {
	testCompareUnsortedFilesBase(t, "func: CompareUnsortedFiles, ", utils.CompareUnsortedFiles)
}

func TestCompareUnsortedLgFilesUnsafe(t *testing.T) {
	testCompareUnsortedFilesBase(t, "func: CompareUnsortedLgFilesUnsafe, ", utils.CompareUnsortedLgFilesUnsafe)
}
