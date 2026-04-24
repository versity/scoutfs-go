#!/bin/bash
#
# generate-defs.sh - Regenerate scoutfsdefs.go from scoutfs kernel headers
#
# This script uses Docker to build a Linux container with the scoutfs
# kernel headers installed, then runs "go tool cgo -godefs" to generate
# the Go type definitions from c_defs_linux.go.
#
# Usage:
#   ./generate-defs.sh [BRANCH]
#
# Arguments:
#   BRANCH  The scoutfs git branch to use for headers (default: main)
#
# Examples:
#   ./generate-defs.sh                      # use main branch
#   ./generate-defs.sh zab/get_changed_inos # use a feature branch
#

set -euo pipefail

BRANCH="${1:-main}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
IMAGE_NAME="scoutfs-godefs"

echo "==> Building Docker image with scoutfs branch: ${BRANCH}"
docker build \
    --build-arg "SCOUTFS_BRANCH=${BRANCH}" \
    --no-cache \
    -f "${SCRIPT_DIR}/Dockerfile.godefs" \
    -t "${IMAGE_NAME}" \
    "${SCRIPT_DIR}"

echo "==> Generating scoutfsdefs.go"
docker run --rm \
    -v "${SCRIPT_DIR}:/output" \
    "${IMAGE_NAME}"

echo "==> Verifying build"
cd "${SCRIPT_DIR}"
GOOS=linux go vet ./...
GOOS=linux go build ./...

echo "==> Done. scoutfsdefs.go has been regenerated from branch: ${BRANCH}"
