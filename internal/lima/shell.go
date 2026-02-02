package lima

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Shell opens an interactive shell in the Lima VM as the 'agent' user
// allowedEnvVars specifies which env vars to pass through (e.g., ANTHROPIC_API_KEY)
func (m *Manager) Shell(name string, allowedEnvVars []string) error {
	// Build env var exports for the agent user's shell
	var envExports []string
	for _, varName := range allowedEnvVars {
		if val := os.Getenv(varName); val != "" {
			envExports = append(envExports, fmt.Sprintf("%s=%s", varName, val))
		}
	}

	// Use sudo with env preservation for allowed vars
	// The command: sudo VAR1=val1 VAR2=val2 -i -u agent
	args := []string{"shell", name, "--", "sudo"}
	args = append(args, envExports...)
	args = append(args, "-i", "-u", "agent")

	cmd := exec.Command("limactl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Block dangerous environment variables from leaking via Lima's propagation
	blockedPatterns := getBlockedEnvPatterns()
	cmd.Env = filterEnv(os.Environ(), blockedPatterns)

	return cmd.Run()
}

// getBlockedEnvPatterns returns patterns of env vars to block from propagation
func getBlockedEnvPatterns() []string {
	return []string{
		"AWS_",
		"GOOGLE_",
		"AZURE_",
		"SSH_",
		"GPG_",
		"HOMEBREW_",
		"ANTHROPIC_",
		"OPENAI_",
		"_TOKEN",
		"_SECRET",
		"_KEY",
		"_PASSWORD",
		"_CREDENTIALS",
		"_API_KEY",
	}
}

// filterEnv removes environment variables matching blocked patterns
func filterEnv(env []string, blocked []string) []string {
	result := make([]string, 0, len(env))
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]

		shouldBlock := false
		for _, pattern := range blocked {
			if strings.HasPrefix(pattern, "_") {
				// Suffix pattern
				if strings.HasSuffix(key, pattern) {
					shouldBlock = true
					break
				}
			} else {
				// Prefix pattern
				if strings.HasPrefix(key, pattern) {
					shouldBlock = true
					break
				}
			}
		}

		if !shouldBlock {
			result = append(result, e)
		}
	}
	return result
}
