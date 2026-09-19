#!/usr/bin/env bash
set -e

# Default variables
MESSAGE=""
VERSION=""
DRY_RUN=false

# Display usage instructions
usage() {
    cat <<EOF
Usage: $0 -m <commit-message> -v <version-tag> [options]

Options:
  -m, --message <msg>     Commit message (e.g. "Release")
  -v, --version <tag>     Version tag (e.g. "v-1.0.0" or "v1.0.0")
  -n, --dry-run           Preview actions without making changes
  -h, --help              Show this help message

Example:
  $0 -m "Release" -v "v-1.0.0"
EOF
}

# Parse command-line arguments
while [ $# -gt 0 ]; do
    case "$1" in
        -m|--message)
            if [ -z "${2:-}" ]; then
                echo "❌ Error: -m / --message requires an argument." >&2
                exit 1
            fi
            MESSAGE="$2"
            shift 2
            ;;
        -v|--version)
            if [ -z "${2:-}" ]; then
                echo "❌ Error: -v / --version requires an argument." >&2
                exit 1
            fi
            VERSION="$2"
            shift 2
            ;;
        -n|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "❌ Error: Unknown option '$1'" >&2
            usage
            exit 1
            ;;
    esac
done

# Validate required arguments
if [ -z "$MESSAGE" ]; then
    echo "❌ Error: Commit message (-m) is required." >&2
    usage
    exit 1
fi

if [ -z "$VERSION" ]; then
    echo "❌ Error: Version tag (-v) is required." >&2
    usage
    exit 1
fi

# Ensure git is available
if ! command -v git >/dev/null 2>&1; then
    echo "❌ Error: git is not installed or not in PATH." >&2
    exit 1
fi

# Ensure inside a git repository
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "❌ Error: Not inside a Git repository." >&2
    exit 1
fi

# Change to repository root
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"

# Determine current branch
CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || git rev-parse --abbrev-ref HEAD 2>/dev/null || true)
if [ -z "$CURRENT_BRANCH" ] || [ "$CURRENT_BRANCH" = "HEAD" ]; then
    echo "❌ Error: Unable to detect current branch or in detached HEAD state." >&2
    exit 1
fi

# Determine remote (prefer configured upstream, fallback to origin)
REMOTE=$(git config "branch.${CURRENT_BRANCH}.remote" 2>/dev/null || true)
if [ -z "$REMOTE" ]; then
    REMOTE="origin"
fi

if ! git remote get-url "$REMOTE" >/dev/null 2>&1; then
    echo "❌ Error: Git remote '$REMOTE' does not exist." >&2
    exit 1
fi

# Check if tag already exists locally
if git rev-parse -q --verify "refs/tags/${VERSION}" >/dev/null 2>&1; then
    echo "❌ Error: Tag '${VERSION}' already exists locally." >&2
    exit 1
fi

echo "=========================================="
echo "⚡ Starting publish workflow"
echo "  Branch:  ${CURRENT_BRANCH}"
echo "  Remote:  ${REMOTE}"
echo "  Version: ${VERSION}"
echo "  Message: ${MESSAGE}"
if [ "$DRY_RUN" = true ]; then
    echo "  Mode:    DRY RUN (no changes will be applied)"
fi
echo "=========================================="

if [ "$DRY_RUN" = true ]; then
    if [ -n "$(git status --porcelain)" ]; then
        echo "🔍 [Dry Run] Would stage and commit modified/untracked files:"
        git status --short
        echo "🔍 [Dry Run] Would run: git commit -m \"${MESSAGE}\""
    else
        echo "ℹ️  [Dry Run] Working tree is clean. Would tag HEAD directly."
    fi
    echo "🔍 [Dry Run] Would push branch: git push ${REMOTE} ${CURRENT_BRANCH}"
    echo "🔍 [Dry Run] Would create tag: git tag -a \"${VERSION}\" -m \"${MESSAGE}\""
    echo "🔍 [Dry Run] Would push tag: git push ${REMOTE} ${VERSION}"
    echo "=========================================="
    echo "✅ Dry run complete."
    exit 0
fi

# Check for uncommitted changes
if [ -n "$(git status --porcelain)" ]; then
    echo "📦 Staging and committing changes..."
    git add -A
    git commit -m "${MESSAGE}"
else
    echo "ℹ️  No uncommitted changes detected. Using current HEAD commit."
fi

# Push current branch to remote
echo "🚀 Pushing branch '${CURRENT_BRANCH}' to '${REMOTE}'..."
git push "${REMOTE}" "${CURRENT_BRANCH}"

# Create annotated tag
echo "🏷️  Creating annotated tag '${VERSION}'..."
git tag -a "${VERSION}" -m "${MESSAGE}"

# Push tag to remote
echo "🚀 Pushing tag '${VERSION}' to '${REMOTE}'..."
if ! git push "${REMOTE}" "${VERSION}"; then
    echo "❌ Failed to push tag '${VERSION}' to '${REMOTE}'." >&2
    echo "💡 You can remove the local tag with: git tag -d ${VERSION}" >&2
    exit 1
fi

echo "=========================================="
echo "✅ Successfully published ${VERSION} to ${CURRENT_BRANCH} on ${REMOTE}!"
echo "=========================================="
