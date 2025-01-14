package utils

import (
	"fmt"
	tableStr "iis-logs-parser/table_string"
	"strings"
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
