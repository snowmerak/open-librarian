package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ParseExcel parses an Excel file and converts sheets to Markdown tables
func ParseExcel(r io.Reader, filename string) (*Document, error) {
	// Read content to support ReaderAt if needed, or OpenReader
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read excel content: %w", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	var sb strings.Builder

	sheets := f.GetSheetList()
	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue // Skip unreadable sheets
		}

		if len(rows) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("\n### Sheet: %s\n\n", sheet))

		// Convert to Markdown Table
		// Find max columns to align the table
		maxCols := 0
		for _, row := range rows {
			if len(row) > maxCols {
				maxCols = len(row)
			}
		}

		if maxCols == 0 {
			continue
		}

		// Header (first row)
		// If explicit header doesn't exist, we might treat first row as header
		// or if empty, generating generic headers (A, B, C...) is complex.
		// Let's assume Row 1 is header.

		// Header Row
		sb.WriteString("|")
		for i := 0; i < maxCols; i++ {
			val := ""
			if i < len(rows[0]) {
				val = normalizeCell(rows[0][i])
			}
			sb.WriteString(fmt.Sprintf(" %s |", val))
		}
		sb.WriteString("\n|")

		// Separator Row
		for i := 0; i < maxCols; i++ {
			sb.WriteString(" --- |")
		}
		sb.WriteString("\n")

		// Data Rows (from index 1)
		for i := 1; i < len(rows); i++ {
			sb.WriteString("|")
			for j := 0; j < maxCols; j++ {
				val := ""
				if j < len(rows[i]) {
					val = normalizeCell(rows[i][j])
				}
				sb.WriteString(fmt.Sprintf(" %s |", val))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return &Document{
		Title:    strings.TrimSuffix(filename, ".xlsx"),
		Content:  strings.TrimSpace(sb.String()),
		Metadata: map[string]string{"type": "excel"},
	}, nil
}

func normalizeCell(val string) string {
	// Escape pipes and newlines for markdown table
	val = strings.ReplaceAll(val, "|", "\\|")
	val = strings.ReplaceAll(val, "\n", " ")
	val = strings.ReplaceAll(val, "\r", "")
	if val == "" {
		return " "
	}
	return val
}
