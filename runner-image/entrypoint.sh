#!/usr/bin/env bash
set -euo pipefail

: "${REPOSITORY:?REPOSITORY is required}"
: "${GITHUB_TOKEN:?GITHUB_TOKEN is required}"
: "${ANTHROPIC_API_KEY:?ANTHROPIC_API_KEY is required}"

TASK_TYPE="${TASK_TYPE:-issue}"
ISSUE_BODY="${ISSUE_BODY:-}"
ISSUE_URL="${ISSUE_URL:-}"
GIT_USER_NAME="${GIT_USER_NAME:-cgate-bot}"
GIT_USER_EMAIL="${GIT_USER_EMAIL:-cgate-bot@users.noreply.github.com}"

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

# --- Issue flow (original) ---
run_issue() {
    : "${ISSUE_NUMBER:?ISSUE_NUMBER is required}"
    : "${ISSUE_TITLE:?ISSUE_TITLE is required}"

    clean_title=$(echo "$ISSUE_TITLE" | sed 's/\[claude bot\]//' | tr '[:upper:]' ' ' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//;s/-$//')
    slug=$(echo "$clean_title" | cut -c1-40 | sed 's/-$//')
    branch="feat/issue-${ISSUE_NUMBER}-${slug}"

    git checkout main
    for i in 1 2 3; do git pull origin main && break || sleep 5; done
    git checkout -b "$branch"

    envsubst '${ISSUE_NUMBER} ${ISSUE_TITLE} ${ISSUE_BODY} ${REPOSITORY} ${ISSUE_URL} ${branch}' < /prompt-template.txt > /tmp/prompt.txt

    claude_args=(-p "$(cat /tmp/prompt.txt)")
    if [ "${SKIP_PERMISSIONS:-}" = "true" ]; then
        claude_args+=(--dangerously-skip-permissions)
    fi

    if [ "$(id -u)" = "0" ]; then
        id -u runner &>/dev/null || useradd -m -d /home/runner -s /bin/bash runner
        cp -r /root/.gitconfig /home/runner/.gitconfig 2>/dev/null || true
        cp -r /root/.git-credentials /home/runner/.git-credentials 2>/dev/null || true
        cp -r /root/.config /home/runner/.config 2>/dev/null || true
        chown -R runner:runner /home/runner /workspace /tmp/prompt.txt

        export branch
        cat > /tmp/run-claude.sh <<'SCRIPT'
#!/bin/bash
cd /workspace/repo
git config --global --add safe.directory /workspace/repo
claude_args=(-p "$(cat /tmp/prompt.txt)")
if [ "${SKIP_PERMISSIONS:-}" = "true" ]; then
    claude_args+=(--dangerously-skip-permissions)
fi
claude "${claude_args[@]}"

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
}

# --- PR Review flow ---
run_pr_review() {
    : "${PR_NUMBER:?PR_NUMBER is required for pr_review task type}"

    # Get PR branch name
    pr_branch=$(gh pr view "$PR_NUMBER" --json headRefName --jq '.headRefName')
    if [ -z "$pr_branch" ]; then
        echo "ERROR: Could not determine branch for PR #$PR_NUMBER"
        exit 1
    fi

    echo "=== PR Review Fix for PR #${PR_NUMBER} on branch ${pr_branch} ==="

    # Fetch and checkout the PR branch
    git fetch origin "$pr_branch"
    git checkout "$pr_branch"
    for i in 1 2 3; do git pull origin "$pr_branch" && break || sleep 5; done

    # Fetch reviews: only CHANGES_REQUESTED and COMMENTED states, ignore PENDING and APPROVED
    reviews_json=$(gh api "repos/${REPOSITORY}/pulls/${PR_NUMBER}/reviews" \
        --jq '[.[] | select(.state == "CHANGES_REQUESTED" or .state == "COMMENTED")]' 2>/dev/null || echo '[]')

    # Fetch all inline review comments (unresolved filtering is delegated to Claude at runtime)
    comments_json=$(gh api "repos/${REPOSITORY}/pulls/${PR_NUMBER}/comments" 2>/dev/null || echo '[]')

    # Build combined review data
    cat > /workspace/reviews.json <<REVIEWEOF
{
  "pr_number": ${PR_NUMBER},
  "repository": "${REPOSITORY}",
  "branch": "${pr_branch}",
  "reviews": ${reviews_json},
  "comments": ${comments_json}
}
REVIEWEOF

    envsubst '${PR_NUMBER} ${REPOSITORY} ${pr_branch}' < /prompt-template-pr-review.txt > /tmp/prompt.txt

    claude_args=(-p "$(cat /tmp/prompt.txt)")
    if [ "${SKIP_PERMISSIONS:-}" = "true" ]; then
        claude_args+=(--dangerously-skip-permissions)
    fi

    if [ "$(id -u)" = "0" ]; then
        id -u runner &>/dev/null || useradd -m -d /home/runner -s /bin/bash runner
        cp -r /root/.gitconfig /home/runner/.gitconfig 2>/dev/null || true
        cp -r /root/.git-credentials /home/runner/.git-credentials 2>/dev/null || true
        cp -r /root/.config /home/runner/.config 2>/dev/null || true
        chown -R runner:runner /home/runner /workspace /tmp/prompt.txt

        export pr_branch PR_NUMBER REPOSITORY
        cat > /tmp/run-claude.sh <<'SCRIPT'
#!/bin/bash
cd /workspace/repo
git config --global --add safe.directory /workspace/repo
claude_args=(-p "$(cat /tmp/prompt.txt)")
if [ "${SKIP_PERMISSIONS:-}" = "true" ]; then
    claude_args+=(--dangerously-skip-permissions)
fi
claude "${claude_args[@]}"

echo "=== Pushing fixes to PR branch ==="
for i in 1 2 3 4 5; do git push origin "${pr_branch}" && break || sleep 10; done
SCRIPT
        chmod +x /tmp/run-claude.sh
        chown runner:runner /tmp/run-claude.sh
        su -s /bin/bash runner -c /tmp/run-claude.sh
    else
        claude "${claude_args[@]}"

        echo "=== Pushing fixes to PR branch ==="
        for i in 1 2 3 4 5; do git push origin "$pr_branch" && break || sleep 10; done
    fi
}

# --- Dispatch ---
if [ "$TASK_TYPE" = "pr_review" ]; then
    run_pr_review
else
    run_issue
fi
