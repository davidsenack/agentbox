package cmd

import (
	"fmt"

	"github.com/davidsenack/agentbox/internal/config"
	"github.com/davidsenack/agentbox/internal/lima"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop an AgentBox VM",
	Long: `Stop an AgentBox VM without destroying it.

This command stops the VM but preserves its state. The next 'enter'
command will start the VM again.

Example:
  agentbox stop myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if project exists
	if !config.Exists(name) {
		return fmt.Errorf("project %q does not exist (no agentbox.yaml found)", name)
	}

	mgr := lima.NewManager()
	vmName := lima.VMName(name)

	if !mgr.Exists(vmName) {
		return fmt.Errorf("Lima VM %q does not exist", vmName)
	}

	running, err := mgr.IsRunning(vmName)
	if err != nil {
		return fmt.Errorf("failed to check VM status: %w", err)
	}

	if !running {
		fmt.Printf("VM %s is already stopped\n", vmName)
		return nil
	}

	fmt.Printf("Stopping VM: %s\n", vmName)
	if err := mgr.Stop(vmName); err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	fmt.Printf("VM stopped successfully\n")
	return nil
}
