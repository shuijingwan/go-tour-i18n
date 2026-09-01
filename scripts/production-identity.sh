#!/usr/bin/env bash

# Shared, eval-free shell reader for production/identity.json. The Python
# validator owns schema and conflict checks; this file only maps ordered fields.

production_identity_script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
readonly PRODUCTION_IDENTITY_TOOL="$production_identity_script_dir/production-identity.py"
unset production_identity_script_dir

load_production_identity_locale() {
    local locale=$1 output
    local -a values=()
    output=$("$PRODUCTION_IDENTITY_TOOL" locale "$locale") || return 1
    mapfile -t values <<<"$output"
    if (( ${#values[@]} != 22 )); then
        printf 'production identity error: locale field count is %d, want 22\n' "${#values[@]}" >&2
        return 1
    fi
    PRODUCTION_LOCALE=${values[0]}
    PRODUCTION_STATE=${values[1]}
    PRODUCTION_HOSTNAME=${values[2]}
    PRODUCTION_CDN=${values[3]}
    PRODUCTION_ORIGIN_SSH_ALIAS=${values[4]}
    PRODUCTION_ORIGIN_IP=${values[5]}
    PRODUCTION_DATA_ROOT=${values[6]}
    PRODUCTION_RELEASES_ROOT=${values[7]}
    PRODUCTION_CURRENT=${values[8]}
    PRODUCTION_DEPLOYMENT_LOCK=${values[9]}
    PRODUCTION_SYSTEMD_SERVICE=${values[10]}
    PRODUCTION_SERVICE_USER=${values[11]}
    PRODUCTION_LOOPBACK_PORT=${values[12]}
    PRODUCTION_LOCALHOST_HEALTH_URL=${values[13]}
    PRODUCTION_ENVIRONMENT_FILE=${values[14]}
    PRODUCTION_NGINX_VHOST_PATH=${values[15]}
    PRODUCTION_TLS_CERTIFICATE_PATH=${values[16]}
    PRODUCTION_TLS_KEY_PATH=${values[17]}
    PRODUCTION_PLAYGROUND_ALLOWED_ORIGIN=${values[18]}
    PRODUCTION_SHARED_ASSETS_POLICY=${values[19]}
    PRODUCTION_PUBLIC_URL=${values[20]}
    PRODUCTION_CACHE_HEADER=${values[21]}
}

load_production_identity_shared() {
    local output
    local -a values=()
    output=$("$PRODUCTION_IDENTITY_TOOL" shared) || return 1
    mapfile -t values <<<"$output"
    if (( ${#values[@]} != 11 )); then
        printf 'production identity error: shared field count is %d, want 11\n' "${#values[@]}" >&2
        return 1
    fi
    PRODUCTION_ALIYUN_SSH_ALIAS=${values[0]}
    PRODUCTION_ZGOCLOUD_SSH_ALIAS=${values[1]}
    PRODUCTION_SHARED_ORIGIN_IP=${values[2]}
    PRODUCTION_PLAYGROUND_VHOST_PATH=${values[3]}
    PRODUCTION_PLAYGROUND_PUBLIC_ORIGIN=${values[4]}
    PRODUCTION_CLOUDFLARE_ZONE_NAME=${values[5]}
    PRODUCTION_CLOUDFLARE_SECRET_FILE=${values[6]}
    PRODUCTION_NGINX_TEST_COMMAND=${values[7]}
    PRODUCTION_NGINX_RELOAD_COMMAND=${values[8]}
    PRODUCTION_SHARED_ASSETS_ORIGIN_ROOT=${values[9]}
    PRODUCTION_SHARED_ASSETS_PUBLIC_ORIGIN=${values[10]}
}
