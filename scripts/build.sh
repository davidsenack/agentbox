#!/bin/bash
set -euo pipefail

# Build script for AgentBox

cd "$(dirname "$0")/.."

VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_TIME="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

echo "Building AgentBox..."
echo "  Version: ${VERSION}"
echo "  Commit:  ${COMMIT}"
echo "  Time:    ${BUILD_TIME}"

LDFLAGS="-s -w"
LDFLAGS="${LDFLAGS} -X main.version=${VERSION}"
LDFLAGS="${LDFLAGS} -X main.commit=${COMMIT}"
LDFLAGS="${LDFLAGS} -X main.buildTime=${BUILD_TIME}"

# Build for current platform
go build -ldflags "${LDFLAGS}" -o agentbox ./cmd/agentbox

echo "Built: ./agentbox"
./agentbox --help | head -5
