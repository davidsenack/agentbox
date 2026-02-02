package lima

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Manager handles Lima VM lifecycle operations
type Manager struct {
	limaHome string
}

// NewManager creates a new Lima manager
func NewManager() *Manager {
	home, _ := os.UserHomeDir()
	return &Manager{
		limaHome: filepath.Join(home, ".lima"),
	}
}

// VMName returns the Lima VM name for a project
func VMName(projectName string) string {
	// Sanitize: replace spaces and special chars
	name := strings.ReplaceAll(projectName, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	return "agentbox-" + name
}

// Create creates a new Lima VM from the given template
func (m *Manager) Create(name string, templatePath string) error {
	cmd := exec.Command("limactl", "create", "--name", name, templatePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Start starts a Lima VM
func (m *Manager) Start(name string) error {
	cmd := exec.Command("limactl", "start", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Stop stops a Lima VM
func (m *Manager) Stop(name string) error {
	cmd := exec.Command("limactl", "stop", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Delete deletes a Lima VM
func (m *Manager) Delete(name string) error {
	cmd := exec.Command("limactl", "delete", "--force", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Exists checks if a Lima VM exists
func (m *Manager) Exists(name string) bool {
	instanceDir := filepath.Join(m.limaHome, name)
	_, err := os.Stat(instanceDir)
	return err == nil
}

// limaInstance represents a Lima instance in JSON output
type limaInstance struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// IsRunning checks if a Lima VM is running
func (m *Manager) IsRunning(name string) (bool, error) {
	cmd := exec.Command("limactl", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list Lima instances: %w", err)
	}

	// Parse JSON lines (one per instance)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var instance limaInstance
		if err := json.Unmarshal([]byte(line), &instance); err != nil {
			continue
		}
		if instance.Name == name {
			return instance.Status == "Running", nil
		}
	}

	return false, nil
}
