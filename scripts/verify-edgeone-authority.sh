#!/usr/bin/env bash

set -Eeuo pipefail

script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
[[ $# == 1 ]] || { printf 'usage: %s <live-edgeone-locale>\n' "${0##*/}" >&2; exit 2; }
export PYTHONDONTWRITEBYTECODE=1
exec python3 "$script_dir/production-cdn.py" preflight --locale "$1"
