#!/usr/bin/env bash

# Shared, fail-closed public verification core.  Callers must validate the
# current export before calling these functions and must install EXIT cleanup.

readonly -a SHARED_ASSETS_EXPECTED_BOUNDARY_PATHS=(
    'tour/script.js'
    'tour/static/img/tree.png'
    'tour/static/partials/editor.html'
)

SHARED_ASSETS_PUBLIC_BASE_URL=''
SHARED_ASSETS_CURL_NETWORK_OPTIONS=()
SHARED_ASSETS_NETWORK_SSH_HOST=''
SHARED_ASSETS_NETWORK_CONTROL_DIR=''
SHARED_ASSETS_NETWORK_CONTROL_PATH=''
SHARED_ASSETS_NETWORK_PROXY_PORT=''

shared_assets_public_error() { printf '[verify-shared-assets-production] ERROR: %s\n' "$*" >&2; }

shared_assets_safe_logical_path() {
    local path=$1
    [[ $path =~ ^[A-Za-z0-9._/-]+$ && $path != /* && $path != *'..'* && $path != *'//'* && $path != *'\\'* ]]
}

shared_assets_setup_network_ssh() {
    [[ $SHARED_ASSETS_NETWORK_SSH_HOST =~ ^[A-Za-z0-9._-]+$ ]] || {
        shared_assets_public_error "network runner has unsafe SSH alias: $SHARED_ASSETS_NETWORK_SSH_HOST"
        return 1
    }
    SHARED_ASSETS_NETWORK_CONTROL_DIR=$(mktemp -d "${TMPDIR:-/tmp}/go-tour-verify-shared-assets-network.XXXXXX") || return 1
    SHARED_ASSETS_NETWORK_CONTROL_PATH=$SHARED_ASSETS_NETWORK_CONTROL_DIR/control
    SHARED_ASSETS_NETWORK_PROXY_PORT=$(python3 - <<'PY'
import socket
with socket.socket() as listener:
    listener.bind(("127.0.0.1", 0))
    print(listener.getsockname()[1])
PY
    ) || return 1
    ssh -o BatchMode=yes -o ConnectTimeout=10 -o ServerAliveInterval=5 \
        -o ServerAliveCountMax=3 -o ConnectionAttempts=3 -o ControlMaster=yes \
        -o ControlPersist=60 -o "ControlPath=$SHARED_ASSETS_NETWORK_CONTROL_PATH" \
        -N -f -D "127.0.0.1:$SHARED_ASSETS_NETWORK_PROXY_PORT" "$SHARED_ASSETS_NETWORK_SSH_HOST" || {
        shared_assets_public_error "network runner SSH ControlMaster + SOCKS failed: $SHARED_ASSETS_NETWORK_SSH_HOST"
        return 1
    }
    SHARED_ASSETS_CURL_NETWORK_OPTIONS=(--socks5-hostname "127.0.0.1:$SHARED_ASSETS_NETWORK_PROXY_PORT")
    printf '[verify-shared-assets-production] public network runner: %s\n' "$SHARED_ASSETS_NETWORK_SSH_HOST"
}

shared_assets_cleanup_network_ssh() {
    if [[ -n $SHARED_ASSETS_NETWORK_CONTROL_DIR && -n $SHARED_ASSETS_NETWORK_CONTROL_PATH && $SHARED_ASSETS_NETWORK_CONTROL_PATH == "$SHARED_ASSETS_NETWORK_CONTROL_DIR/control" && ${SHARED_ASSETS_NETWORK_CONTROL_DIR##*/} == go-tour-verify-shared-assets-network.* ]]; then
        if [[ -S $SHARED_ASSETS_NETWORK_CONTROL_PATH || -e $SHARED_ASSETS_NETWORK_CONTROL_PATH ]]; then
            ssh -o BatchMode=yes -o ConnectTimeout=10 -o "ControlPath=$SHARED_ASSETS_NETWORK_CONTROL_PATH" \
                -O exit "$SHARED_ASSETS_NETWORK_SSH_HOST" >/dev/null 2>&1 || true
        fi
        rm -f -- "$SHARED_ASSETS_NETWORK_CONTROL_PATH" "$SHARED_ASSETS_NETWORK_CONTROL_PATH.pid"
        rmdir -- "$SHARED_ASSETS_NETWORK_CONTROL_DIR" 2>/dev/null || true
    fi
}

shared_assets_public_curl() { curl "${SHARED_ASSETS_CURL_NETWORK_OPTIONS[@]}" "$@"; }

shared_assets_public_http_request() {
    local url=$1 body=$2 headers=$3 attempt=1 code
    while :; do
        # Do not use curl -f: 522/525 must remain observable HTTP statuses.
        code=$(shared_assets_public_curl -sS -D "$headers" -o "$body" -w '%{http_code}' "$url" || true)
        case $code in
            522|525)
                if (( attempt < 3 )); then
                    printf '[verify-shared-assets-production] transient HTTP %s, retry %d/2: %s\n' "$code" "$attempt" "$url" >&2
                    sleep "$attempt"
                    attempt=$((attempt + 1))
                    continue
                fi
                ;;
        esac
        printf '%s\n' "${code:-000}"
        return 0
    done
}

shared_assets_verify_public_assets() {
    local manifest=$1 checksum path body headers code count=0 got
    while IFS=' ' read -r checksum path; do
        [[ $checksum =~ ^[0-9a-f]{64}$ ]] && shared_assets_safe_logical_path "$path" || { shared_assets_public_error 'invalid formal SHA256SUMS entry'; return 1; }
        body=$(mktemp) || return 1
        headers=$(mktemp) || { rm -f -- "$body"; return 1; }
        code=$(shared_assets_public_http_request "$SHARED_ASSETS_PUBLIC_BASE_URL/$path" "$body" "$headers")
        rm -f -- "$headers"
        if [[ $code != 200 ]]; then rm -f -- "$body"; shared_assets_public_error "public asset request failed: $path"; return 1; fi
        got=$(sha256sum -- "$body"); got=${got%% *}; rm -f -- "$body"
        [[ $got == "$checksum" ]] || { shared_assets_public_error "public asset SHA-256 mismatch: $path"; return 1; }
        count=$((count + 1)); printf 'PUBLIC SHA-256 VERIFIED: %s\n' "$path"
    done <"$manifest"
    [[ $count == 11 ]] || { shared_assets_public_error "expected 11 public allowlist files, got $count"; return 1; }
}

shared_assets_verify_boundaries() {
    local path headers status
    for path in "${SHARED_ASSETS_EXPECTED_BOUNDARY_PATHS[@]}"; do
        headers=$(mktemp) || return 1
        status=$(shared_assets_public_http_request "$SHARED_ASSETS_PUBLIC_BASE_URL/$path" /dev/null "$headers")
        rm -f -- "$headers"
        [[ $status == 404 ]] || { shared_assets_public_error "expected boundary HTTP 404: /$path"; return 1; }
        printf 'BOUNDARY VERIFIED: 404 /%s\n' "$path"
    done
}
