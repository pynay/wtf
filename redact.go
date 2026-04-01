package main

// redact.go — Strips sensitive data from error output before sending to the LLM.

import (
	"regexp"
)

// redactPatterns are compiled regexes matching common secret formats.
var redactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),                                                    // OpenAI keys
	regexp.MustCompile(`(?i)(key|token)[-_][A-Za-z0-9]{16,}`),                                     // Generic key/token prefixes
	regexp.MustCompile(`gh[pos]_[A-Za-z0-9]{36,}`),                                                // GitHub tokens
	regexp.MustCompile(`AKIA[A-Z0-9]{16}`),                                                        // AWS access key IDs
	regexp.MustCompile(`://[^:]+:([^@]{8,})@`),                                                    // Connection string passwords
	regexp.MustCompile(`(?i)(password|secret|token|api_key|apikey|auth)=[A-Za-z0-9+/=]{20,}`),     // Env var secrets
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-._~+/]+=*`),                                     // Bearer tokens
}

// redactSecrets replaces anything matching secret patterns with [REDACTED].
func redactSecrets(text string) string {
	result := text
	for _, pattern := range redactPatterns {
		result = pattern.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}
