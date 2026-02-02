package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidsenack/agentbox/internal/config"
	"github.com/davidsenack/agentbox/internal/lima"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset <name>",
	Short: "Reset an AgentBox to clean state",
	Long: `Reset an AgentBox sandbox to a clean state.

This command:
  1. Stops the VM if running
  2. Destroys the VM and its disk
  3. Recreates the VM from the template
  4. Preserves workspace/ and artifacts/ directories

Use this when you want to start fresh without losing your code.

Example:
  agentbox reset myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runReset,
}

func runReset(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if project exists
	if !config.Exists(name) {
		return fmt.Errorf("project %q does not exist (no agentbox.yaml found)", name)
	}

	// Load configuration
	cfg, err := config.Load(name)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	mgr := lima.NewManager()
	vmName := lima.VMName(name)

	// Stop VM if running
	if mgr.Exists(vmName) {
		running, err := mgr.IsRunning(vmName)
		if err != nil {
			fmt.Printf("Warning: failed to check VM status: %v\n", err)
		}

		if running {
			fmt.Printf("Stopping VM: %s\n", vmName)
			if err := mgr.Stop(vmName); err != nil {
				return fmt.Errorf("failed to stop VM: %w", err)
			}
		}

		// Delete VM
		fmt.Printf("Deleting VM: %s\n", vmName)
		if err := mgr.Delete(vmName); err != nil {
			return fmt.Errorf("failed to delete VM: %w", err)
		}
	}

	// Clear network log
	absPath, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	networkLogPath := filepath.Join(absPath, ".agentbox", "network.log")
	if err := os.WriteFile(networkLogPath, []byte{}, 0600); err != nil {
		fmt.Printf("Warning: failed to clear network log: %v\n", err)
	}

	// Regenerate Lima template
	limaTemplate, err := lima.GenerateTemplate(cfg, absPath)
	if err != nil {
		return fmt.Errorf("failed to generate Lima template: %w", err)
	}

	templatePath := filepath.Join(absPath, ".agentbox", "lima.yaml")
	if err := os.WriteFile(templatePath, []byte(limaTemplate), 0600); err != nil {
		return fmt.Errorf("failed to write Lima template: %w", err)
	}

	// Recreate VM
	fmt.Printf("Creating VM: %s\n", vmName)
	if err := mgr.Create(vmName, templatePath); err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	fmt.Printf("\nAgentBox reset complete!\n")
	fmt.Printf("Workspace and artifacts preserved.\n")
	fmt.Printf("\nRun 'agentbox enter %s' to continue.\n", name)

	return nil
}
