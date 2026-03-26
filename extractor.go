package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// urlPattern matches http/https URLs in markdown content.
// Handles: [text](url), <url>, bare urls, image ![alt](url)
var urlPattern = regexp.MustCompile(`https?://[^\s\)\]"'<>` + "`" + `]+`)

// fencedCodeBlock matches ``` or ~~~ fenced code blocks (multiline).
var fencedCodeBlock = regexp.MustCompile("(?s)```[\\s\\S]*?```|~~~[\\s\\S]*?~~~")

// inlineCode matches `single-line inline code`.
var inlineCode = regexp.MustCompile("`[^`\n]+`")

// invalidURLChars matches URLs containing shell variables or template syntax
// that cannot be valid real URLs.
var invalidURLChars = regexp.MustCompile(`[${}|\\^]`)

// FindMarkdownFiles returns all .md files under dir recursively.
func FindMarkdownFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// stripCode removes fenced code blocks and inline code spans from markdown
// so URLs inside code examples are not checked.
func stripCode(s string) string {
	s = fencedCodeBlock.ReplaceAllString(s, "")
	s = inlineCode.ReplaceAllString(s, "")
	return s
}

// ExtractURLs returns all unique http/https URLs found in a markdown file,
// excluding URLs inside code blocks or containing invalid host characters.
func ExtractURLs(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := stripCode(string(data))
	raw := urlPattern.FindAllString(content, -1)

	seen := make(map[string]bool)
	var urls []string
	for _, u := range raw {
		// trim trailing punctuation that is not part of the URL
		u = strings.TrimRight(u, ".,;:!?)")
		// skip URLs with shell variables or invalid host characters
		if invalidURLChars.MatchString(u) {
			continue
		}
		if !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	return urls, nil
}
