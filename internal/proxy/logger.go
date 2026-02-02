package proxy

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/davidsenack/agentbox/internal/secrets"
)

// Logger writes network access logs with redaction
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	redactor *secrets.Redactor
}

// NewLogger creates a new network logger
func NewLogger(path string, redactor *secrets.Redactor) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f, redactor: redactor}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	return l.file.Close()
}

// LogPass logs a passed-through connection
func (l *Logger) LogPass(host, client string) {
	l.log("PASS", host, client, "")
}

// LogAuthInjected logs when auth was injected
func (l *Logger) LogAuthInjected(host, client string) {
	l.log("AUTH", host, client, "credentials injected")
}

// LogAuthSkipped logs when auth injection was skipped
func (l *Logger) LogAuthSkipped(host, reason string) {
	l.log("SKIP", host, "", reason)
}

// LogError logs an error during connection
func (l *Logger) LogError(host string, err error) {
	l.log("ERROR", host, "", err.Error())
}

func (l *Logger) log(action, host, client, detail string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().UTC().Format(time.RFC3339)

	var msg string
	if client != "" {
		msg = fmt.Sprintf("%s [%s] host=%s client=%s", timestamp, action, host, client)
	} else {
		msg = fmt.Sprintf("%s [%s] host=%s", timestamp, action, host)
	}

	if detail != "" {
		// Redact any secrets from messages
		detail = l.redactor.Redact(detail)
		msg += fmt.Sprintf(" detail=%q", detail)
	}

	fmt.Fprintln(l.file, msg)
}
