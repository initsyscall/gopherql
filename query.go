package main

import (
	"fmt"
	"strings"
)

func formatResults(columns []string, rows [][]string) string {
	if len(columns) == 0 {
		return "OK"
	}

	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = len(col)
	}
	for _, row := range rows {
		for i, val := range row {
			if i < len(widths) && len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	var sb strings.Builder

	sep := "+"
	for _, w := range widths {
		sep += strings.Repeat("-", w+2) + "+"
	}
	sep += "\n"

	sb.WriteString(sep)

	for i, col := range columns {
		sb.WriteString(fmt.Sprintf("| %-*s ", widths[i], col))
	}
	sb.WriteString("|\n")
	sb.WriteString(sep)

	for _, row := range rows {
		for i, val := range row {
			if i < len(widths) {
				sb.WriteString(fmt.Sprintf("| %-*s ", widths[i], val))
			}
		}
		sb.WriteString("|\n")
	}
	sb.WriteString(sep)

	return sb.String()
}
