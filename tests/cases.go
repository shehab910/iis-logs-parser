package tests

import "iis-logs-parser/parser"

type Case struct {
	input    func() string
	expected func() interface{}
}

// enum of case types
type CaseType int

const (
	case1 CaseType = iota
	ParseCorrectLine
	ParseCorrectLines
)

var Cases = map[CaseType]Case{
	case1: {
		input:    func() string { return "test" },
		expected: func() interface{} { return "test" },
	},
	ParseCorrectLine: {
		input: func() string {
			return "2023-10-10 12:00:00 192.168.1.1 GET /index.html - 80 - 192.168.1.100 Mozilla/5.0 200 0 0 123"
		},
		expected: func() interface{} {
			return &parser.LogEntry{
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
		},
	},
	ParseCorrectLines: {
		input: func() string {
			return `#Fields: date time s-ip cs-method cs-uri-stem cs-uri-query s-port cs-username c-ip cs(User-Agent) sc-status sc-substatus sc-win32-status time-taken
2023-10-10 12:00:00 192.168.1.1 GET /index.html - 80 - 192.168.1.100 Mozilla/5.0 200 0 0 123
2023-10-10 12:00:01 192.168.1.1 GET /about.html - 80 - 192.168.1.101 Mozilla/5.0 404 0 0 456
2023-10-10 12:00:02 192.168.1.1 GET /contact.html - 80 - 192.168.1.102 Mozilla/5.0 500 0 0 789`
		},
		expected: func() interface{} {
			return []*parser.LogEntry{
				{
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
				},
				{
					Date:        "2023-10-10",
					Time:        "12:00:01",
					ServerIP:    "192.168.1.1",
					Method:      "GET",
					URIStem:     "/about.html",
					URIQuery:    "-",
					Port:        "80",
					Username:    "-",
					ClientIP:    "192.168.1.101",
					UserAgent:   "Mozilla/5.0",
					Status:      "404",
					SubStatus:   "0",
					Win32Status: "0",
					TimeTaken:   "456",
				},
				{
					Date:        "2023-10-10",
					Time:        "12:00:02",
					ServerIP:    "192.168.1.1",
					Method:      "GET",
					URIStem:     "/contact.html",
					URIQuery:    "-",
					Port:        "80",
					Username:    "-",
					ClientIP:    "192.168.1.102",
					UserAgent:   "Mozilla/5.0",
					Status:      "500",
					SubStatus:   "0",
					Win32Status: "0",
					TimeTaken:   "789",
				},
			}
		},
	},
}
