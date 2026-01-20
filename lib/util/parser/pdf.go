package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ParsePDF parses a PDF file and extracts text
func ParsePDF(r io.Reader, filename string) (*Document, error) {
	// Read all content to memory to create ReaderAt
	// Note: For very large files, this might be an issue.
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read pdf content: %w", err)
	}

	reader, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to create pdf reader: %w", err)
	}

	var textBuilder strings.Builder

	// Read all pages
	totalPage := reader.NumPage()
	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := reader.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			// Iterate even if one page fails?
			continue
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	return &Document{
		Title:    strings.TrimSuffix(filename, ".pdf"),
		Content:  strings.TrimSpace(textBuilder.String()),
		Metadata: map[string]string{"type": "pdf"},
	}, nil
}
