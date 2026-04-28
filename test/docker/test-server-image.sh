#!/usr/bin/env bash
set -euo pipefail

IMAGE="cgate:test-${$}"
CONTAINER="cgate-test-${$}"
HOST_PORT=18080

cleanup() {
    docker rm -f "$CONTAINER" 2>/dev/null || true
    docker rmi "$IMAGE" 2>/dev/null || true
}
trap cleanup EXIT

if ! docker info >/dev/null 2>&1; then
    echo "SKIP: Docker daemon not available"
    exit 0
fi

echo "--- Building server image ---"
docker build -t "$IMAGE" .

echo "--- Starting server container ---"
docker run -d --name "$CONTAINER" \
    -p "${HOST_PORT}:8080" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    "$IMAGE"

echo "--- Waiting for server to listen on :${HOST_PORT} ---"
MAX_WAIT=30
for i in $(seq 1 "$MAX_WAIT"); do
    if curl -sf "http://localhost:${HOST_PORT}/api/tasks" >/dev/null 2>&1; then
        echo "Server ready (${i}s)"
        break
    fi
    if [ "$i" -eq "$MAX_WAIT" ]; then
        echo "FAIL: server did not respond within ${MAX_WAIT}s"
        docker logs "$CONTAINER" 2>&1 || true
        exit 1
    fi
    sleep 1
done

echo "--- Verifying API response ---"
STATUS=$(curl -sf -o /dev/null -w '%{http_code}' "http://localhost:${HOST_PORT}/api/tasks")
if [ "$STATUS" != "200" ]; then
    echo "FAIL: GET /api/tasks returned ${STATUS}, expected 200"
    exit 1
fi
echo "PASS: GET /api/tasks -> ${STATUS}"

echo ""
echo "All server image tests passed."
