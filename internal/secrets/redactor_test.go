package secrets

import (
	"strings"
	"testing"
)

func TestNewRedactor(t *testing.T) {
	patterns := []string{
		`sk-ant-[a-zA-Z0-9-]+`,
		`sk-[a-zA-Z0-9]{48}`,
	}

	r := NewRedactor(patterns)

	if len(r.patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(r.patterns))
	}
}

func TestNewRedactorInvalidPattern(t *testing.T) {
	patterns := []string{
		`sk-ant-[a-zA-Z0-9-]+`, // valid
		`[invalid`,             // invalid regex
	}

	r := NewRedactor(patterns)

	// Should only have 1 valid pattern
	if len(r.patterns) != 1 {
		t.Errorf("expected 1 valid pattern, got %d", len(r.patterns))
	}
}

func TestRedactAnthropicKey(t *testing.T) {
	r := NewRedactor([]string{`sk-ant-[a-zA-Z0-9-]+`})

	tests := []struct {
		input    string
		contains string
	}{
		{
			input:    "My key is sk-ant-api03-abcdefghij123456",
			contains: "[REDACTED]",
		},
		{
			input:    "curl -H 'x-api-key: sk-ant-test-key-12345'",
			contains: "[REDACTED]",
		},
		{
			input:    "No key here",
			contains: "No key here",
		},
	}

	for _, tt := range tests {
		result := r.Redact(tt.input)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("expected result to contain %q, got %q", tt.contains, result)
		}
	}
}

func TestRedactOpenAIKey(t *testing.T) {
	r := NewRedactor([]string{`sk-[a-zA-Z0-9]{48}`})

	// sk- plus exactly 48 alphanumeric chars = 51 total
	key := "sk-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOP123456"
	input := "Using key: " + key

	result := r.Redact(input)

	if strings.Contains(result, key) {
		t.Error("key should be redacted")
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Error("result should contain [REDACTED]")
	}
}

func TestRedactPreservesPartialInfo(t *testing.T) {
	r := NewRedactor([]string{`sk-ant-[a-zA-Z0-9-]+`})

	// Long key should show first 4 and last 4 chars
	input := "sk-ant-api03-verylongkeyvalue12345"
	result := r.Redact(input)

	// Should contain partial key info for debugging
	if !strings.Contains(result, "sk-a") {
		t.Error("result should contain first 4 chars")
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Error("result should contain [REDACTED]")
	}
}

func TestRedactMultiplePatterns(t *testing.T) {
	r := NewRedactor([]string{
		`sk-ant-[a-zA-Z0-9-]+`,
		`password=[^\s]+`,
	})

	input := "key=sk-ant-test123 password=secret123"
	result := r.Redact(input)

	if strings.Contains(result, "sk-ant-test123") {
		t.Error("API key should be redacted")
	}
	if strings.Contains(result, "secret123") {
		t.Error("password should be redacted")
	}
}

func TestRedactEmptyPatterns(t *testing.T) {
	r := NewRedactor([]string{})

	input := "sk-ant-test123"
	result := r.Redact(input)

	// With no patterns, nothing should be redacted
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}
