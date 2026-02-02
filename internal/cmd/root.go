package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	verbose     bool
	versionStr  string
	commitStr   string
	buildTimeStr string
)

// SetVersion sets the version information
func SetVersion(version, commit, buildTime string) {
	versionStr = version
	commitStr = commit
	buildTimeStr = buildTime
}

var rootCmd = &cobra.Command{
	Use:   "agentbox",
	Short: "Terminal-first sandbox for AI agents",
	Long: `AgentBox creates isolated Linux VMs for safe AI agent execution on macOS.

The sandbox provides:
  - Full filesystem isolation (only workspace/artifacts mounted)
  - Host secret protection (SSH keys, AWS creds, API keys never in VM)
  - API key injection via proxy (keys stay on host)
  - Open network (agent can reach any host)
  - Easy reset to clean state`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("agentbox %s\n", versionStr)
		fmt.Printf("  commit: %s\n", commitStr)
		fmt.Printf("  built:  %s\n", buildTimeStr)
	},
}

// Execute runs the root command
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(enterCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(versionCmd)
}
