#!/bin/sh
# Setup git authentication for private submodules if GH_TOKEN is available.
# Usage:
#   GH_TOKEN=ghp_xxx pnpm install   (or set in .env)

if [ -n "$GH_TOKEN" ]; then
  git config --global url."https://x-access-token:${GH_TOKEN}@github.com/".insteadOf "https://github.com/"
fi

git submodule update --init --remote || echo "Submodule checkout skipped (private repo requires GH_TOKEN)"

if [ -n "$GH_TOKEN" ]; then
  git config --global url."https://github.com/".insteadOf "https://x-access-token:${GH_TOKEN}@github.com/"
fi