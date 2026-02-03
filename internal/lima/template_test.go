package lima

import (
	"strings"
	"testing"

	"github.com/davidsenack/agentbox/internal/config"
)

func TestGenerateTemplate(t *testing.T) {
	cfg := config.DefaultConfig()
	projectDir := "/tmp/testproject"

	template, err := GenerateTemplate(cfg, projectDir)
	if err != nil {
		t.Fatalf("failed to generate template: %v", err)
	}

	// Check essential components
	checks := []string{
		`vmType: "vz"`,                    // Uses Apple Virtualization
		`cpus: 4`,                         // Default CPUs
		`memory: "4GiB"`,                  // Default memory
		`mountPoint: "/workspace"`,        // Workspace mount
		`mountPoint: "/artifacts"`,        // Artifacts mount
		`propagateProxyEnv: false`,        // No env propagation
		`forwardAgent: false`,             // No SSH agent forwarding
		projectDir + "/workspace",         // Host workspace path
		projectDir + "/artifacts",         // Host artifacts path
	}

	for _, check := range checks {
		if !strings.Contains(template, check) {
			t.Errorf("template missing expected content: %q", check)
		}
	}
}

func TestGenerateTemplateGasTown(t *testing.T) {
	cfg := config.DefaultConfig()
	projectDir := "/tmp/testproject"
	rigName := "testrig"
	repoURL := "https://github.com/user/repo.git"

	template, err := GenerateTemplateGasTown(cfg, projectDir, rigName, repoURL)
	if err != nil {
		t.Fatalf("failed to generate Gas Town template: %v", err)
	}

	// Check Gas Town specific components
	checks := []string{
		rigName,                           // Rig name in script
		repoURL,                           // Repo URL in script
		"gt install",                      // HQ initialization
		"gt rig add",                      // Rig creation
		"GT_ROOT",                         // Environment variable
		"/home/agent/gt",                  // Gas Town directory
	}

	for _, check := range checks {
		if !strings.Contains(template, check) {
			t.Errorf("Gas Town template missing expected content: %q", check)
		}
	}
}

func TestGenerateTemplateNoHostMounts(t *testing.T) {
	cfg := config.DefaultConfig()
	projectDir := "/tmp/testproject"

	template, err := GenerateTemplate(cfg, projectDir)
	if err != nil {
		t.Fatalf("failed to generate template: %v", err)
	}

	// Should NOT contain home directory mounts
	if strings.Contains(template, "~") && strings.Contains(template, "location:") {
		// Check it's not a home mount (only workspace/artifacts should be mounted)
		lines := strings.Split(template, "\n")
		for _, line := range lines {
			if strings.Contains(line, "location:") && strings.Contains(line, "~") {
				t.Error("template should not mount home directory")
			}
		}
	}

	// Should only have 2 mounts (workspace and artifacts)
	mountCount := strings.Count(template, "mountPoint:")
	if mountCount != 2 {
		t.Errorf("expected 2 mounts, found %d", mountCount)
	}
}

func TestGenerateTemplateProxyConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Network.ProxyPort = 8080 // Custom port
	projectDir := "/tmp/testproject"

	template, err := GenerateTemplate(cfg, projectDir)
	if err != nil {
		t.Fatalf("failed to generate template: %v", err)
	}

	// Check proxy port is in the template
	if !strings.Contains(template, "8080") {
		t.Error("template should contain custom proxy port")
	}
}

func TestVMName(t *testing.T) {
	tests := []struct {
		project  string
		expected string
	}{
		{"myproject", "agentbox-myproject"},
		{"test", "agentbox-test"},
		{"my-app", "agentbox-my-app"},
	}

	for _, tt := range tests {
		result := VMName(tt.project)
		if result != tt.expected {
			t.Errorf("VMName(%q) = %q, want %q", tt.project, result, tt.expected)
		}
	}
}

func TestIndent(t *testing.T) {
	input := "line1\nline2\nline3"
	result := indent(4, input)

	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "    ") {
			t.Errorf("line should be indented with 4 spaces: %q", line)
		}
	}
}
