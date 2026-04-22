#!/bin/sh
# Exports OAuth credentials from GitHub repository variables into the current shell.
#
# Usage:
#   eval "$(./scripts/load-credentials.sh)"
#   make wolf-build   # or make publish
#
# First-time setup (run once per credential set):
#   gh variable set OAUTH_CLIENT_ID_INERD       --body "..."
#   gh variable set OAUTH_CLIENT_SECRET_INERD   --body "..."
#   gh variable set OAUTH_CLIENT_ID_MAROMBA     --body "..."
#   gh variable set OAUTH_CLIENT_SECRET_MAROMBA --body "..."
#   gh variable set OAUTH_CLIENT_ID_REACT       --body "..."
#   gh variable set OAUTH_CLIENT_SECRET_REACT   --body "..."
#   gh variable set OAUTH_CLIENT_ID_GEN_LANG    --body "..."
#   gh variable set OAUTH_CLIENT_SECRET_GEN_LANG --body "..."

set -e

vars=$(gh variable list --json name,value \
  -q '.[] | select(.name | startswith("OAUTH_")) | "export \(.name)=\(.value)"')

if [ -z "$vars" ]; then
  echo "⚠ No OAUTH_ variables found in GitHub repo." >&2
  echo "  Run 'gh variable set OAUTH_CLIENT_ID_INERD --body ...' etc. first." >&2
  exit 1
fi

echo "$vars"
