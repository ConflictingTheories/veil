package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"
)

// markdownToHTML converts basic markdown to HTML
func markdownToHTML(markdown string) string {
	if markdown == "" {
		return ""
	}

	result := html.EscapeString(markdown)

	// Headers
	result = regexp.MustCompile(`(?m)^##### (.*?)$`).ReplaceAllString(result, "<h5>$1</h5>")
	result = regexp.MustCompile(`(?m)^#### (.*?)$`).ReplaceAllString(result, "<h4>$1</h4>")
	result = regexp.MustCompile(`(?m)^### (.*?)$`).ReplaceAllString(result, "<h3>$1</h3>")
	result = regexp.MustCompile(`(?m)^## (.*?)$`).ReplaceAllString(result, "<h2>$1</h2>")
	result = regexp.MustCompile(`(?m)^# (.*?)$`).ReplaceAllString(result, "<h1>$1</h1>")

	// Bold and italic
	result = regexp.MustCompile(`\*\*\*(.*?)\*\*\*`).ReplaceAllString(result, "<strong><em>$1</em></strong>")
	result = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(result, "<strong>$1</strong>")
	result = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllString(result, "<em>$1</em>")
	result = regexp.MustCompile(`___(.*?)___`).ReplaceAllString(result, "<strong><em>$1</em></strong>")
	result = regexp.MustCompile(`__(.*?)__`).ReplaceAllString(result, "<strong>$1</strong>")
	result = regexp.MustCompile(`_(.*?)_`).ReplaceAllString(result, "<em>$1</em>")

	// Code
	result = regexp.MustCompile("`([^`]+)`").ReplaceAllString(result, "<code>$1</code>")

	// Links
	result = regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`).ReplaceAllString(result, `<a href="$2">$1</a>`)

	// Lists
	lines := strings.Split(result, "\n")
	var output strings.Builder
	inList := false
	inOrderedList := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Unordered list
		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
			if !inList {
				output.WriteString("<ul>\n")
				inList = true
			}
			content := strings.TrimPrefix(strings.TrimPrefix(trimmed, "* "), "- ")
			output.WriteString(fmt.Sprintf("<li>%s</li>\n", content))
		} else if regexp.MustCompile(`^\d+\.\s`).MatchString(trimmed) {
			// Ordered list
			if !inOrderedList {
				output.WriteString("<ol>\n")
				inOrderedList = true
			}
			content := regexp.MustCompile(`^\d+\.\s`).ReplaceAllString(trimmed, "")
			output.WriteString(fmt.Sprintf("<li>%s</li>\n", content))
		} else {
			// Close lists
			if inList {
				output.WriteString("</ul>\n")
				inList = false
			}
			if inOrderedList {
				output.WriteString("</ol>\n")
				inOrderedList = false
			}
			output.WriteString(line + "\n")
		}
	}

	// Close any remaining lists
	if inList {
		output.WriteString("</ul>\n")
	}
	if inOrderedList {
		output.WriteString("</ol>\n")
	}

	result = output.String()

	// Paragraphs - wrap standalone lines
	lines = strings.Split(result, "\n")
	var paragraphs strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "<") {
			paragraphs.WriteString(fmt.Sprintf("<p>%s</p>\n", trimmed))
		} else {
			paragraphs.WriteString(line + "\n")
		}
	}

	return paragraphs.String()
}

// slugify converts a string to a URL-friendly slug
func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and special characters with hyphens
	s = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`[\s_]+`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")

	// Trim hyphens from start and end
	s = strings.Trim(s, "-")

	return s
}

// truncate truncates a string to a maximum length with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Find the last space before maxLen
	truncated := s[:maxLen]
	if lastSpace := strings.LastIndexFunc(truncated, unicode.IsSpace); lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// excerpt generates an excerpt from markdown content
func excerpt(content string, maxLen int) string {
	// Strip markdown syntax
	text := content
	text = regexp.MustCompile(`#+ `).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\*\*?(.*?)\*\*?`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`\[(.*?)\]\(.*?\)`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile("`(.*?)`").ReplaceAllString(text, "$1")

	// Get first paragraph
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return truncate(trimmed, maxLen)
		}
	}

	return truncate(text, maxLen)
}
