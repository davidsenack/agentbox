package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/davidsenack/agentbox/internal/config"
	"github.com/davidsenack/agentbox/internal/lima"
	"github.com/spf13/cobra"
)

var (
	forceDelete bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an AgentBox completely",
	Long: `Delete an AgentBox sandbox and all its files.

This command:
  1. Stops the VM if running
  2. Destroys the VM and its disk
  3. Deletes the GitHub repository (if created with --github or --gastown)
  4. Removes any Gas Town rig (if gt is installed)
  5. Deletes the entire project directory (including workspace and artifacts)

WARNING: This is destructive and cannot be undone!

Example:
  agentbox delete myproject
  agentbox delete myproject --force  # Skip confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "skip confirmation prompt")
}

func runDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if project exists
	if !config.Exists(name) {
		return fmt.Errorf("project %q does not exist", name)
	}

	// Confirm deletion unless --force is used
	if !forceDelete {
		fmt.Printf("This will permanently delete '%s' including all workspace files.\n", name)
		fmt.Print("Are you sure? [y/N] ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	mgr := lima.NewManager()
	vmName := lima.VMName(name)

	// Stop and delete VM if it exists
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

		fmt.Printf("Deleting VM: %s\n", vmName)
		if err := mgr.Delete(vmName); err != nil {
			return fmt.Errorf("failed to delete VM: %w", err)
		}
	}

	// Try to delete GitHub repo if it exists
	// Check if gh CLI is available and user is authenticated
	if _, err := exec.LookPath("gh"); err == nil {
		authCheck := exec.Command("gh", "auth", "status")
		if authCheck.Run() == nil {
			// Check if repo exists by trying to view it
			viewCmd := exec.Command("gh", "repo", "view", name, "--json", "name")
			if viewCmd.Run() == nil {
				fmt.Printf("Deleting GitHub repository: %s\n", name)
				deleteCmd := exec.Command("gh", "repo", "delete", name, "--yes")
				if err := deleteCmd.Run(); err != nil {
					fmt.Printf("Warning: failed to delete GitHub repo: %v\n", err)
				}
			}
		}
	}

	// Try to remove Gas Town rig if gt is available
	if _, err := exec.LookPath("gt"); err == nil {
		// Check if this is a Gas Town rig by looking for .beads directory
		beadsDir := filepath.Join(name, ".beads")
		if _, err := os.Stat(beadsDir); err == nil {
			fmt.Printf("Removing Gas Town rig: %s\n", name)
			removeCmd := exec.Command("gt", "rig", "remove", name, "--force")
			if err := removeCmd.Run(); err != nil {
				// Not a fatal error - we'll delete the directory anyway
				fmt.Printf("Warning: failed to remove Gas Town rig: %v\n", err)
			}
		}
	}

	// Delete project directory
	fmt.Printf("Removing project directory: %s\n", name)
	if err := os.RemoveAll(name); err != nil {
		return fmt.Errorf("failed to delete project directory: %w", err)
	}

	fmt.Printf("\nAgentBox '%s' deleted.\n", name)

	return nil
}
