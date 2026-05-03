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

clean_title=$(echo "$ISSUE_TITLE" | sed 's/\[claude bot\]//' | tr '[:upper:]' ' ' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//;s/-$//')
slug=$(echo "$clean_title" | cut -c1-40 | sed 's/-$//')
branch="feat/issue-${ISSUE_NUMBER}-${slug}"

git config --global user.name "$GIT_USER_NAME"
git config --global user.email "$GIT_USER_EMAIL"
git config --global credential.helper store
git config --global http.version HTTP/1.1
git config --global --add safe.directory /workspace/repo
echo "https://x-access-token:${GITHUB_TOKEN}@github.com" > ~/.git-credentials
chmod 600 ~/.git-credentials

echo "$GITHUB_TOKEN" | gh auth login --with-token || true

repo_url="https://x-access-token:${GITHUB_TOKEN}@github.com/${REPOSITORY}.git"
for i in 1 2 3 4 5; do git clone "$repo_url" /workspace/repo && break || (rm -rf /workspace/repo && sleep 10); done
cd /workspace/repo

git checkout main
for i in 1 2 3; do git pull origin main && break || sleep 5; done
git checkout -b "$branch"

envsubst < /prompt-template.txt > /tmp/prompt.txt

claude_args=(-p "$(cat /tmp/prompt.txt)")
if [ "${SKIP_PERMISSIONS:-}" = "true" ]; then
    claude_args+=(--dangerously-skip-permissions)
fi

# Create non-root user for claude (root not allowed with --dangerously-skip-permissions)
if [ "$(id -u)" = "0" ]; then
    id -u runner &>/dev/null || useradd -m -d /home/runner -s /bin/bash runner
    cp -r /root/.gitconfig /home/runner/.gitconfig 2>/dev/null || true
    cp -r /root/.git-credentials /home/runner/.git-credentials 2>/dev/null || true
    cp -r /root/.config /home/runner/.config 2>/dev/null || true
    chown -R runner:runner /home/runner /workspace /tmp/prompt.txt

    # Write a launcher script that includes git push + PR creation
    export branch
    cat > /tmp/run-claude.sh <<'SCRIPT'
#!/bin/bash
cd /workspace/repo
git config --global --add safe.directory /workspace/repo
claude -p "$(cat /tmp/prompt.txt)" --dangerously-skip-permissions

echo "=== Pushing branch ==="
for i in 1 2 3 4 5; do git push -u origin "${branch}" && break || sleep 10; done

echo "=== Creating PR ==="
pr_title=$(echo "${ISSUE_TITLE}" | sed 's/[[:space:]]*\[claude bot\]//')
gh pr create \
    --title "$pr_title" \
    --body "Closes #${ISSUE_NUMBER}

Automated implementation by CGate.

Changes:
$(git log --oneline main..HEAD)" \
    --base main \
    --head "${branch}" || echo "PR already exists or creation skipped"
SCRIPT
    chmod +x /tmp/run-claude.sh
    chown runner:runner /tmp/run-claude.sh
    su -s /bin/bash runner -c /tmp/run-claude.sh
else
    claude "${claude_args[@]}"

    for i in 1 2 3 4 5; do git push -u origin "$branch" && break || sleep 10; done

    pr_title=$(echo "$ISSUE_TITLE" | sed 's/[[:space:]]*\[claude bot\]//')
    gh pr create \
        --title "$pr_title" \
        --body "Closes #${ISSUE_NUMBER}

Automated implementation by CGate.

Changes:
$(git log --oneline main..HEAD)" \
        --base main \
        --head "$branch" || echo "PR already exists or creation skipped"
fi
