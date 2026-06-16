#!/usr/bin/env bash
# Run a command inside the Debian 13 build container with the project mounted.
# Usage: build/docker.sh <command...>
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${ATHENA_IMAGE:-athena-build:deb13}"

if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
    echo "=== Building image $IMAGE ==="
    docker build -t "$IMAGE" -f "$ROOT/build/Dockerfile.debian13" "$ROOT/build"
fi

exec docker run --rm -v "$ROOT:/work" -w /work "$IMAGE" -c "$*"
