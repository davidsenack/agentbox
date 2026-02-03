# AgentBox

**Isolated Linux sandboxes for AI agents on macOS.**

AgentBox creates secure, isolated Linux VMs where AI coding assistants (Claude Code, Cursor, Aider, etc.) can run with full capabilities while your host machine's secrets remain protected. Your SSH keys, AWS credentials, and API tokens never enter the sandbox - even when the agent has full network access.

**Key features:**
- 🔒 **Host secrets protected** - SSH keys, AWS creds, API keys never enter the VM
- 🌐 **Full network access** - Agents can clone repos, install packages, deploy code
- 🔑 **Transparent API auth** - Anthropic API key injected by proxy, invisible to agent
- ⚡ **Fast startup** - Pre-built images boot in under 30 seconds
- 🛠️ **Batteries included** - Node, Python, Go, Claude Code, and more pre-installed
- 🏭 **Gas Town integration** - Optional multi-agent workspace support

## Quick Start

```bash
# Install (requires Lima, optionally gh for GitHub integration)
brew install lima gh
go install github.com/davidsenack/agentbox/cmd/agentbox@latest

# Create and enter a sandbox
agentbox create myproject
agentbox enter myproject

# Or create with a GitHub repo for the project
agentbox create myproject --github
agentbox enter myproject

# Inside the sandbox - full network access
[agentbox] agent@lima:~$ git clone https://github.com/...  # Works
[agentbox] agent@lima:~$ curl https://api.anthropic.com   # Works (auth injected)
[agentbox] agent@lima:~$ aws s3 ls                        # Works (if you provide creds)
[agentbox] agent@lima:~$ exit

# Reset to clean state (preserves workspace)
agentbox reset myproject
```

## Security Model

**The core protection: Your host secrets never enter the VM.**

```
┌─────────────────────────────────────────────────────────────────┐
│                         macOS Host                              │
│                                                                 │
│  ~/.ssh/id_rsa        ─── NEVER MOUNTED ───                    │
│  ~/.aws/credentials   ─── NEVER MOUNTED ───                    │
│  Keychain             ─── NOT ACCESSIBLE ───                   │
│                                                                 │
│  ANTHROPIC_API_KEY ──┐                                         │
│                      ▼                                         │
│  ┌─────────────┐    ┌──────────────┐    ┌────────────────────┐ │
│  │  agentbox   │───▶│    Proxy     │◀───│   Lima VM (vz)     │ │
│  │    CLI      │    │ (injects     │    │  ┌──────────────┐  │ │
│  └─────────────┘    │  API key)    │    │  │ agent user   │  │ │
│         │           └──────────────┘    │  │              │  │ │
│         │                               │  │ NO API KEY!  │  │ │
│         ▼                               │  │ NO SSH KEYS! │  │ │
│  ┌─────────────┐                        │  │ NO AWS CREDS │  │ │
│  │ myproject/  │                        │  └──────────────┘  │ │
│  │ workspace/  │◀──────────────────────▶│     /workspace     │ │
│  └─────────────┘                        └────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### What's Protected

| Secret | Protection |
|--------|------------|
| SSH keys (~/.ssh) | Not mounted - agent can't access |
| AWS credentials (~/.aws) | Not mounted - agent can't access |
| Keychain | Not accessible from VM |
| ANTHROPIC_API_KEY | Injected by proxy - never in VM |
| Other env secrets | Blocked from propagating to VM |

### What's Allowed

| Capability | Status |
|------------|--------|
| Full network access | ✓ Open |
| Push to GitHub | ✓ (bring your own deploy key) |
| Deploy to AWS | ✓ (bring scoped credentials) |
| Call Claude API | ✓ (auth auto-injected by proxy) |

## How API Key Injection Works

The ANTHROPIC_API_KEY never enters the VM. Instead:

1. You set `ANTHROPIC_API_KEY` on your host
2. When you run `agentbox enter`, the proxy starts
3. The VM's traffic goes through the proxy
4. When the proxy sees requests to `api.anthropic.com`, it injects the `x-api-key` header
5. The agent gets responses but never sees the actual key

**Even if malicious code runs `env` or `printenv`, the API key isn't there.**

## Commands

| Command | Description |
|---------|-------------|
| `agentbox create <name>` | Create a new sandbox project |
| `agentbox create <name> --github` | Create project with a private GitHub repo |
| `agentbox create <name> --github --public` | Create project with a public GitHub repo |
| `agentbox create <name> --gastown` | Create as a Gas Town rig (implies --github) |
| `agentbox enter <name>` | Enter the sandbox (starts VM + proxy) |
| `agentbox stop <name>` | Stop the VM without destroying it |
| `agentbox reset <name>` | Destroy VM and recreate (preserves workspace) |
| `agentbox delete <name>` | Delete project completely (VM + all files) |
| `agentbox delete <name> -f` | Force delete without confirmation |
| `agentbox list` | List projects in current directory |

## Configuration

Each project has an `agentbox.yaml` configuration file:

```yaml
runtime: lima

