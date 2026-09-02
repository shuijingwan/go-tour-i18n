#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
# shellcheck source=production-identity.sh
source "$script_dir/production-identity.sh"
# shellcheck source=shared-assets-public-verification.sh
source "$script_dir/shared-assets-public-verification.sh"

usage() { printf 'usage: %s <validated-shared-assets-export>\n' "${0##*/}" >&2; }
(( $# == 1 )) || { usage; exit 2; }
export_dir=$1
[[ -d $export_dir && ! -L $export_dir && -f $export_dir/SHA256SUMS && ! -L $export_dir/SHA256SUMS ]] || {
    shared_assets_public_error 'export must be a real directory with SHA256SUMS'
    exit 1
}
load_production_identity_shared || { shared_assets_public_error 'formal shared production identity is invalid'; exit 1; }
SHARED_ASSETS_PUBLIC_BASE_URL=$PRODUCTION_SHARED_ASSETS_PUBLIC_ORIGIN
SHARED_ASSETS_NETWORK_SSH_HOST=$PRODUCTION_ZGOCLOUD_SSH_ALIAS
trap shared_assets_cleanup_network_ssh EXIT
shared_assets_setup_network_ssh
shared_assets_verify_public_assets "$export_dir/SHA256SUMS"
shared_assets_verify_boundaries
printf 'SHARED ASSETS PUBLIC VERIFICATION: PASSED\n'
