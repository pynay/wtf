package main

// prompt.go — System prompt and user prompt template.

import "fmt"

const systemPrompt = `You are a senior developer helping debug a command-line error.
Be concise and practical. No filler, no caveats, just the answer.

Respond in this exact format:

WHAT HAPPENED
One to three sentences explaining the error in plain English.

FIX
The exact command(s) to run. One per line.

WHY
One sentence explaining why this fixes it.

Keep your response under 150 words. Most likely cause first.`

func getSystemPrompt() string {
	return systemPrompt
}

// buildUserPrompt fills in the error-specific details for the LLM.
func buildUserPrompt(command string, ctx ErrorContext, stderr string) string {
	return fmt.Sprintf(`Failed command: %s
Shell: %s
OS: %s
Working directory: %s
Project type: %s

Error output:
%s`, command, ctx.Shell, ctx.OS, ctx.Cwd, ctx.ProjectType, stderr)
}
