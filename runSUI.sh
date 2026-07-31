#!/bin/sh

set -eu

repo_dir=$(unset CDPATH; cd -- "$(dirname -- "$0")" && pwd)

sh "$repo_dir/build.sh"
cd "$repo_dir"
SUI_DB_FOLDER="db" SUI_DEBUG=true ./sui
