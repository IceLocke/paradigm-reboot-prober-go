#!/bin/sh
# Setup git authentication for private submodules if GH_TOKEN is available,
# then initialise / update submodules.
#
# Usage:
#   GH_TOKEN=ghp_xxx pnpm install   (or set as Vercel / CI env var)
#
# On Vercel the automatic clone stage tries to fetch submodules WITHOUT
# environment variables, leaving them in a broken half-initialised state.
# We therefore clean up first so that `git submodule update` actually
# re-clones with the correct credentials.

set -e

if [ -n "$GH_TOKEN" ]; then
  git config --global url."https://x-access-token:${GH_TOKEN}@github.com/".insteadOf "https://github.com/"
fi

# Clean up any half-initialised submodule state left by a prior failed clone
# (e.g. Vercel's automatic submodule fetch that runs without GH_TOKEN).
git submodule deinit -f --all 2>/dev/null || true
rm -rf .git/modules/web/public 2>/dev/null || true

git submodule update --init --recursive || echo "Submodule checkout skipped (private repo requires GH_TOKEN)"
