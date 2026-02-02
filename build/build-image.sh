#!/bin/bash
set -euo pipefail

# =============================================================================
# AgentBox Image Builder
# Uses Lima to build a pre-provisioned disk image
#
# NOTE: This script must be run locally on macOS with Lima installed.
# GitHub Actions runners don't support the required virtualization.
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_NAME="agentbox-build-$$"
OUTPUT_DIR="${SCRIPT_DIR}/output"
ARCH="${1:-$(uname -m)}"

# Normalize arch names
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    ARCH="arm64"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

echo "=========================================="
echo "AgentBox Image Builder"
echo "Architecture: $ARCH"
echo "=========================================="

# Check for Lima
if ! command -v limactl &> /dev/null; then
    echo "Error: Lima is not installed. Install with: brew install lima"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Create temporary Lima config for building
# Use VZ (Apple Virtualization.framework) for best performance on macOS
LIMA_CONFIG=$(mktemp)
cat > "$LIMA_CONFIG" << EOF
# Temporary Lima config for building AgentBox image
vmType: "vz"
vmOpts:
  vz:
    rosetta:
      enabled: true
      binfmt: true

cpus: 4
memory: "8GiB"
disk: "30GiB"

images:
  - location: "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-arm64.img"
    arch: "aarch64"
  - location: "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img"
    arch: "x86_64"

mounts: []

provision:
  - mode: system
    script: |
$(sed 's/^/      /' "$SCRIPT_DIR/provision.sh")

containerd:
  system: false
  user: false
EOF

cleanup() {
    echo "Cleaning up..."
    limactl stop "$BUILD_NAME" 2>/dev/null || true
    limactl delete "$BUILD_NAME" --force 2>/dev/null || true
    rm -f "$LIMA_CONFIG"
}
trap cleanup EXIT

# Start Lima VM and wait for provisioning
echo "Starting build VM..."
limactl create --name="$BUILD_NAME" "$LIMA_CONFIG" --tty=false
limactl start "$BUILD_NAME"

echo "Waiting for provisioning to complete..."
# Lima will wait for provisioning automatically

# Stop the VM cleanly
echo "Stopping VM..."
limactl stop "$BUILD_NAME"

# Get the disk path
DISK_PATH="$HOME/.lima/$BUILD_NAME/diffdisk"

if [ ! -f "$DISK_PATH" ]; then
    echo "Error: Disk not found at $DISK_PATH"
    exit 1
fi

# Copy and compress the disk
OUTPUT_FILE="$OUTPUT_DIR/agentbox-ubuntu-24.04-$ARCH.qcow2"
echo "Copying disk image..."
cp "$DISK_PATH" "$OUTPUT_FILE"

echo "Compressing disk image..."
# Convert to compressed qcow2
qemu-img convert -c -O qcow2 "$OUTPUT_FILE" "$OUTPUT_FILE.tmp"
mv "$OUTPUT_FILE.tmp" "$OUTPUT_FILE"

# Create checksum
echo "Creating checksum..."
shasum -a 256 "$OUTPUT_FILE" > "$OUTPUT_FILE.sha256"

# Get final size
SIZE=$(ls -lh "$OUTPUT_FILE" | awk '{print $5}')

echo ""
echo "=========================================="
echo "Build Complete!"
echo "=========================================="
echo "Image: $OUTPUT_FILE"
echo "Size: $SIZE"
echo "Checksum: $(cat "$OUTPUT_FILE.sha256")"
echo "=========================================="
