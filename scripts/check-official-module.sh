#!/usr/bin/env bash
set -euo pipefail

module_file=${1:-go.mod}
if [[ ! -f "$module_file" ]]; then
    printf 'Go module file does not exist: %s\n' "$module_file" >&2
    exit 1
fi

go mod edit -json "$module_file" >/dev/null

if ! grep -Eiq '^module[[:space:]]+github\.com/deposist/s-ui-x[[:space:]]*$' "$module_file"; then
    printf 'unexpected module path; expected github.com/deposist/s-ui-x\n' >&2
    exit 1
fi

if grep -Eiq 'sing-box-extended|github\.com/shtorm-7/|extended-' "$module_file"; then
    printf 'forbidden extended dependency, module, version, or replacement found in %s\n' "$module_file" >&2
    exit 1
fi

sing_box_count=$(grep -Ec '^[[:space:]]*(require[[:space:]]+)?github\.com/sagernet/sing-box[[:space:]]+v1\.13\.15([[:space:]]|$)' "$module_file" || true)
if [[ "$sing_box_count" != 1 ]]; then
    printf 'sing-box must be required directly at v1.13.15\n' >&2
    exit 1
fi

if grep -Eiq 'github\.com/sagernet/sing-box[[:space:]].*=>' "$module_file"; then
    printf 'sing-box replacement is forbidden\n' >&2
    exit 1
fi

printf 'official Go module guard passed: github.com/deposist/s-ui-x; sing-box v1.13.15\n'
