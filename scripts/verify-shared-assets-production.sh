#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

readonly RECEIPT_SCHEMA='go-tour-i18n/shared-assets-production-receipt/v1'
script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
# shellcheck source=production-identity.sh
source "$script_dir/production-identity.sh"
# shellcheck source=shared-assets-public-verification.sh
source "$script_dir/shared-assets-public-verification.sh"
unset script_dir

error() { printf '[verify-shared-assets-production] ERROR: %s\n' "$*" >&2; }
usage() { printf 'usage: %s <verification-receipt.json>\n' "${0##*/}" >&2; }

parse_receipt() {
    local receipt=$1
    python3 - "$receipt" <<'PY'
import json
import sys

required = {
    "schema", "export_dir", "manifest_sha256", "deployment_result",
    "production_base_url", "changed_paths", "boundary_paths",
}
try:
    with open(sys.argv[1], encoding="utf-8") as source:
        receipt = json.load(source)
except (OSError, json.JSONDecodeError) as exc:
    raise SystemExit(f"invalid JSON receipt: {exc}")
if type(receipt) is not dict or set(receipt) != required:
    raise SystemExit("receipt must contain exactly the required object keys")
for key in required - {"changed_paths", "boundary_paths"}:
    if type(receipt[key]) is not str:
        raise SystemExit(f"receipt field {key} must be a string")
for key in ("changed_paths", "boundary_paths"):
    if type(receipt[key]) is not list or any(type(value) is not str for value in receipt[key]):
        raise SystemExit(f"receipt field {key} must be an array of strings")
values = [
    receipt["schema"], receipt["export_dir"], receipt["manifest_sha256"],
    receipt["deployment_result"], receipt["production_base_url"],
    str(len(receipt["changed_paths"])), *receipt["changed_paths"],
    str(len(receipt["boundary_paths"])), *receipt["boundary_paths"],
]
if any("\0" in value for value in values):
    raise SystemExit("receipt strings must not contain NUL")
sys.stdout.buffer.write(b"\0".join(value.encode("utf-8") for value in values) + b"\0")
PY
}

require_http_200() {
    local url=$1 headers=$2 code
    code=$(shared_assets_public_http_request "$url" /dev/null "$headers")
    [[ $code == 200 ]] || {
        error "expected HTTP 200: $url"
        return 1
    }
}

cache_status() {
    local headers=$1
    awk 'tolower($1) == "cf-cache-status:" { value=$2 } END { sub(/\r$/, "", value); if (value != "") print value }' "$headers"
}

verify_cache_url() {
    local url=$1 headers
    headers=$(mktemp) || return 1
    require_http_200 "$url" "$headers" || { rm -f -- "$headers"; return 1; }
    [[ $(cache_status "$headers") == MISS ]] || { rm -f -- "$headers"; error "expected CF-Cache-Status MISS: $url"; return 1; }
    require_http_200 "$url" "$headers" || { rm -f -- "$headers"; return 1; }
    [[ $(cache_status "$headers") == HIT ]] || { rm -f -- "$headers"; error "expected CF-Cache-Status HIT: $url"; return 1; }
    rm -f -- "$headers"
    printf 'CACHE VERIFIED: MISS -> HIT %s\n' "$url"
}

