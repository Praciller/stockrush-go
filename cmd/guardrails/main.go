package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const maxFileSize = 5 << 20

var required = []string{
	"README.md", ".env.example", "Dockerfile", "docker-compose.yml", "go.mod",
	"db/migrations/001_init.sql", "internal/store/reservation.go", "loadtest/flash-sale.js",
	"web/package-lock.json", "docs/portfolio-review.md", "reports/local_portfolio_report.md",
}

var secretMarkers = [][]byte{
	[]byte("-----BEGIN " + "PRIVATE KEY-----"),
	[]byte("-----BEGIN RSA " + "PRIVATE KEY-----"),
	[]byte("ghp" + "_"),
	[]byte("github" + "_pat_"),
	[]byte("sk" + "_live_"),
	[]byte("AK" + "IA"),
}

func main() {
	var failures []string
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			failures = append(failures, "missing required file: "+path)
		}
	}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() && path != "." && (name == ".git" || name == "node_modules" || name == "dist" || name == ".playwright-cli" || name == "tmp") {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		if name == ".env" || strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".key") {
			failures = append(failures, "forbidden secret file: "+path)
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxFileSize {
			failures = append(failures, fmt.Sprintf("file exceeds 5 MiB: %s (%d bytes)", path, info.Size()))
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, marker := range secretMarkers {
			if bytes.Contains(body, marker) {
				failures = append(failures, fmt.Sprintf("possible secret marker %q in %s", marker, path))
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "guardrail scan failed:", err)
		os.Exit(1)
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Fprintln(os.Stderr, failure)
		}
		os.Exit(1)
	}
	fmt.Println("repository guardrails: PASS")
}
