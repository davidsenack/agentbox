#!/bin/bash
set -euo pipefail

# =============================================================================
# AgentBox Image Provisioning Script
# This script is used by Packer to build the base image
# =============================================================================

export DEBIAN_FRONTEND=noninteractive

echo "=========================================="
echo "AgentBox Image Build - Starting"
echo "=========================================="

# --- System Update ---
echo "Updating system packages..."
apt-get update
apt-get upgrade -y

# --- System Packages ---
echo "Installing system packages..."
apt-get install -y \
    zsh \
    build-essential \
    curl \
    wget \
    git \
    jq \
    ripgrep \
    fzf \
    tmux \
    neovim \
    unzip \
    ca-certificates \
    gnupg \
    sudo \
    openssh-server

# --- Node.js (via NodeSource for latest LTS) ---
echo "Installing Node.js..."
mkdir -p /etc/apt/keyrings
curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_22.x nodistro main" > /etc/apt/sources.list.d/nodesource.list
apt-get update
apt-get install -y nodejs

# --- Python 3 ---
echo "Installing Python..."
apt-get install -y python3 python3-pip python3-venv

# --- Go ---
echo "Installing Go..."
GO_VERSION="1.22.0"
ARCH=$(dpkg --print-architecture)
curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" | tar -C /usr/local -xzf -
echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
chmod +x /etc/profile.d/go.sh

# --- Starship Prompt ---
echo "Installing Starship..."
curl -fsSL https://starship.rs/install.sh | sh -s -- -y

# --- mise (version manager) ---
echo "Installing mise..."
export HOME=/root
curl -fsSL https://mise.run | sh
mv /root/.local/bin/mise /usr/local/bin/mise 2>/dev/null || true

# --- Create agent user ---
echo "Creating agent user..."
useradd -m -s /bin/zsh -G sudo agent
echo "agent ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/agent
chmod 0440 /etc/sudoers.d/agent

# --- Agent directories ---
sudo -u agent mkdir -p /home/agent/.local/bin
sudo -u agent mkdir -p /home/agent/.config/nvim
sudo -u agent mkdir -p /home/agent/go/bin

# --- Oh My Zsh ---
echo "Installing Oh My Zsh..."
sudo -u agent sh -c 'RUNZSH=no CHSH=no sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"'

# --- Agent .zshrc ---
cat > /home/agent/.zshrc << 'ZSHRC'
# AgentBox zsh configuration
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="robbyrussell"
plugins=(git fzf)
source $ZSH/oh-my-zsh.sh

# Environment
export PATH="$HOME/.local/bin:/usr/local/go/bin:$HOME/go/bin:$PATH"
export GOPATH="$HOME/go"

# Proxy (for API key injection) - set at runtime by agentbox enter
if [ -f /etc/agentbox/proxy.conf ]; then
    export $(grep -v '^#' /etc/agentbox/proxy.conf | xargs)
fi

# Starship prompt
eval "$(starship init zsh)"

# mise (version manager)
if command -v mise &> /dev/null; then
    eval "$(mise activate zsh)"
fi

# Start in workspace
cd /workspace 2>/dev/null || true
ZSHRC

chown agent:agent /home/agent/.zshrc

# --- Neovim Config ---
cat > /home/agent/.config/nvim/init.lua << 'NVIMCONFIG'
-- AgentBox Neovim configuration
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.expandtab = true
vim.opt.tabstop = 4
vim.opt.shiftwidth = 4
vim.opt.smartindent = true
vim.opt.wrap = false
vim.opt.cursorline = true
vim.opt.termguicolors = true
vim.opt.signcolumn = "yes"
vim.opt.clipboard = "unnamedplus"
vim.opt.ignorecase = true
vim.opt.smartcase = true

-- Key mappings
vim.g.mapleader = " "
vim.keymap.set("n", "<leader>w", ":w<CR>")
vim.keymap.set("n", "<leader>q", ":q<CR>")
NVIMCONFIG

chown -R agent:agent /home/agent/.config

# --- Install AI Tools ---
echo "Installing AI coding tools..."

# claude-code (install globally as root)
npm install -g @anthropic-ai/claude-code || echo "Warning: claude-code installation failed"

# opencode (install for agent user)
sudo -u agent bash -c 'export PATH="/usr/local/go/bin:$PATH"; export GOPATH="$HOME/go"; /usr/local/go/bin/go install github.com/opencode-ai/opencode@latest' || echo "Warning: opencode installation failed"

# --- Cleanup ---
echo "Cleaning up..."
apt-get clean
rm -rf /var/lib/apt/lists/*
rm -rf /tmp/*
rm -rf /var/tmp/*

# --- Create marker file ---
echo "AgentBox Base Image" > /etc/agentbox-image
date >> /etc/agentbox-image

echo "=========================================="
echo "AgentBox Image Build - Complete"
echo "=========================================="
echo "Installed: zsh, oh-my-zsh, starship, tmux"
echo "Installed: node $(node --version), python3, go"
echo "Installed: neovim, ripgrep, fzf, jq, mise"
echo "Installed: claude-code, opencode"
echo "=========================================="
