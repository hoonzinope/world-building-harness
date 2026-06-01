#!/usr/bin/env bash
set -euo pipefail

PACK_ID="${1:-}"
shift || true
REQUEST="${*:-}"

if [[ -z "$PACK_ID" || -z "$REQUEST" ]]; then
  echo "usage: scripts/codex_pack.sh <pack-id> <request...>" >&2
  exit 2
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PACK_ROOT="$REPO_ROOT/packs/$PACK_ID"

if [[ ! -f "$PACK_ROOT/harness.yaml" ]]; then
  echo "pack not found: $PACK_ID" >&2
  exit 2
fi

export WORLD_TOOL_REGISTRY="$REPO_ROOT/.worlds.yaml"
export PATH="$REPO_ROOT/bin:$PATH"

codex exec \
  -C "$REPO_ROOT" \
  --add-dir "$PACK_ROOT" \
  --sandbox danger-full-access \
  --skip-git-repo-check \
  "You are operating world-harness for pack '$PACK_ID'.

Use world-tool and the local registry at $WORLD_TOOL_REGISTRY.
Do not edit content/ directly. Create or update drafts first, validate them, and summarize draft_path plus validation status.
If the user asks for story material, create storylet or event/character/place drafts as appropriate.

Request:
$REQUEST"
