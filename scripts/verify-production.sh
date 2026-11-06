#!/usr/bin/env bash
# Compatibility entrypoint; current route authority and all HTTP checks are
# shared by first, repeat and batch adapters.
set -Eeuo pipefail
script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
export PYTHONDONTWRITEBYTECODE=1
exec python3 "$script_dir/verify-production.py" "$@"
