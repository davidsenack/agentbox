package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/davidsenack/agentbox/internal/config"
	"github.com/davidsenack/agentbox/internal/lima"
	"github.com/davidsenack/agentbox/internal/proxy"
	"github.com/davidsenack/agentbox/internal/secrets"
	"github.com/spf13/cobra"
)

var enterCmd = &cobra.Command{
	Use:   "enter <name>",
	Short: "Enter an AgentBox sandbox",
	Long: `Enter an AgentBox sandbox environment.

This command:
  1. Starts the HTTP proxy (with auth injection for configured hosts)
  2. Starts the Lima VM if not running
  3. Opens an interactive shell inside the VM

API keys are injected by the proxy - they never enter the VM.
The shell starts in /workspace. Exit the shell to return to the host.

Example:
  agentbox enter myproject`,
	Args: cobra.ExactArgs(1),
	RunE: runEnter,
}

func runEnter(cmd *cobra.Command, args []string) error {
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

	absPath, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create redactor for log sanitization
	redactor := secrets.NewRedactor(cfg.Secrets.RedactPatterns)

	// Set up network log
	networkLogPath := filepath.Join(absPath, ".agentbox", "network.log")
	networkLog, err := proxy.NewLogger(networkLogPath, redactor)
	if err != nil {
		return fmt.Errorf("failed to create network logger: %w", err)
	}
	defer networkLog.Close()

	// Start proxy with auth injection
	proxyServer := proxy.New(cfg.Network.ProxyPort, cfg.Network.InjectAuth, networkLog)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := proxyServer.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Proxy error: %v\n", err)
		}
	}()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Start VM if needed
	mgr := lima.NewManager()
	vmName := lima.VMName(name)

	if !mgr.Exists(vmName) {
		return fmt.Errorf("Lima VM %q does not exist. Run 'agentbox create %s' first", vmName, name)
	}

	running, err := mgr.IsRunning(vmName)
	if err != nil {
		return fmt.Errorf("failed to check VM status: %w", err)
	}

	if !running {
		fmt.Printf("Starting VM: %s\n", vmName)
		if err := mgr.Start(vmName); err != nil {
			return fmt.Errorf("failed to start VM: %w", err)
		}
	}

	// Show auth injection info
	authHosts := make([]string, 0, len(cfg.Network.InjectAuth))
	for _, auth := range cfg.Network.InjectAuth {
		authHosts = append(authHosts, auth.Host)
	}

	fmt.Printf("Entering AgentBox: %s\n", name)
	if len(cfg.Secrets.AllowedEnvVars) > 0 {
		fmt.Printf("Passing env vars: %v\n", cfg.Secrets.AllowedEnvVars)
	}
	fmt.Println("Type 'exit' to leave the sandbox")
	fmt.Println()

	// Enter shell with allowed env vars
	if err := mgr.Shell(vmName, cfg.Secrets.AllowedEnvVars); err != nil {
		return fmt.Errorf("shell error: %w", err)
	}

	fmt.Println("\nExited AgentBox")
	return nil
}
