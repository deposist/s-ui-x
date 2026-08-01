#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

real_cp=$(command -v cp)
real_mv=$(command -v mv)
fixture="$tmp_dir/repo"
fake_bin="$tmp_dir/bin"
mkdir -p "$fixture/frontend" "$fixture/web/html" "$fake_bin"
"$real_cp" "$root_dir/build.sh" "$fixture/build.sh"

cat > "$fake_bin/node" <<'EOF'
#!/bin/sh
exit 0
EOF

cat > "$fake_bin/go" <<'EOF'
#!/bin/sh
if [ "$1" = env ] && [ "$2" = GOARCH ]; then
  printf 'amd64\n'
fi
exit 0
EOF

cat > "$fake_bin/npm" <<'EOF'
#!/bin/sh
set -eu
case "$*" in
  ci)
    exit 0
    ;;
  "run build")
    if [ "${FAIL_FRONTEND_BUILD:-0}" = 1 ]; then
      exit 42
    fi
    rm -rf dist
    mkdir -p dist/assets
    printf '<!doctype html>\n' > dist/index.html
    printf 'new bundle\n' > dist/assets/app.js
    ;;
  "run verify:dist")
    test -f dist/index.html
    test -f dist/assets/app.js
    ;;
  *)
    printf 'unexpected npm command: %s\n' "$*" >&2
    exit 64
    ;;
esac
EOF

cat > "$fake_bin/cp" <<EOF
#!/bin/sh
case "\$*" in
  *'.html-stage.'*)
    if [ "\${FAIL_STAGE_COPY:-0}" = 1 ]; then
      exit 43
    fi
    ;;
esac
exec "$real_cp" "\$@"
EOF

cat > "$fake_bin/mv" <<EOF
#!/bin/sh
if [ "\${FAIL_STAGE_INSTALL:-0}" = 1 ]; then
  case "\$1|\$2" in
    *'.html-stage.'*'|'*/web/html)
      exit 44
      ;;
  esac
fi
exec "$real_mv" "\$@"
EOF
chmod +x "$fixture/build.sh" "$fake_bin/node" "$fake_bin/go" "$fake_bin/npm" "$fake_bin/cp" "$fake_bin/mv"

reset_last_good_assets() {
  rm -rf "$fixture/frontend/dist" "$fixture/web/html"
  mkdir -p "$fixture/web/html/assets"
  printf '<!doctype html>last good\n' > "$fixture/web/html/index.html"
  printf 'last good bundle\n' > "$fixture/web/html/assets/last-good.js"
}

assert_last_good_assets() {
  grep -q 'last good' "$fixture/web/html/index.html"
  grep -q 'last good bundle' "$fixture/web/html/assets/last-good.js"
  if find "$fixture/web" -maxdepth 1 -type d \( -name '.html-stage.*' -o -name '.html-backup.*' \) | grep -q .; then
    echo 'build left a staging or backup directory behind' >&2
    exit 1
  fi
}

reset_last_good_assets
if PATH="$fake_bin:$PATH" FAIL_FRONTEND_BUILD=1 "$fixture/build.sh" >/dev/null 2>&1; then
  echo 'build unexpectedly succeeded after frontend build failure' >&2
  exit 1
fi
assert_last_good_assets

reset_last_good_assets
if PATH="$fake_bin:$PATH" FAIL_STAGE_COPY=1 "$fixture/build.sh" >/dev/null 2>&1; then
  echo 'build unexpectedly succeeded after staged asset copy failure' >&2
  exit 1
fi
assert_last_good_assets

reset_last_good_assets
if PATH="$fake_bin:$PATH" FAIL_STAGE_INSTALL=1 "$fixture/build.sh" >/dev/null 2>&1; then
  echo 'build unexpectedly succeeded after staged asset install failure' >&2
  exit 1
fi
assert_last_good_assets

reset_last_good_assets
PATH="$fake_bin:$PATH" "$fixture/build.sh" >/dev/null

grep -q 'new bundle' "$fixture/web/html/assets/app.js"
if [ -e "$fixture/web/html/assets/last-good.js" ]; then
  echo 'successful replacement retained a stale frontend asset' >&2
  exit 1
fi
if find "$fixture/web" -maxdepth 1 -type d \( -name '.html-stage.*' -o -name '.html-backup.*' \) | grep -q .; then
  echo 'successful build left a staging or backup directory behind' >&2
  exit 1
fi

printf 'build asset replacement regression test passed\n'
