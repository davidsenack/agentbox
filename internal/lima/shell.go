package lima

import (
	"os"
	"os/exec"
	"strings"
)

// Shell opens an interactive shell in the Lima VM as the 'agent' user
// No secrets are passed - API keys are injected by the host proxy
func (m *Manager) Shell(name string) error {
	// Use sudo -i -u agent to get a login shell as the agent user
	// Agent uses zsh with oh-my-zsh, starts in /workspace
	cmd := exec.Command("limactl", "shell", name, "--", "sudo", "-i", "-u", "agent")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Block dangerous environment variables from leaking
	// Even though we don't inject secrets, block them from Lima's default propagation
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
