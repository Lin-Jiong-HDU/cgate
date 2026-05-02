#!/usr/bin/env bash
set -euo pipefail

: "${REPOSITORY:?REPOSITORY is required}"
: "${ISSUE_NUMBER:?ISSUE_NUMBER is required}"
: "${ISSUE_TITLE:?ISSUE_TITLE is required}"
: "${GITHUB_TOKEN:?GITHUB_TOKEN is required}"
: "${ANTHROPIC_API_KEY:?ANTHROPIC_API_KEY is required}"

ISSUE_BODY="${ISSUE_BODY:-}"
ISSUE_URL="${ISSUE_URL:-}"
GIT_USER_NAME="${GIT_USER_NAME:-cgate-bot}"
GIT_USER_EMAIL="${GIT_USER_EMAIL:-cgate-bot@users.noreply.github.com}"

clean_title=$(echo "$ISSUE_TITLE" | sed 's/\[claude bot\]//' | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//;s/-$//')
slug=$(echo "$clean_title" | cut -c1-40 | sed 's/-$//')
branch="feat/issue-${ISSUE_NUMBER}-${slug}"

export ISSUE_NUMBER ISSUE_TITLE ISSUE_BODY ISSUE_URL REPOSITORY branch

git config --global user.name "$GIT_USER_NAME"
git config --global user.email "$GIT_USER_EMAIL"
git config --global credential.helper store
git config --global http.version HTTP/1.1
echo "https://x-access-token:${GITHUB_TOKEN}@github.com" > ~/.git-credentials
chmod 600 ~/.git-credentials

echo "$GITHUB_TOKEN" | gh auth login --with-token || true

repo_url="https://x-access-token:${GITHUB_TOKEN}@github.com/${REPOSITORY}.git"
for i in 1 2 3 4 5; do git clone "$repo_url" /workspace/repo && break || (rm -rf /workspace/repo && sleep 10); done
cd /workspace/repo

git checkout main
for i in 1 2 3; do git pull origin main && break || sleep 5; done
git checkout -b "$branch"

prompt=$(envsubst < /prompt-template.txt)

claude_args=(-p "$prompt")
if [ "${SKIP_PERMISSIONS:-}" = "true" ]; then
    claude_args+=(--dangerously-skip-permissions)
fi

# Create non-root user for claude (root not allowed with --dangerously-skip-permissions)
if [ "$(id -u)" = "0" ]; then
    id -u runner &>/dev/null || useradd -m -d /home/runner -s /bin/bash runner
    cp -r /root/.gitconfig /home/runner/.gitconfig 2>/dev/null || true
    cp -r /root/.git-credentials /home/runner/.git-credentials 2>/dev/null || true
    cp -r /root/.config /home/runner/.config 2>/dev/null || true
    chown -R runner:runner /home/runner /workspace
    su -s /bin/bash runner -c "cd /workspace/repo && claude -p $(printf '%q' "$prompt") --dangerously-skip-permissions"
else
    claude "${claude_args[@]}"
fi

for i in 1 2 3 4 5; do git push -u origin "$branch" && break || sleep 10; done

pr_title=$(echo "$ISSUE_TITLE" | sed 's/[[:space:]]*\[claude bot\]//')
gh pr create \
    --title "$pr_title" \
    --body "Closes #${ISSUE_NUMBER}

Automated implementation by CGate.

Changes:
$(git log --oneline main..HEAD)" \
    --base main \
    --head "$branch"