main() {
    local receipt schema export_dir manifest_sha result base actual_sha manifest parsed changed_count boundary_count index=0
    local -a receipt_values changed_paths boundary_paths
    (( $# == 1 )) || { usage; return 2; }
    receipt=$1
    for command_name in awk curl go mktemp python3 readlink rmdir sha256sum ssh; do command -v "$command_name" >/dev/null || { error "required command missing: $command_name"; return 1; }; done
    load_production_identity_shared || { error 'formal shared production identity is invalid'; return 1; }
    SHARED_ASSETS_PUBLIC_BASE_URL=$PRODUCTION_SHARED_ASSETS_PUBLIC_ORIGIN
    SHARED_ASSETS_NETWORK_SSH_HOST=$PRODUCTION_ZGOCLOUD_SSH_ALIAS
    [[ -f $receipt && ! -L $receipt ]] || { error 'receipt must be a real regular file, not a symlink'; return 1; }
    receipt=$(readlink -f -- "$receipt")
    parsed=$(mktemp) || return 1
    if ! parse_receipt "$receipt" >"$parsed"; then rm -f -- "$parsed"; error 'invalid receipt schema'; return 1; fi
    mapfile -d '' -t receipt_values <"$parsed"
    rm -f -- "$parsed"
    [[ ${#receipt_values[@]} -ge 7 ]] || { error 'invalid receipt schema'; return 1; }
    schema=${receipt_values[index++]}; export_dir=${receipt_values[index++]}; manifest_sha=${receipt_values[index++]}; result=${receipt_values[index++]}; base=${receipt_values[index++]}; changed_count=${receipt_values[index++]}
    [[ $changed_count =~ ^[0-9]+$ && $changed_count -le 100 ]] || { error 'invalid receipt changed paths'; return 1; }
    changed_paths=("${receipt_values[@]:index:changed_count}"); index=$((index + changed_count))
    [[ ${receipt_values[index]+set} ]] || { error 'invalid receipt boundary paths'; return 1; }
    boundary_count=${receipt_values[index++]}
    [[ $boundary_count =~ ^[0-9]+$ && $boundary_count -le 10 ]] || { error 'invalid receipt boundary paths'; return 1; }
    boundary_paths=("${receipt_values[@]:index:boundary_count}"); index=$((index + boundary_count))
    [[ $index == ${#receipt_values[@]} ]] || { error 'invalid receipt schema'; return 1; }
    [[ $schema == "$RECEIPT_SCHEMA" && $base == "$SHARED_ASSETS_PUBLIC_BASE_URL" && $manifest_sha =~ ^[0-9a-f]{64}$ ]] || { error 'receipt contains unsupported schema, base URL, or manifest SHA'; return 1; }
    [[ $result == NO_CHANGES || $result == DEPLOYED ]] || { error 'receipt has unsupported deployment result'; return 1; }
    [[ $export_dir == /* && -d $export_dir && ! -L $export_dir && $(readlink -f -- "$export_dir") == "$export_dir" ]] || { error 'receipt export directory is not a canonical real directory'; return 1; }
    [[ $receipt == "$export_dir.verification-receipt.json" ]] || { error 'receipt path is not the formal sibling of its export directory'; return 1; }
    [[ ${boundary_paths[*]} == "${SHARED_ASSETS_EXPECTED_BOUNDARY_PATHS[*]}" ]] || { error 'receipt boundary paths do not match the fixed policy'; return 1; }
    if [[ $result == NO_CHANGES ]]; then
        [[ ${#changed_paths[@]} == 0 ]] || { error 'NO_CHANGES receipt contains changed paths'; return 1; }
    else
        [[ ${#changed_paths[@]} -gt 0 ]] || { error 'DEPLOYED receipt has no changed paths'; return 1; }
    fi
    for path in "${changed_paths[@]}"; do shared_assets_safe_logical_path "$path" || { error 'receipt contains unsafe changed path'; return 1; }; done
    script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
    repository_root=$(cd -P -- "$script_dir/.." && pwd -P)
    (cd -- "$repository_root" && go run -mod=readonly ./cmd/tour-i18n assets validate --input "$export_dir" >/dev/null) || { error 'export no longer passes formal assets validation; rerun the shared-assets flow'; return 1; }
    manifest="$export_dir/SHA256SUMS"
    actual_sha=$(sha256sum -- "$manifest"); actual_sha=${actual_sha%% *}
    [[ $actual_sha == "$manifest_sha" ]] || { error 'receipt manifest identity does not match current export; rerun the shared-assets flow'; return 1; }
    trap shared_assets_cleanup_network_ssh EXIT
    shared_assets_setup_network_ssh || return 1
    if [[ $result == DEPLOYED ]]; then
        for path in "${changed_paths[@]}"; do verify_cache_url "$SHARED_ASSETS_PUBLIC_BASE_URL/$path"; done
    else
        printf 'SKIP CACHE PURGE VERIFICATION: NO CHANGES\n'
    fi
    shared_assets_verify_public_assets "$manifest"
    shared_assets_verify_boundaries
    printf 'SHARED ASSETS PRODUCTION VERIFICATION: PASSED\n'
}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
    main "$@"
fi
