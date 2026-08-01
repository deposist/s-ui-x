#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
verifier="$root_dir/frontend/scripts/verify-dist-assets.mjs"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

mkdir -p "$tmp_dir/good/assets"
printf '<script type="module" src="./assets/app.js"></script>\n' > "$tmp_dir/good/index.html"
printf 'import "./chunk.js";\n' > "$tmp_dir/good/assets/app.js"
printf 'export default 1;\n' > "$tmp_dir/good/assets/chunk.js"
printf 'body { background: url("./bg.svg"); }\n' > "$tmp_dir/good/assets/style.css"
printf '<svg xmlns="http://www.w3.org/2000/svg"/>\n' > "$tmp_dir/good/assets/bg.svg"
node "$verifier" "$tmp_dir/good"

mkdir -p "$tmp_dir/missing/assets"
printf '<script type="module" src="./assets/app.js"></script>\n' > "$tmp_dir/missing/index.html"
printf 'import "./missing.js";\n' > "$tmp_dir/missing/assets/app.js"
if node "$verifier" "$tmp_dir/missing" >/dev/null 2>&1; then
    echo 'dangling JavaScript reference was accepted' >&2
    exit 1
fi

mkdir -p "$tmp_dir/unsafe/assets"
printf '<!doctype html>\n' > "$tmp_dir/unsafe/index.html"
printf 'asset\n' > "$tmp_dir/unsafe/assets/_unsafe.js"
if node "$verifier" "$tmp_dir/unsafe" >/dev/null 2>&1; then
    echo 'unsafe embedded asset name was accepted' >&2
    exit 1
fi

printf 'dist asset reference regression test passed\n'
