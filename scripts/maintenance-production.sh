#!/usr/bin/env bash

set -Eeuo pipefail

script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
export PYTHONDONTWRITEBYTECODE=1
exec python3 "$script_dir/maintenance-production.py" "$@"
