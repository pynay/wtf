package main

// api.go — API key management and streaming LLM requests.
// Uses the OpenAI chat completions API with SSE for real-time token streaming.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	apiURL   = "https://api.openai.com/v1/chat/completions"
	apiModel = "gpt-4o-mini"
)

// getAPIKey checks common env vars and a config file for an API key.
// Prints setup instructions and exits if no key is found.
func getAPIKey() string {
	// Check env vars that developers commonly have set
	envVars := []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY"}
	for _, name := range envVars {
		key := os.Getenv(name)
		if key != "" {
			return strings.TrimSpace(key)
		}
	}

	// Fall back to config file at ~/.config/wtf/api_key
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".config", "wtf", "api_key")
		data, err := os.ReadFile(configPath)
		if err == nil {
			key := strings.TrimSpace(string(data))
			if key != "" {
				return key
			}
		}
	}

	fmt.Fprintln(os.Stderr, "No API key found. Set one of these environment variables:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  export OPENAI_API_KEY=your-key-here")
	fmt.Fprintln(os.Stderr, "  export ANTHROPIC_API_KEY=your-key-here")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Or use a config file:")
	fmt.Fprintln(os.Stderr, "  mkdir -p ~/.config/wtf")
	fmt.Fprintln(os.Stderr, "  echo 'your-api-key' > ~/.config/wtf/api_key")
	os.Exit(1)

	return ""
}

// JSON request/response types for the OpenAI chat completions API.

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type streamDelta struct {
	Content string `json:"content"`
}

type streamChoice struct {
	Delta streamDelta `json:"delta"`
}

type streamResponse struct {
	Choices []streamChoice `json:"choices"`
}

// streamExplanation sends the prompt to the LLM and prints the response
// token-by-token to stdout as it arrives via SSE.
// Returns the full response text so callers can use it (e.g., --fix mode).
func streamExplanation(apiKey, systemPrompt, userPrompt string) (string, error) {
	req := chatRequest{
		Model: apiModel,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Stream: true,
	}

	var captured strings.Builder

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	httpResponse, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed (check your internet connection): %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResponse.Body)
		return "", fmt.Errorf("API returned status %d: %s", httpResponse.StatusCode, string(body))
	}

	// Parse SSE stream: each line starting with "data: " contains a JSON chunk.
	// The stream ends with "data: [DONE]".
	scanner := bufio.NewScanner(httpResponse.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		if data == "[DONE]" {
			break
		}

		var chunk streamResponse
		json.Unmarshal([]byte(data), &chunk)

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			fmt.Print(content)
			captured.WriteString(content)
		}
	}

	return captured.String(), nil
}
