#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
checker="$root_dir/scripts/check-release-assets.sh"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

mkdir -p "$tmp_dir/bin" "$tmp_dir/assets"
printf 'alpha\n' > "$tmp_dir/assets/alpha.bin"
printf 'beta\n' > "$tmp_dir/assets/beta.bin"
printf '[]\n' > "$tmp_dir/remote.json"

cat > "$tmp_dir/bin/gh" <<'PY'
#!/usr/bin/env python
import hashlib
import json
import os
import sys

state_path = os.environ["FAKE_GH_STATE"]
args = sys.argv[1:]
if args[:1] == ["api"]:
    if any("releases/tags/" in arg for arg in args):
        print("42")
        raise SystemExit(0)
    if any("/assets?" in arg for arg in args):
        with open(state_path, encoding="utf-8") as handle:
            print(json.dumps([json.load(handle)], separators=(",", ":")))
        raise SystemExit(0)
if args[:2] == ["release", "upload"]:
    asset = args[3]
    with open(asset, "rb") as handle:
        digest = hashlib.sha256(handle.read()).hexdigest()
    with open(state_path, encoding="utf-8") as handle:
        assets = json.load(handle)
    assets.append({"name": os.path.basename(asset), "state": "uploaded", "digest": "sha256:" + digest})
    with open(state_path, "w", encoding="utf-8") as handle:
        json.dump(assets, handle)
    raise SystemExit(0)
print("unexpected fake gh invocation: " + repr(args), file=sys.stderr)
raise SystemExit(2)
PY
chmod +x "$tmp_dir/bin/gh"

cat > "$tmp_dir/bin/jq" <<'PY'
#!/usr/bin/env python
import json
import re
import sys

args = sys.argv[1:]
query = next((arg for arg in reversed(args) if not arg.startswith("-")), "")
raw = sys.stdin.read()
data = json.loads(raw) if raw.strip() else None
if query == "add":
    flattened = []
    for page in data:
        flattened.extend(page)
    print(json.dumps(flattened, separators=(",", ":")))
elif query == "all(.[]; .state == \"uploaded\" and (.digest // \"\" | test(\"^sha256:[0-9a-f]{64}$\")))":
    ok = all(item.get("state") == "uploaded" and re.fullmatch(r"sha256:[0-9a-f]{64}", item.get("digest", "")) for item in data)
    print("true" if ok else "false")
elif query == ".[]":
    for item in data:
        print(json.dumps(item, separators=(",", ":")))
elif query == ".[].name":
    for item in data:
        print(item.get("name", ""))
elif query == ".name":
    print(data.get("name", ""))
elif query == ".digest // \"\"":
    print(data.get("digest", ""))
elif query == ".state":
    print(data.get("state", ""))
else:
    print("unexpected fake jq query: " + repr(query), file=sys.stderr)
    raise SystemExit(2)
PY
chmod +x "$tmp_dir/bin/jq"

export PATH="$tmp_dir/bin:$PATH"
export GH_TOKEN=test-token
export FAKE_GH_STATE="$tmp_dir/remote.json"

# Reconcile an empty remote release, then verify the exact inventory and digests.
bash "$checker" example/repo v1.2.3 "$tmp_dir/assets"
bash "$checker" --verify example/repo v1.2.3 "$tmp_dir/assets"

# A local digest change must fail closed.
printf 'tampered\n' > "$tmp_dir/assets/alpha.bin"
if bash "$checker" --verify example/repo v1.2.3 "$tmp_dir/assets" >/dev/null 2>&1; then
    echo 'local digest tampering was accepted' >&2
    exit 1
fi
printf 'alpha\n' > "$tmp_dir/assets/alpha.bin"

# An unexpected remote asset must fail exact inventory verification.
python - "$tmp_dir/remote.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    assets = json.load(handle)
assets.append({"name": "unexpected.bin", "state": "uploaded", "digest": "sha256:" + "0" * 64})
with open(sys.argv[1], "w", encoding="utf-8") as handle:
    json.dump(assets, handle)
PY
if bash "$checker" --verify example/repo v1.2.3 "$tmp_dir/assets" >/dev/null 2>&1; then
    echo 'unexpected remote asset was accepted' >&2
    exit 1
fi

# Empty local directories are rejected before any remote operation.
mkdir "$tmp_dir/empty"
if bash "$checker" --verify example/repo v1.2.3 "$tmp_dir/empty" >/dev/null 2>&1; then
    echo 'empty release asset directory was accepted' >&2
    exit 1
fi

printf 'release asset reconciliation regression test passed\n'

# Keep workflow sequencing and publication classification explicit even when no
# GitHub Actions runner is available locally.
release_workflow="$root_dir/.github/workflows/release.yml"
python - "$release_workflow" <<'PY'
from pathlib import Path
import sys

text = Path(sys.argv[1]).read_text(encoding="utf-8")
checks = [
    ("publish waits for both build families", "needs: [release-ref, build-linux, build-windows]"),
    ("local checksums precede release reconciliation", "Verify local release checksums"),
    ("asset reconciliation precedes publication", "Reconcile release assets by authoritative digest"),
    ("publication follows exact remote verification", "Verify exact remote release assets and digests"),
    ("stable releases claim latest", "args+=(--prerelease=false --latest)"),
    ("prereleases avoid latest", "args+=(--prerelease --latest=false)"),
]
for label, needle in checks:
    if needle not in text:
        raise SystemExit(f"missing release workflow invariant: {label}")
if text.index("Verify local release checksums") > text.index("Reconcile release assets by authoritative digest"):
    raise SystemExit("release reconciliation ran before local checksum verification")
if text.index("Reconcile release assets by authoritative digest") > text.index("Publish verified release"):
    raise SystemExit("publication ran before asset reconciliation")
if text.index("Verify exact remote release assets and digests") > text.index("Publish verified release"):
    raise SystemExit("publication ran before exact remote verification")
print("release workflow semantic structure ok")
PY

printf 'release workflow and asset regression tests passed\n'
