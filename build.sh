#!/bin/sh

set -eu

repo_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
frontend_dir=$repo_dir/frontend
frontend_dist=$frontend_dir/dist
web_dir=$repo_dir/web
web_html=$web_dir/html
stage_dir=
backup_dir=

die() {
  echo "Error: $*" >&2
  exit 1
}

cleanup() {
  status=$?
  trap - EXIT HUP INT TERM

  if [ -n "$stage_dir" ] && { [ -e "$stage_dir" ] || [ -L "$stage_dir" ]; }; then
    rm -rf "$stage_dir" || :
  fi

  if [ -n "$backup_dir" ] && { [ -e "$backup_dir" ] || [ -L "$backup_dir" ]; }; then
    if { [ -e "$backup_dir/html" ] || [ -L "$backup_dir/html" ]; } && ! { [ -e "$web_html" ] || [ -L "$web_html" ]; }; then
      mv "$backup_dir/html" "$web_html" || echo "Error: failed to restore prior web assets from $backup_dir/html" >&2
    fi
    if ! { [ -e "$backup_dir/html" ] || [ -L "$backup_dir/html" ]; }; then
      rm -rf "$backup_dir" || :
    else
      echo "Warning: prior web assets were preserved at $backup_dir/html" >&2
    fi
  fi

  exit "$status"
}

cd "$frontend_dir"
npm ci
npm run build
npm run verify:dist

# Copy the verified output into a sibling directory first. Existing embedded
# assets remain untouched until the complete staging copy succeeds.
mkdir -p "$web_dir"
stage_dir=$(mktemp -d "$web_dir/.html-stage.XXXXXX") || die "failed to create frontend staging directory"
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
cp -R "$frontend_dist"/. "$stage_dir"/ || die "failed to stage frontend production assets"

if [ -e "$web_html" ] || [ -L "$web_html" ]; then
  backup_dir=$(mktemp -d "$web_dir/.html-backup.XXXXXX") || die "failed to create web asset backup directory"
  mv "$web_html" "$backup_dir/html" || die "failed to preserve prior web assets"
fi

if ! mv "$stage_dir" "$web_html"; then
  die "failed to replace embedded web assets"
fi
stage_dir=

if [ -n "$backup_dir" ]; then
  rm -rf "$backup_dir" || die "failed to remove prior web asset backup"
  backup_dir=
fi

trap - EXIT HUP INT TERM

cd "$repo_dir"
echo "Backend"

BUILD_TAGS="with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_musl,badlinkname,tfogo_checklinkname0,with_tailscale"

# Embed the release artifact platform suffix so the in-panel self-update knows
# which `s-ui-linux-<platform>.tar.gz` asset to fetch (config.ResolveArtifactPlatform).
ARCH="$(go env GOARCH)"
case "$ARCH" in
  arm) ARTIFACT_PLATFORM="armv$(go env GOARM 2>/dev/null || echo 7)" ;;
  *)   ARTIFACT_PLATFORM="$ARCH" ;;
esac
LDFLAGS="-w -s -checklinkname=0 -extldflags \"-Wl,-no_warn_duplicate_libraries\" -X github.com/deposist/s-ui-x/config.ArtifactPlatform=${ARTIFACT_PLATFORM}"
go build -ldflags "$LDFLAGS" -tags "$BUILD_TAGS" -o sui main.go
