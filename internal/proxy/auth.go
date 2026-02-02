package proxy

import (
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/davidsenack/agentbox/internal/config"
)

// AuthInjector injects authentication headers for configured hosts
type AuthInjector struct {
	mu      sync.RWMutex
	configs map[string]authEntry // hostname -> auth config
}

type authEntry struct {
	header string
	value  string
}

// NewAuthInjector creates a new auth injector from config
func NewAuthInjector(configs []config.AuthConfig) *AuthInjector {
	a := &AuthInjector{
		configs: make(map[string]authEntry),
	}

	for _, cfg := range configs {
		host := strings.ToLower(cfg.Host)
		value := os.Getenv(cfg.Env)
		if value != "" {
			a.configs[host] = authEntry{
				header: cfg.Header,
				value:  value,
			}
		}
	}

	return a
}

// NeedsInjection checks if a host requires auth injection
func (a *AuthInjector) NeedsInjection(hostname string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	hostname = strings.ToLower(hostname)
	_, ok := a.configs[hostname]
	return ok
}

// Inject adds authentication header if configured for this host
// Returns true if auth was injected
func (a *AuthInjector) Inject(hostname string, headers http.Header) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	hostname = strings.ToLower(hostname)
	entry, ok := a.configs[hostname]
	if !ok {
		return false
	}

	headers.Set(entry.header, entry.value)
	return true
}

// Hosts returns list of hosts with auth configured (for logging)
func (a *AuthInjector) Hosts() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	hosts := make([]string, 0, len(a.configs))
	for h := range a.configs {
		hosts = append(hosts, h)
	}
	return hosts
}
