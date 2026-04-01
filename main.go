package main

// main.go — Entry point for the `wtf` CLI tool.
// Orchestrates: API key → temp files → context → redact → prompt → stream.
// With `fix` subcommand, pipes the explanation into Claude Code to auto-fix.

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

// hasArg checks if a positional argument is present in os.Args.
func hasArg(name string) bool {
	for _, arg := range os.Args[1:] {
		if arg == name {
			return true
		}
	}
	return false
}

// getFixMode reads the fix mode preference from ~/.config/wtf/fix_mode.
// Returns "oneshot" (default) or "interactive".
func getFixMode() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "oneshot"
	}
	configPath := filepath.Join(homeDir, ".config", "wtf", "fix_mode")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "oneshot"
	}
	mode := strings.TrimSpace(string(data))
	if mode == "interactive" {
		return "interactive"
	}
	return "oneshot"
}

// getAgent reads the preferred coding agent from ~/.config/wtf/agent.
// Returns "claude" by default.
func getAgent() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "claude"
	}
	configPath := filepath.Join(homeDir, ".config", "wtf", "agent")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "claude"
	}
	agent := strings.TrimSpace(string(data))
	if agent != "" {
		return agent
	}
	return "claude"
}

// buildFixPrompt creates the prompt string sent to Claude Code.
// Should include the failed command, stderr output, and wtf's explanation,
// with an instruction to fix the issue.
func buildFixPrompt(command, stderr, explanation string) string {
	return fmt.Sprintf(`The following command failed:
%s

Error output:
%s

Explanation:
%s

Please fix this issue.`, command, stderr, explanation)
}

// runAgentOneShot runs the coding agent in non-interactive mode (agent -p "prompt").
// Prints the agent's output to the terminal and exits when done.
func runAgentOneShot(prompt string) error {
	agent := getAgent()
	cmd := exec.Command(agent, "-p", prompt, "--allowedTools", "Edit,Read,Write,Glob,Grep", "--permission-mode", "acceptEdits")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runAgentInteractive launches the coding agent interactively with context.
// The user gets a live session with the error context pre-loaded.
func runAgentInteractive(prompt string) error {
	agent := getAgent()
	cmd := exec.Command(agent, "-p", prompt, "--continue", "--allowedTools", "Edit,Read,Write,Glob,Grep", "--permission-mode", "acceptEdits")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	apiKey := getAPIKey()
	fixMode := hasArg("fix")

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

	// If fix subcommand is set, hand off to Claude Code to apply the fix.
	// Default: one-shot mode. With -i/--interactive: interactive session.
	if fixMode {
		fmt.Printf("\nHanding off to %s to fix...\n", getAgent())
		fmt.Println("")
		prompt := buildFixPrompt(lastCommand, redactedStderr, explanation)

		if getFixMode() == "interactive" {
			err = runAgentInteractive(prompt)
		} else {
			err = runAgentOneShot(prompt)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running coding agent: %v\n", err)
			os.Exit(1)
		}
	}
}