vm:
  cpus: 4
  memory: "4GiB"
  disk: "30GiB"

network:
  proxy_port: 3128
  # Proxy injects auth for these hosts - secrets stay on host
  inject_auth:
    - host: api.anthropic.com
      header: x-api-key
      env: ANTHROPIC_API_KEY

secrets:
  # Env vars to pass to the VM (for tools that need API keys)
  allowed_env_vars:
    - ANTHROPIC_API_KEY
  # Patterns to redact from logs
  redact_patterns:
    - "sk-ant-[a-zA-Z0-9-]+"

mounts:
  - host: "./workspace"
    guest: "/workspace"
    writable: true
  - host: "./artifacts"
    guest: "/artifacts"
    writable: true
```

## Working with Git/GitHub

Since your SSH keys aren't in the VM, you have options:

### Option 1: HTTPS with token
```bash
# Generate a fine-grained PAT for just this repo
# In the VM:
git config --global credential.helper store
git clone https://github.com/you/repo.git
# Enter token when prompted (it's stored in VM only)
```

### Option 2: Deploy key
```bash
# Generate a key IN the VM (stays in VM)
ssh-keygen -t ed25519 -f ~/.ssh/deploy_key
# Add the public key to your repo as a deploy key
```

### Option 3: Temporary key injection
```bash
# Copy a scoped key to workspace before entering
cp ~/.ssh/repo_specific_key myproject/workspace/.ssh/
# In VM, it's available at /workspace/.ssh/
```

## Pre-installed Software

AgentBox VMs come ready to work with these tools pre-installed.

> **Note:** On first VM creation, provisioning installs all software automatically. This takes a few minutes on the initial boot, but subsequent starts are fast. You can also build pre-provisioned images locally for instant boot (see [Building Pre-provisioned Images](#building-pre-provisioned-images)).

**Shell & Terminal:**
- zsh with Oh My Zsh
- Custom Starship prompt: `agent@agentbox/path branch +!?`
  - Shows git branch and status (staged, modified, untracked)
- zsh-autosuggestions (suggests commands as you type)
- zsh-syntax-highlighting (colors valid/invalid commands)
- tmux
- fzf (fuzzy finder)

**Languages & Runtimes:**
- Node.js 22 LTS + npm
- Python 3 + pip + venv
- Go 1.22
- mise (version manager for additional runtimes)

**Development Tools:**
- git
- neovim (with sensible defaults)
- ripgrep (fast grep)
- jq (JSON processor)
- build-essential (gcc, make)

**AI Coding Tools:**
- claude-code (`claude` command)
- opencode (`opencode` command)
- Gas Town (`gt` and `bd` commands)

## Installing Additional Packages

Inside the sandbox, you have full sudo access:

```bash
# System packages (apt)
sudo apt-get update
sudo apt-get install <package>

# Node packages (global)
npm install -g <package>

# Python packages
pip install <package>
# Or use a venv:
python3 -m venv .venv && source .venv/bin/activate

# Go tools
go install <package>@latest

