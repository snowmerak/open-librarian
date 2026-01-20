package parser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// ParseDocx extracts text from a .docx file (which is a zip archive)
func ParseDocx(r io.Reader, filename string) (*Document, error) {
	// Read full content to support random access needed by zip reader
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read docx content: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to open docx as zip: %w", err)
	}

	// Find word/document.xml
	var documentXML *zip.File
	for _, f := range zipReader.File {
		if f.Name == "word/document.xml" {
			documentXML = f
			break
		}
	}

	if documentXML == nil {
		return nil, fmt.Errorf("invalid docx: word/document.xml not found")
	}

	// Open and parse the XML
	rc, err := documentXML.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	text, err := extractMarkdownFromDocxXML(rc)
	if err != nil {
		return nil, err
	}

	return &Document{
		Title:    strings.TrimSuffix(filename, ".docx"),
		Content:  text,
		Metadata: map[string]string{"type": "docx"},
	}, nil
}

func extractMarkdownFromDocxXML(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)
	var sb strings.Builder

	var (
		inTable           = false
		tableRows         = [][]string{}
		currentRow        = []string{}
		currentCellBuffer bytes.Buffer

		paragraphBuffer   bytes.Buffer
		currentHeadingLvl = 0
	)

	for {
		t, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch se := t.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "tbl":
				inTable = true
				tableRows = [][]string{}
			case "tr":
				if inTable {
					currentRow = []string{}
				}
			case "tc":
				if inTable {
					currentCellBuffer.Reset()
				}
			case "p":
				paragraphBuffer.Reset()
				currentHeadingLvl = 0
			case "pStyle":
				// This usually appears inside pPr inside p
				for _, attr := range se.Attr {
					if attr.Name.Local == "val" {
						val := attr.Value
						if strings.HasPrefix(val, "Heading") || strings.HasPrefix(val, "heading") {
							var lvl int
							cleanVal := strings.Map(func(r rune) rune {
								if r >= '0' && r <= '9' {
									return r
								}
								return -1
							}, val)
							if len(cleanVal) > 0 {
								fmt.Sscanf(cleanVal, "%d", &lvl)
								currentHeadingLvl = lvl
							}
						}
					}
				}
			case "t":
				var text string
				if err := decoder.DecodeElement(&text, &se); err == nil {
					if inTable {
						currentCellBuffer.WriteString(text)
					} else {
						paragraphBuffer.WriteString(text)
					}
				}
			}
		case xml.EndElement:
			switch se.Name.Local {
			case "tbl":
				inTable = false
				if len(tableRows) > 0 {
					sb.WriteString("\n")
					maxCols := 0
					for _, row := range tableRows {
						if len(row) > maxCols {
							maxCols = len(row)
						}
					}

					sanitize := func(s string) string {
						return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " "))
					}

					for i, row := range tableRows {
						sb.WriteString("|")
						for j := 0; j < maxCols; j++ {
							val := ""
							if j < len(row) {
								val = sanitize(row[j])
							}
							sb.WriteString(" " + val + " |")
						}
						sb.WriteString("\n")
						if i == 0 {
							sb.WriteString("|")
							for j := 0; j < maxCols; j++ {
								sb.WriteString(" --- |")
							}
							sb.WriteString("\n")
						}
					}
					sb.WriteString("\n")
				}
			case "tr":
				if inTable {
					tableRows = append(tableRows, currentRow)
				}
			case "tc":
				if inTable {
					currentRow = append(currentRow, currentCellBuffer.String())
				}
			case "p":
				text := paragraphBuffer.String()
				if strings.TrimSpace(text) == "" {
					continue
				}

				if inTable {
					currentCellBuffer.WriteString(" ")
				} else {
					if currentHeadingLvl > 0 {
						sb.WriteString("\n" + strings.Repeat("#", currentHeadingLvl) + " " + text + "\n")
					} else {
						sb.WriteString("\n" + text + "\n")
					}
				}
			}
		}
	}

	return strings.TrimSpace(sb.String()), nil
}
