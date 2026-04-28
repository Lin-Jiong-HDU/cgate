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
MAX_TURNS="${MAX_TURNS:-15}"

clean_title=$(echo "$ISSUE_TITLE" | sed 's/\[claude bot\]//' | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//;s/-$//')
slug=$(echo "$clean_title" | cut -c1-40 | sed 's/-$//')
branch="feat/issue-${ISSUE_NUMBER}-${slug}"

export ISSUE_NUMBER ISSUE_TITLE ISSUE_BODY ISSUE_URL REPOSITORY branch

git config --global user.name "$GIT_USER_NAME"
git config --global user.email "$GIT_USER_EMAIL"
git config --global credential.helper store
echo "https://x-access-token:${GITHUB_TOKEN}@github.com" > ~/.git-credentials
chmod 600 ~/.git-credentials

echo "$GITHUB_TOKEN" | gh auth login --with-token

repo_url="https://x-access-token:${GITHUB_TOKEN}@github.com/${REPOSITORY}.git"
git clone "$repo_url" /workspace/repo
cd /workspace/repo

git checkout main
git pull origin main
git checkout -b "$branch"

prompt=$(envsubst < /prompt-template.txt)

claude --max-turns "$MAX_TURNS" --prompt "$prompt"

git push -u origin "$branch"

pr_title=$(echo "$ISSUE_TITLE" | sed 's/[[:space:]]*\[claude bot\]//')
gh pr create \
    --title "$pr_title" \
    --body "Closes #${ISSUE_NUMBER}

Automated implementation by CGate.

Changes:
$(git log --oneline main..HEAD)" \
    --base main \
    --head "$branch"
