#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
guard="$root_dir/scripts/check-official-module.sh"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

cat > "$tmp_dir/official.mod" <<'EOF'
module github.com/deposist/s-ui-x

go 1.26.4

require github.com/sagernet/sing-box v1.13.15

replace github.com/quic-go/quic-go => github.com/quic-go/quic-go v0.57.1
EOF

assert_accepts() {
    if ! "$guard" "$1" >/dev/null; then
        printf 'official module fixture was rejected: %s\n' "$1" >&2
        exit 1
    fi
}

assert_rejects() {
    if "$guard" "$1" >/dev/null 2>&1; then
        printf 'forbidden module fixture was accepted: %s\n' "$1" >&2
        exit 1
    fi
}

assert_accepts "$tmp_dir/official.mod"

cat > "$tmp_dir/extended-module.mod" <<'EOF'
module github.com/deposist/s-ui-x

go 1.26.4

require github.com/sagernet/sing-box v1.13.15
replace github.com/sagernet/sing-box => github.com/deposist/sing-box-extended v1.13.15-extended-1
EOF
assert_rejects "$tmp_dir/extended-module.mod"

cat > "$tmp_dir/shtorm-module.mod" <<'EOF'
module github.com/deposist/s-ui-x

go 1.26.4

require github.com/sagernet/sing-box v1.13.15
require github.com/shtorm-7/sing v0.8.11
EOF
assert_rejects "$tmp_dir/shtorm-module.mod"

cat > "$tmp_dir/extended-version.mod" <<'EOF'
module github.com/deposist/s-ui-x

go 1.26.4

require github.com/sagernet/sing-box v1.13.15-extended-1
EOF
assert_rejects "$tmp_dir/extended-version.mod"

cat > "$tmp_dir/extended-replacement.mod" <<'EOF'
module github.com/deposist/s-ui-x

go 1.26.4

require github.com/sagernet/sing-box v1.13.15
replace github.com/quic-go/quic-go => github.com/quic-go/quic-go v0.57.1-extended-1
EOF
assert_rejects "$tmp_dir/extended-replacement.mod"

cat > "$tmp_dir/extended-project.mod" <<'EOF'
module github.com/deposist/s-ui-x-extended

go 1.26.4

require github.com/sagernet/sing-box v1.13.15
EOF
assert_rejects "$tmp_dir/extended-project.mod"

cat > "$tmp_dir/wrong-sing-box.mod" <<'EOF'
module github.com/deposist/s-ui-x

go 1.26.4

require github.com/sagernet/sing-box v1.13.14
EOF
assert_rejects "$tmp_dir/wrong-sing-box.mod"

printf 'official module guard regression test passed\n'
