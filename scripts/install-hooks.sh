#!/usr/bin/env bash
# Point git at the tracked .githooks directory so the shared pre-commit hook runs.
# Run once per clone, from the repo root:  bash scripts/install-hooks.sh
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
git config core.hooksPath .githooks
chmod +x .githooks/* 2>/dev/null || true
echo "Hooks installed: core.hooksPath = .githooks"
echo "The pre-commit hook now rejects commits containing merge conflict markers."
