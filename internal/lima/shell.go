package lima

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Shell opens an interactive shell in the Lima VM as the 'agent' user
// allowedEnvVars specifies which env vars to inject securely (stored in root-only files)
func (m *Manager) Shell(name string, allowedEnvVars []string) error {
	// Inject secrets securely - write to root-only files, not env vars
	// This way `echo $ANTHROPIC_API_KEY` shows nothing
	if err := injectSecrets(name, allowedEnvVars); err != nil {
		return fmt.Errorf("failed to inject secrets: %w", err)
	}

	// Start the shell without any secrets in environment
	args := []string{"shell", name, "--", "sudo", "-i", "-u", "agent"}

	cmd := exec.Command("limactl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Block dangerous environment variables from leaking via Lima's propagation
	blockedPatterns := getBlockedEnvPatterns()
	cmd.Env = filterEnv(os.Environ(), blockedPatterns)

	return cmd.Run()
}

// injectSecrets writes allowed env vars to secure root-only files in the VM
// The claude wrapper script reads from these files
func injectSecrets(vmName string, allowedEnvVars []string) error {
	for _, varName := range allowedEnvVars {
		val := os.Getenv(varName)
		if val == "" {
			continue
		}

		// Write to /etc/agentbox/secrets/<varname> with mode 600, owned by root
		// The wrapper scripts read from here using sudo
		script := fmt.Sprintf(`
mkdir -p /etc/agentbox/secrets
echo -n %q > /etc/agentbox/secrets/%s
chmod 600 /etc/agentbox/secrets/%s
chown root:root /etc/agentbox/secrets/%s
`, val, varName, varName, varName)

		cmd := exec.Command("limactl", "shell", vmName, "--", "sudo", "sh", "-c", script)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to inject %s: %w", varName, err)
		}
	}
	return nil
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
