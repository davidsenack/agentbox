package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidsenack/agentbox/internal/config"
	"github.com/davidsenack/agentbox/internal/lima"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List AgentBox projects",
	Long: `List all AgentBox projects in the current directory.

Shows project name, VM status, and configuration summary.`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	// Find agentbox.yaml files in current directory
	entries, err := os.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	mgr := lima.NewManager()
	found := false

	fmt.Println("AgentBox Projects:")
	fmt.Println()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		configPath := filepath.Join(name, config.ConfigFileName)
		if _, err := os.Stat(configPath); err != nil {
			continue
		}

		found = true
		vmName := lima.VMName(name)
		status := "not created"

		if mgr.Exists(vmName) {
			running, err := mgr.IsRunning(vmName)
			if err != nil {
				status = "unknown"
			} else if running {
				status = "running"
			} else {
				status = "stopped"
			}
		}

		fmt.Printf("  %s\n", name)
		fmt.Printf("    VM: %s (%s)\n", vmName, status)
	}

	if !found {
		fmt.Println("  (no projects found)")
		fmt.Println()
		fmt.Println("Create one with: agentbox create <name>")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
}
