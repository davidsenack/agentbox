package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidsenack/agentbox/internal/config"
	"github.com/davidsenack/agentbox/internal/lima"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new AgentBox project",
	Long: `Create a new AgentBox project with isolated workspace.

This command:
  1. Creates the project directory structure
  2. Generates default agentbox.yaml configuration
  3. Provisions a Lima VM (stopped)

Example:
  agentbox create myproject
  cd myproject
  agentbox enter myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if directory already exists
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}

	fmt.Printf("Creating AgentBox project: %s\n", name)

	// Create directory structure
	dirs := []string{
		name,
		filepath.Join(name, ".agentbox"),
		filepath.Join(name, "workspace"),
		filepath.Join(name, "artifacts"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create .gitkeep files
	for _, dir := range []string{"workspace", "artifacts"} {
		gitkeep := filepath.Join(name, dir, ".gitkeep")
		if err := os.WriteFile(gitkeep, []byte{}, 0644); err != nil {
			return fmt.Errorf("failed to create .gitkeep: %w", err)
		}
	}

	// Create default configuration
	cfg := config.DefaultConfig()
	if err := config.Save(name, cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Create .gitignore
	gitignore := filepath.Join(name, ".gitignore")
	gitignoreContent := `.agentbox/
`
	if err := os.WriteFile(gitignore, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Get absolute path for Lima template
	absPath, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Generate Lima template
	limaTemplate, err := lima.GenerateTemplate(cfg, absPath)
	if err != nil {
		return fmt.Errorf("failed to generate Lima template: %w", err)
	}

	templatePath := filepath.Join(name, ".agentbox", "lima.yaml")
	if err := os.WriteFile(templatePath, []byte(limaTemplate), 0600); err != nil {
		return fmt.Errorf("failed to write Lima template: %w", err)
	}

	// Create Lima VM
	mgr := lima.NewManager()
	vmName := lima.VMName(name)

	if mgr.Exists(vmName) {
		fmt.Printf("Lima VM %q already exists, skipping creation\n", vmName)
	} else {
		fmt.Printf("Creating Lima VM: %s\n", vmName)
		if err := mgr.Create(vmName, templatePath); err != nil {
			return fmt.Errorf("failed to create Lima VM: %w", err)
		}
	}

	fmt.Printf("\nAgentBox project created successfully!\n\n")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", name)
	fmt.Printf("  agentbox enter %s\n", name)

	return nil
}
