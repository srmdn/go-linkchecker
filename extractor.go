package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// urlPattern matches http/https URLs in markdown content.
// Handles: [text](url), <url>, bare urls, image ![alt](url)
var urlPattern = regexp.MustCompile(`https?://[^\s\)\]"'<>]+`)

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

// ExtractURLs returns all unique http/https URLs found in a markdown file.
func ExtractURLs(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw := urlPattern.FindAllString(string(data), -1)

	seen := make(map[string]bool)
	var urls []string
	for _, u := range raw {
		// trim trailing punctuation that is not part of the URL
		u = strings.TrimRight(u, ".,;:!?)")
		if !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	return urls, nil
}
