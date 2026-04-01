package main

// context.go — Gathers environmental context about the user's system
// to help the LLM give more relevant suggestions.

import (
	"os"
	"path/filepath"
	"runtime"
)

// ErrorContext holds environmental info sent to the LLM.
type ErrorContext struct {
	OS          string
	Shell       string
	Cwd         string
	ProjectType string
}

// gatherContext collects OS, shell, cwd, and project type.
func gatherContext() ErrorContext {
	var context ErrorContext

	context.OS = runtime.GOOS

	shellpath := os.Getenv("SHELL")
	shellbase := "unknown"
	if shellpath != "" {
		shellbase = filepath.Base(shellpath)
	}
	context.Shell = shellbase

	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	context.Cwd = cwd

	context.ProjectType = detectProjectType(cwd)

	return context
}

// detectProjectType checks for known project files in a directory
// and returns a label like "Node.js", "Go", "Python", etc.
// Returns "unknown" if no recognized files are found.
func detectProjectType(dir string) string {
	indicators := []struct {
		File        string
		ProjectType string
	}{
		{"package.json", "Node.js"},
		{"go.mod", "Go"},
		{"Cargo.toml", "Rust"},
		{"pyproject.toml", "Python"},
		{"requirements.txt", "Python"},
		{"Gemfile", "Ruby"},
		{"pom.xml", "Java (Maven)"},
		{"build.gradle", "Java/Kotlin (Gradle)"},
		{"Dockerfile", "Docker"},
		{"docker-compose.yml", "Docker Compose"},
	}

	for _, ind := range indicators {
		if _, err := os.Stat(filepath.Join(dir, ind.File)); err == nil {
			return ind.ProjectType
		}
	}

	return "unknown"
}
