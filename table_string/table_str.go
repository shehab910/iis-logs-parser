package tableStr

import (
	"fmt"
	"strings"
)

type TableStr struct {
	Headers    []string
	Rows       [][]string
	headersSep []string
}

func New() *TableStr {
	return &TableStr{}
}

func (t *TableStr) SetRows(rows [][]string) {
	t.Rows = rows
}

func (t *TableStr) SetHeaders(headers []string) error {
	if len(headers) == 0 {
		return fmt.Errorf("headers cannot be empty")
	}
	t.Headers = headers

	t.headersSep = make([]string, len(headers))
	for i, h := range headers {
		t.headersSep[i] = strings.Repeat("-", len(h))
	}

	t.Rows = make([][]string, 0)
	return nil
}

func (t *TableStr) isValid() error {
	if len(t.Headers) == 0 {
		return fmt.Errorf("headers not set")
	}
	if len(t.Rows) == 0 {
		return fmt.Errorf("rows not set")
	}

	for _, row := range t.Rows {
		if len(row) != len(t.Headers) {
			return fmt.Errorf("invalid row format expected %d fields\nRow: %v", len(t.Headers), row)
		}
	}
	return nil
}

func (t *TableStr) String() (string, error) {
	if err := t.isValid(); err != nil {
		return "", err
	}
	sb := strings.Builder{}

	addRow := func(row ...string) {
		sb.WriteString("\n| ")
		for i, v := range row {
			fmtStr := fmt.Sprintf("%%-%dv | ", len(t.Headers[i]))
			sb.WriteString(fmt.Sprintf(fmtStr, v))
		}
		sb.WriteString("\n")
	}

	addRow(t.Headers...)

	addRow(t.headersSep...)

	for _, v := range t.Rows {
		addRow(v...)
	}
	return sb.String(), nil
}
