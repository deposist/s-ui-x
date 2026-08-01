#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
verifier="$root_dir/frontend/scripts/verify-dist-assets.mjs"

node "$verifier" "$root_dir/web/html"
(
    cd "$root_dir"
    go test ./web -run 'TestEmbedded(ProductionAssetsMatchEmbedFS|BuiltJavaScriptAssetServesWithMime)$'
)

echo 'embedded asset verification regression test passed'
