package parser

import (
	"bufio"
	"io"
	"strings"
)

// ParseText parses plain text or markdown files
func ParseText(r io.Reader, filename string) (*Document, error) {
	contentBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	content := string(contentBytes)

	// Check for YAML Frontmatter
	// ---
	// title: ...
	// ---

	title := strings.TrimSuffix(filename, ".md")
	title = strings.TrimSuffix(title, ".txt")
	metadata := make(map[string]string)

	if strings.HasPrefix(content, "---\n") || strings.HasPrefix(content, "---\r\n") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			// parts[0] is empty
			// parts[1] is frontmatter
			// parts[2] is content
			frontmatter := parts[1]
			content = strings.TrimSpace(parts[2])

			// Simple Yaml Parser for Title
			scanner := bufio.NewScanner(strings.NewReader(frontmatter))
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(strings.ToLower(line), "title:") {
					t := strings.TrimPrefix(line[6:], " ")
					t = strings.Trim(t, ` "'`)
					if t != "" {
						title = t
					}
				}
				// Can parse other metadata here
			}
		}
	}

	return &Document{
		Title:    title,
		Content:  content,
		Metadata: metadata,
	}, nil
}