# Use mise for additional language versions
mise use node@20
mise use python@3.11
mise use go@1.21
```

**Note:** Installed packages persist until you run `agentbox reset`. The reset command destroys the VM but preserves your `/workspace` files.

## Project Structure

After `agentbox create myproject`:

```
myproject/
├── agentbox.yaml      # Configuration
├── .agentbox/         # Runtime state (gitignored)
│   ├── lima.yaml      # Generated Lima template
│   └── network.log    # Network access log
├── workspace/         # Your code (mounted to /workspace)
└── artifacts/         # Output files (mounted to /artifacts)
```

## Threat Model

### What AgentBox Protects Against

- **Host secret theft**: SSH keys, AWS creds, tokens never enter VM
- **API key exfiltration**: Anthropic key injected by proxy, never visible in VM
- **Host filesystem damage**: Only workspace/artifacts are mounted
- **Persistence after reset**: VM state destroyed, only workspace survives

### What AgentBox Does NOT Protect Against

- **Secrets you put in workspace**: If you copy credentials there, they're accessible
- **Scoped credentials you provide**: Deploy keys, tokens you give the agent can be used/exfiltrated
- **Data exfiltration via network**: Agent can send your code anywhere (network is open)
- **VM escape exploits**: Mitigated by Apple Virtualization.framework, but not guaranteed
- **Denial of service**: Agent can fill disk/CPU within VM

### Key Insight

AgentBox protects your **ambient credentials** - the keys and tokens that exist on your host and would normally be accessible to any process. It doesn't prevent an agent from misusing credentials you explicitly provide.

## Requirements

- macOS (Apple Silicon recommended)
- [Lima](https://lima-vm.io/) (`brew install lima`)
- [GitHub CLI](https://cli.github.com/) (`brew install gh`) - optional, for `--github` flag
- Go 1.21+ (for building from source)

## Building from Source

```bash
git clone https://github.com/davidsenack/agentbox
cd agentbox
go build -o agentbox ./cmd/agentbox
```

## Building Pre-provisioned Images

For faster VM creation, you can build pre-provisioned disk images locally:

```bash
# Clone the repo
git clone https://github.com/davidsenack/agentbox
cd agentbox

# Build image for your architecture
cd build
./build-image.sh arm64   # For Apple Silicon Macs
./build-image.sh amd64   # For Intel Macs

# Images are created in build/output/
ls output/
# agentbox-ubuntu-24.04-arm64.qcow2
# agentbox-ubuntu-24.04-arm64.qcow2.sha256
```

Pre-provisioned images reduce first-boot time from several minutes to under 30 seconds.

To use a pre-built image, either:
1. Upload it to a GitHub release (the template will auto-detect)
2. Modify `internal/lima/template.go` to point to your image location

## Claude Code Authentication

AgentBox supports two authentication methods for Claude Code:

### Option 1: Max Subscription (OAuth Login)

If you have a Claude Max subscription, log in once inside the sandbox:

```bash
agentbox enter myproject
# Inside the sandbox:
claude                     # First run opens browser for OAuth login
# After login, claude works until you reset the VM
```

The OAuth token stays inside the VM - your host credentials are never exposed.

### Option 2: API Key (Secure Injection)

If you have an Anthropic API key, set it on your host:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."   # On your host (add to ~/.zshrc)
agentbox enter myproject
```

The key is securely injected but **never visible** inside the VM:

```bash
# Inside the VM:
echo $ANTHROPIC_API_KEY    # Shows nothing
env | grep ANTHROPIC       # Shows nothing
cat /etc/agentbox/secrets/*  # Permission denied

# But claude-code works!
claude "hello"             # Works - key is injected by secure wrapper
```

**How it works:**
1. Key is stored in `/etc/agentbox/secrets/` (root-only, mode 600)
2. A wrapper script reads the key via sudo and injects it only for the claude process
3. The key never appears in shell environment or process listings

Configure which keys to inject in `agentbox.yaml`:
```yaml
secrets:
  allowed_env_vars:
    - ANTHROPIC_API_KEY
    - OPENAI_API_KEY  # Add more as needed
```

## License

MIT License - see [LICENSE](LICENSE) for details.
