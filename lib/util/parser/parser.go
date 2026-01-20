package parser

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Document represents parsed document content
type Document struct {
	Title    string
	Content  string
	Metadata map[string]string
}

// Parse parses the content from reader based on file extension
func Parse(r io.Reader, filename string) (*Document, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".pdf":
		return ParsePDF(r, filename)
	case ".docx":
		return ParseDocx(r, filename)
	case ".xlsx", ".xls":
		return ParseExcel(r, filename)
	case ".md", ".markdown", ".txt":
		return ParseText(r, filename)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}
