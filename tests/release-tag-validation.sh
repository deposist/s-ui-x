#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
validator="$root_dir/scripts/validate-release-tag.sh"
checkout_verifier="$root_dir/scripts/verify-release-tag-checkout.sh"

for tag in v1.2.3 v2.0.0-rc.1 v1.5.2-beta-hotfix2 v3.4.5-hotfix1 v3.4.5-preview.7; do
    test "$("$validator" "$tag")" = "$tag"
done
test "$("$validator" --prerelease v1.2.3)" = false
for tag in v2.0.0-rc.1 v1.5.2-beta-hotfix2 v3.4.5-hotfix1 v3.4.5-preview.7; do
    test "$("$validator" --prerelease "$tag")" = true
done

invalid_tags=(
    '1.2.3' 'v1.2' 'v01.2.3' 'v1.2.3-01' 'v1.2.3-RC.1'
    'v1.2.3 quoted' 'v1.2.3"' 'v1.2.3;echo injected'
    "v1.2.3\$(echo injected)" $'v1.2.3\necho injected'
)
for tag in "${invalid_tags[@]}"; do
    if "$validator" "$tag" >/dev/null 2>&1; then
        printf 'accepted invalid release tag: %q\n' "$tag" >&2
        exit 1
    fi
done
if "$validator" v1.2.3 unexpected >/dev/null 2>&1; then
    echo 'accepted unexpected release-tag argument' >&2
    exit 1
fi

for workflow in release.yml windows.yml docker.yml; do
    path="$root_dir/.github/workflows/$workflow"
    grep -q 'uses: ./.github/actions/checkout-release-tag' "$path"
    grep -qF "event_commit: \${{ github.event_name == 'push' && github.sha || '' }}" "$path"
done
grep -qF "ref: refs/tags/\${{ steps.input.outputs.tag }}" "$root_dir/.github/actions/checkout-release-tag/action.yml"
grep -qF 'fetch-depth: 0' "$root_dir/.github/actions/checkout-release-tag/action.yml"

# Exercise the verifier against lightweight and annotated tags, moved tags, and
# a tag whose config/version no longer matches.
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT
mkdir -p "$tmp_dir/config"
git -C "$tmp_dir" init --quiet
git -C "$tmp_dir" config user.name release-test
git -C "$tmp_dir" config user.email release-test@example.invalid
printf '1.0.0\n' > "$tmp_dir/config/version"
git -C "$tmp_dir" add config/version
git -C "$tmp_dir" commit --quiet -m v1
git -C "$tmp_dir" tag v1.0.0
old_commit=$(git -C "$tmp_dir" rev-parse HEAD)
printf '2.0.0\n' > "$tmp_dir/config/version"
git -C "$tmp_dir" commit --quiet -am v2
git -C "$tmp_dir" tag -a v2.0.0 -m annotated
new_commit=$(git -C "$tmp_dir" rev-parse HEAD)
(
    cd "$tmp_dir"
    "$checkout_verifier" v2.0.0 "$new_commit" "$new_commit" >/dev/null
    if "$checkout_verifier" v2.0.0 "$old_commit" >/dev/null 2>&1; then
        echo 'moved release tag was accepted' >&2
        exit 1
    fi
    if "$checkout_verifier" v2.0.0 "$new_commit" "$old_commit" >/dev/null 2>&1; then
        echo 'event commit mismatch was accepted' >&2
        exit 1
    fi
    if "$checkout_verifier" v1.0.0 >/dev/null 2>&1; then
        echo 'non-HEAD release tag was accepted' >&2
        exit 1
    fi
)

echo 'release tag validation regression test passed'
