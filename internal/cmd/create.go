package cmd

import (
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
	createGitHub  bool
	createPublic  bool
	createGasTown bool
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new AgentBox project",
	Long: `Create a new AgentBox project with isolated workspace.

This command:
  1. Creates the project directory structure
  2. Generates default agentbox.yaml configuration
  3. Provisions a Lima VM (stopped)
  4. Optionally creates a GitHub repo for the project
  5. Optionally registers as a Gas Town rig

Example:
  agentbox create myproject
  agentbox create myproject --github           # Create with private GitHub repo
  agentbox create myproject --github --public  # Create with public GitHub repo
  agentbox create myproject --gastown          # Create as Gas Town rig (implies --github)`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().BoolVar(&createGitHub, "github", false, "Create a GitHub repository for the project")
	createCmd.Flags().BoolVar(&createPublic, "public", false, "Make the GitHub repo public (default: private)")
	createCmd.Flags().BoolVar(&createGasTown, "gastown", false, "Register as a Gas Town rig (implies --github)")
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

	// Create .gitignore for the box
	gitignore := filepath.Join(name, ".gitignore")
	gitignoreContent := `.agentbox/
`
	if err := os.WriteFile(gitignore, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Gas Town implies GitHub
	if createGasTown {
		createGitHub = true
	}

	// Create GitHub repo if requested
	var repoURL string
	if createGitHub {
		var err error
		repoURL, err = setupGitHubRepo(name)
		if err != nil {
			return fmt.Errorf("failed to setup GitHub repo: %w", err)
		}
	}

	// Register as Gas Town rig if requested
	if createGasTown {
		if err := setupGasTownRig(name, repoURL); err != nil {
			return fmt.Errorf("failed to setup Gas Town rig: %w", err)
		}
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

// setupGitHubRepo creates a GitHub repo and initializes git in the workspace
// Returns the repo URL for use with Gas Town
func setupGitHubRepo(name string) (string, error) {
	workspacePath := filepath.Join(name, "workspace")

	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh CLI not found. Install with: brew install gh")
	}

	// Check if user is authenticated
	authCheck := exec.Command("gh", "auth", "status")
	if err := authCheck.Run(); err != nil {
		return "", fmt.Errorf("not authenticated with GitHub. Run: gh auth login")
	}

	fmt.Printf("Creating GitHub repository: %s\n", name)

	// Initialize git in workspace first
	gitInit := exec.Command("git", "init")
	gitInit.Dir = workspacePath
	if err := gitInit.Run(); err != nil {
		return "", fmt.Errorf("failed to initialize git: %w", err)
	}

	// Create the GitHub repo
	args := []string{"repo", "create", name, "--source", workspacePath}
	if createPublic {
		args = append(args, "--public")
	} else {
		args = append(args, "--private")
	}

	cmd := exec.Command("gh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create GitHub repo: %w", err)
	}

	// Create a .gitignore in workspace
	workspaceGitignore := filepath.Join(workspacePath, ".gitignore")
	gitignoreContent := `# OS files
.DS_Store
Thumbs.db

# Editor files
*.swp
*.swo
*~
.idea/
.vscode/

# Build artifacts
*.o
*.a
*.so
*.dylib

# Dependencies (uncomment as needed)
# node_modules/
# vendor/
# __pycache__/
# *.pyc
`
	if err := os.WriteFile(workspaceGitignore, []byte(gitignoreContent), 0644); err != nil {
		return "", fmt.Errorf("failed to create workspace .gitignore: %w", err)
	}

	// Create initial README in workspace
	readmePath := filepath.Join(workspacePath, "README.md")
	readmeContent := fmt.Sprintf("# %s\n\nProject created with [AgentBox](https://github.com/davidsenack/agentbox).\n", name)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return "", fmt.Errorf("failed to create README: %w", err)
	}

	// Stage and commit the initial files
	gitAdd := exec.Command("git", "add", ".")
	gitAdd.Dir = workspacePath
	if err := gitAdd.Run(); err != nil {
		return "", fmt.Errorf("failed to stage files: %w", err)
	}

	gitCommit := exec.Command("git", "commit", "-m", "Initial commit\n\nCreated with AgentBox")
	gitCommit.Dir = workspacePath
	if err := gitCommit.Run(); err != nil {
		// Ignore if nothing to commit
		if !strings.Contains(err.Error(), "nothing to commit") {
			return "", fmt.Errorf("failed to commit: %w", err)
		}
	}

	// Push to remote
	gitPush := exec.Command("git", "push", "-u", "origin", "main")
	gitPush.Dir = workspacePath
	gitPush.Stdout = os.Stdout
	gitPush.Stderr = os.Stderr
	if err := gitPush.Run(); err != nil {
		fmt.Printf("Warning: failed to push to remote: %v\n", err)
		fmt.Println("You can push manually later with: git push -u origin main")
	}

	// Get the repo URL (SSH format for gt rig add)
	getURL := exec.Command("gh", "repo", "view", name, "--json", "sshUrl", "-q", ".sshUrl")
	urlOutput, err := getURL.Output()
	repoURL := strings.TrimSpace(string(urlOutput))
	if err != nil || repoURL == "" {
		// Fallback to HTTPS URL
		getURL = exec.Command("gh", "repo", "view", name, "--json", "url", "-q", ".url")
		urlOutput, _ = getURL.Output()
		repoURL = strings.TrimSpace(string(urlOutput))
	}

	if repoURL != "" {
		fmt.Printf("GitHub repo created: %s\n", repoURL)
	}

	return repoURL, nil
}

// setupGasTownRig registers the project as a Gas Town rig
func setupGasTownRig(name, repoURL string) error {
	// Check if gt CLI is available
	if _, err := exec.LookPath("gt"); err != nil {
		return fmt.Errorf("gt CLI not found. Install Gas Town first")
	}

	if repoURL == "" {
		return fmt.Errorf("no repository URL available for Gas Town rig")
	}

	fmt.Printf("Registering Gas Town rig: %s\n", name)

	// Run gt rig add
	cmd := exec.Command("gt", "rig", "add", name, repoURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to register Gas Town rig: %w", err)
	}

	fmt.Printf("Gas Town rig registered: %s\n", name)
	return nil
}
