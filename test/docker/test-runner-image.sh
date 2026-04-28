#!/usr/bin/env bash
set -euo pipefail

IMAGE="claude-code-runner:test-${$}"

cleanup() {
    docker rmi "$IMAGE" 2>/dev/null || true
}
trap cleanup EXIT

if ! docker info >/dev/null 2>&1; then
    echo "SKIP: Docker daemon not available"
    exit 0
fi

echo "--- Building runner image ---"
docker build -t "$IMAGE" ./runner-image/

echo "--- Checking CLI tools ---"
docker run --rm --entrypoint bash "$IMAGE" -c '
errors=0

check() {
    if command -v "$1" >/dev/null 2>&1; then
        ver=$("$1" --version 2>&1 | head -1)
        echo "  PASS: $1 ($ver)"
    else
        echo "  FAIL: $1 not found in PATH"
        errors=$((errors + 1))
    fi
}

check gh
check golangci-lint
check node
check npm
check claude

if [ "$errors" -gt 0 ]; then
    echo "FAIL: $errors tool(s) missing"
    exit 1
fi
'

echo "--- Checking entrypoint.sh ---"
docker run --rm --entrypoint bash "$IMAGE" -c '
if [ ! -f /entrypoint.sh ]; then
    echo "FAIL: /entrypoint.sh does not exist"
    exit 1
fi
if [ ! -x /entrypoint.sh ]; then
    echo "FAIL: /entrypoint.sh is not executable"
    exit 1
fi
echo "  PASS: /entrypoint.sh exists and is executable"
'

echo "--- Verifying entrypoint rejects missing env vars ---"
if docker run --rm "$IMAGE" 2>/dev/null; then
    echo "FAIL: entrypoint should exit with error when env vars are missing"
    exit 1
fi
echo "  PASS: entrypoint correctly rejects missing env vars"

echo ""
echo "All runner image tests passed."
