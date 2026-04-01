package main

// main.go — Entry point for the `wtf` CLI tool.
// Orchestrates: API key → temp files → context → redact → prompt → stream.
// With --fix flag, pipes the explanation into Claude Code to auto-fix.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// readTmpFile reads a file from the system temp directory and returns
// its contents as a trimmed string.
func readTmpFile(filename string) (string, error) {
	tempdir := os.TempDir()
	path := filepath.Join(tempdir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// hasFlag checks if a flag is present in os.Args.
func hasFlag(flag string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return false
}

// runClaudeFix pipes the error context and explanation into Claude Code.
//
// TODO: Implement this function.
//
// HINTS:
//   - exec.Command("claude", args...) creates a command to run
//   - You can pass a prompt directly with: exec.Command("claude", "-p", prompt)
//   - cmd.Stdin, cmd.Stdout, cmd.Stderr can be wired to os.Stdin/Stdout/Stderr
//     so Claude Code runs interactively in the user's terminal
//   - cmd.Run() executes the command and waits for it to finish
//   - Build a prompt string that gives Claude Code the context it needs:
//     the failed command, the error output, and the explanation from wtf
//   - The prompt should tell Claude to fix the issue, not just explain it
func runClaudeFix(command, stderr, explanation string) error {
	// TODO: Build a prompt for Claude Code that includes:
	//   - The failed command
	//   - The error output
	//   - The explanation from wtf
	//   - An instruction to fix the issue

	// TODO: Create the exec.Command with "claude" and "-p" flag

	// TODO: Wire cmd.Stdout and cmd.Stderr to os.Stdout and os.Stderr
	//   so the user sees Claude Code's output in real time

	// TODO: Run the command and return any error

	_ = command
	_ = stderr
	_ = explanation
	_ = exec.Command

	return nil
}

func main() {
	apiKey := getAPIKey()
	fixMode := hasFlag("--fix")

	lastCommand, err := readTmpFile("wtf_last_command")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No recent failed command found.")
			fmt.Println("Make sure you've installed the shell hook: bash setup.sh")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error reading last command: %v\n", err)
		os.Exit(1)
	}
	if lastCommand == "" {
		fmt.Println("No recent error to explain.")
		os.Exit(0)
	}

	lastStderr, err := readTmpFile("wtf_last_stderr")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No error output captured.")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error reading stderr: %v\n", err)
		os.Exit(1)
	}
	if lastStderr == "" {
		fmt.Println("The last failed command didn't produce any error output.")
		os.Exit(0)
	}

	context := gatherContext()
	redactedStderr := redactSecrets(lastStderr)

	systemPrompt := getSystemPrompt()
	userPrompt := buildUserPrompt(lastCommand, context, redactedStderr)

	fmt.Println("")
	explanation, err := streamExplanation(apiKey, systemPrompt, userPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("")

	// If --fix flag is set, hand off to Claude Code to apply the fix
	if fixMode {
		fmt.Println("\nHanding off to Claude Code to fix...")
		fmt.Println("")
		err = runClaudeFix(lastCommand, redactedStderr, explanation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running Claude Code: %v\n", err)
			os.Exit(1)
		}
	}
}
