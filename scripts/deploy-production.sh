#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

readonly HEALTH_ATTEMPTS=12
readonly HEALTH_INTERVAL=3
readonly NO_OLD_RELEASE='NO_OLD_RELEASE'
readonly ALREADY_CURRENT='ALREADY_CURRENT'
readonly PUBLIC_CURL_CONNECT_TIMEOUT=5
readonly PUBLIC_CURL_MAX_TIME=15
readonly PUBLIC_CURL_RETRY_ATTEMPTS=3
readonly -a SSH_BASE_OPTIONS=(
    -o BatchMode=yes
    -o ConnectTimeout=10
    -o ServerAliveInterval=5
    -o ServerAliveCountMax=3
    -o ConnectionAttempts=3
    -o ControlMaster=auto
    -o ControlPersist=60
)
SSH_OPTIONS=("${SSH_BASE_OPTIONS[@]}")
RSYNC_SSH_COMMAND='ssh'
SSH_CONTROL_DIR=''
SSH_CONTROL_PATH=''

script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
# shellcheck source=production-identity.sh
source "$script_dir/production-identity.sh"
unset script_dir

RELEASE_LOCALE=''
SSH_HOST=''
SERVICE_USER=''
RELEASES_DIR=''
CURRENT_LINK=''
DEPLOY_LOCK=''
SERVICE=''
HEALTH_URL=''
PUBLIC_URL=''
PUBLIC_ACCEPTANCE_HINT=''
EXPECTED_DEPLOYMENT_MODE=''

setup_ssh_multiplex() {
    local option quoted
    SSH_CONTROL_DIR=$(mktemp -d "${TMPDIR:-/tmp}/go-tour-production-ssh.XXXXXX") || {
        error '无法创建本次 deployment 专用的 SSH control 目录'
        return 1
    }
    SSH_CONTROL_PATH="$SSH_CONTROL_DIR/control"
    SSH_OPTIONS=("${SSH_BASE_OPTIONS[@]}" -o "ControlPath=$SSH_CONTROL_PATH")
    RSYNC_SSH_COMMAND='ssh'
    for option in "${SSH_OPTIONS[@]}"; do
        printf -v quoted '%q' "$option"
        RSYNC_SSH_COMMAND+=" $quoted"
    done
}

cleanup_ssh_multiplex() {
    if [[ -n $SSH_CONTROL_DIR && -n $SSH_CONTROL_PATH && $SSH_CONTROL_PATH == "$SSH_CONTROL_DIR/control" && ${SSH_CONTROL_DIR##*/} == go-tour-production-ssh.* ]]; then
        if [[ -S $SSH_CONTROL_PATH ]]; then
            ssh "${SSH_OPTIONS[@]}" -O exit "$SSH_HOST" >/dev/null 2>&1 || true
        fi
        rm -f -- "$SSH_CONTROL_PATH"
        rmdir -- "$SSH_CONTROL_DIR" 2>/dev/null || true
    fi
    SSH_CONTROL_DIR=''
    SSH_CONTROL_PATH=''
}

log() {
    printf '[deploy] %s\n' "$*"
}

error() {
    printf '[deploy] ERROR: %s\n' "$*" >&2
}

usage() {
    printf 'usage: %s /tmp/go-tour-release-YYYYMMDD-<locale>-<shortsha>\n' "${0##*/}" >&2
}

select_deployment_profile() {
    local locale=$1

    if ! load_production_identity_locale "$locale"; then
        error "unsupported or invalid production locale in release.json: $locale"
        return 1
    fi
    SSH_HOST=$PRODUCTION_ORIGIN_SSH_ALIAS
    SERVICE_USER=$PRODUCTION_SERVICE_USER
    RELEASES_DIR=$PRODUCTION_RELEASES_ROOT
    CURRENT_LINK=$PRODUCTION_CURRENT
    DEPLOY_LOCK=$PRODUCTION_DEPLOYMENT_LOCK
    SERVICE=$PRODUCTION_SYSTEMD_SERVICE
    HEALTH_URL=$PRODUCTION_LOCALHOST_HEALTH_URL
    PUBLIC_URL=$PRODUCTION_PUBLIC_URL
    PUBLIC_ACCEPTANCE_HINT='inspect the CDN/reverse-proxy cache and refresh it manually if needed'
    case $PRODUCTION_STATE in
        live) EXPECTED_DEPLOYMENT_MODE=EXISTING ;;
        first-production) EXPECTED_DEPLOYMENT_MODE=FIRST_DEPLOYMENT ;;
        *) error "unsupported production state: $PRODUCTION_STATE"; return 1 ;;
    esac
}

manual_check_hint() {
    error '远端状态无法安全确定。不要直接重复部署；请先执行：'
    printf '%s\n' \
        "  ssh $SSH_HOST 'readlink -f $CURRENT_LINK'" \
        "  ssh $SSH_HOST 'systemctl status $SERVICE --no-pager -l'" \
        "  ssh $SSH_HOST 'curl -sS -o /dev/null -w \"%{http_code}\\n\" $HEALTH_URL'" \
        "  ssh $SSH_HOST 'journalctl -u $SERVICE -n 80 --no-pager'" >&2
    error "部署锁和现场已保留：$DEPLOY_LOCK"
}

validate_local_tools() {
    local command_name

    for command_name in awk basename curl date find mktemp python3 rsync sha256sum sort ssh xargs; do
        command -v "$command_name" >/dev/null || {
            error "required local command is missing: $command_name"
            return 1
        }
    done
}

release_tree_sha256() {
    local root=$1

    (
        cd -- "$root"
        find . -type f -print0 |
            LC_ALL=C sort -z |
            xargs -0 sha256sum |
            sha256sum |
            awk '{ print $1 }'
    )
}

release_name_from_path() {
    local local_name remote_name

    local_name=$(basename -- "$1")
    if [[ $local_name != go-tour-release-* ]]; then
        error "release directory basename must start with go-tour-release-: $local_name"
        return 1
    fi

    remote_name=${local_name#go-tour-release-}
    if [[ -z $remote_name || $remote_name == '.' || $remote_name == '..' || ! $remote_name =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
        error "unsafe remote release name derived from $local_name"
        return 1
    fi
    printf '%s\n' "$remote_name"
}

validate_local_release() {
    local input_path=$1
    local release_dir entry entry_name symlink unsupported checksum_output checksum_count
    local -a root_entries=()

    if [[ ! -d $input_path || -L $input_path ]]; then
        error "release path must be a real directory, not a symlink: $input_path"
        return 1
    fi
    release_dir=$(cd -P -- "$input_path" && pwd -P)

    mapfile -d '' root_entries < <(find "$release_dir" -mindepth 1 -maxdepth 1 -print0)
    if (( ${#root_entries[@]} != 4 )); then
        error 'release root must contain exactly bin, _content, release.json, and SHA256SUMS'
        return 1
    fi
    for entry in "${root_entries[@]}"; do
        entry_name=${entry##*/}
        case $entry_name in
            bin|_content|release.json|SHA256SUMS) ;;
            *)
                error "unexpected release root entry: $entry_name"
                return 1
                ;;
        esac
    done

    if [[ ! -d $release_dir/bin || ! -d $release_dir/_content || ! -f $release_dir/release.json || ! -f $release_dir/SHA256SUMS ]]; then
        error 'release root entries have unexpected types'
        return 1
    fi
    if [[ ! -f $release_dir/bin/tour || ! -x $release_dir/bin/tour ]]; then
        error 'bin/tour must be a regular executable file'
        return 1
    fi
    symlink=$(find "$release_dir" -type l -print -quit)
    unsupported=$(find "$release_dir" ! -type d ! -type f -print -quit)
    if [[ -n $symlink || -n $unsupported ]]; then
        error "release contains a symlink or unsupported file: ${symlink:-$unsupported}"
        return 1
    fi

    RELEASE_LOCALE=$(python3 - "$release_dir" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
try:
    manifest = json.loads((root / "release.json").read_text(encoding="utf-8"))
except (OSError, json.JSONDecodeError) as exc:
    raise SystemExit(f"bundle metadata error: {exc}")
locale = manifest.get("locale")
if type(locale) is not str or not locale:
    raise SystemExit(f"release.json constraint failed: locale={locale!r}, want non-empty string")
print(locale)
PY
    ) || {
        error 'release locale validation failed'
        return 1
    }

    # This whitelist is resolved before any SSH, upload, lock, or production change.
    select_deployment_profile "$RELEASE_LOCALE" || return 1

    if ! python3 - "$release_dir" "$RELEASE_LOCALE" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
profile_locale = sys.argv[2]
try:
    manifest = json.loads((root / "release.json").read_text(encoding="utf-8"))
    metadata = json.loads(
        (root / "_content" / "tour" / "site-metadata.json").read_text(encoding="utf-8")
    )
except (OSError, json.JSONDecodeError) as exc:
    raise SystemExit(f"bundle metadata error: {exc}")

expected = {
    "schema_version": 2,
    "locale": profile_locale,
    "pages": 103,
    "articles": 7,
    "execution_transport": "http-playground-proxy",
    "execution_provider": "play.golang.org",
    "local_socket_enabled": False,
    "goos": "linux",
    "goarch": "amd64",
}
for key, want in expected.items():
    if key not in manifest or manifest[key] != want or type(manifest[key]) is not type(want):
        raise SystemExit(
            f"release.json constraint failed: {key}={manifest.get(key)!r}, want {want!r}"
        )

for key in ("translation_units", "eligible_examples"):
    value = manifest.get(key)
    if type(value) is not int or value < 0:
        raise SystemExit(f"release.json constraint failed: {key}={value!r}, want non-negative int")
if manifest["translation_units"] != manifest["pages"] + manifest["eligible_examples"]:
    raise SystemExit(
        "release.json constraint failed: translation_units must equal pages + eligible_examples"
    )

for key in (
    "locale",
    "published_at",
    "upstream_commit",
    "upstream_commit_time",
    "pages",
    "articles",
):
    if key not in manifest or key not in metadata or manifest[key] != metadata[key]:
        raise SystemExit(f"site-metadata.json does not match release.json field {key}")
PY
    then
        error 'release metadata validation failed'
        return 1
    fi

    if ! checksum_output=$(cd -- "$release_dir" && sha256sum -c --strict SHA256SUMS); then
        printf '%s\n' "$checksum_output" >&2
        error 'local SHA256 verification failed'
        return 1
    fi
    checksum_count=$(awk 'END { print NR }' "$release_dir/SHA256SUMS")
    log "SHA256 verification: PASS ($checksum_count files)"
    log "local release preflight passed: $release_dir"
}

prepare_remote() {
    local remote_staging=$1
    local remote_final=$2

    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$RELEASES_DIR" "$CURRENT_LINK" "$DEPLOY_LOCK" "$SERVICE" \
        "$remote_staging" "$remote_final" "$EXPECTED_DEPLOYMENT_MODE" <<'REMOTE_PREPARE'
set -Eeuo pipefail
IFS=$'\n\t'
readonly NO_OLD_RELEASE='NO_OLD_RELEASE'

releases_dir=$1
current_link=$2
deploy_lock=$3
service=$4
staging=$5
final=$6
expected_mode=$7

fail() {
    printf '[deploy:remote] ERROR: %s\n' "$*" >&2
    exit 1
}

[[ $(id -u) == 0 ]] || fail 'remote SSH user must be root'
[[ -d $releases_dir ]] || fail "release root does not exist: $releases_dir"
for command_name in rsync sha256sum find chmod chown systemctl readlink ln mv; do
    command -v "$command_name" >/dev/null || fail "required remote command is missing: $command_name"
done
if [[ -L $current_link ]]; then
    deployment_mode=EXISTING
    old=$(readlink -f -- "$current_link") || fail 'cannot resolve current release'
    [[ -d $old ]] || fail "current does not resolve to a directory: $old"
    case $old in
        "$releases_dir"/*) ;;
        *) fail "current points outside release root: $old" ;;
    esac
elif [[ ! -e $current_link ]]; then
    deployment_mode=FIRST_DEPLOYMENT
    old="$NO_OLD_RELEASE"
else
    fail "current exists but is not a symlink: $current_link"
fi
[[ $deployment_mode == "$expected_mode" ]] || fail "remote state is $deployment_mode but formal production identity requires $expected_mode"
systemctl cat "$service" >/dev/null || fail "systemd service does not exist: $service"
if [[ $deployment_mode == EXISTING && $final == "$old" ]]; then
    [[ -d $releases_dir && ! -L $releases_dir ]] || fail "release root is not a real directory: $releases_dir"
    [[ -d $final && ! -L $final ]] || fail "current release is not a real directory: $final"
    [[ ! -e $deploy_lock && ! -L $deploy_lock ]] || fail "deployment lock exists; already-current state requires manual inspection: $deploy_lock"
    printf 'ALREADY_CURRENT\t%s\n' "$old"
    exit 0
fi
[[ ! -e $final && ! -L $final ]] || fail "remote release already exists: $final"
[[ ! -e $staging && ! -L $staging ]] || fail "remote staging already exists: $staging"

if ! mkdir -- "$deploy_lock"; then
    fail "deployment lock exists; another or an unfinished deployment may need manual inspection: $deploy_lock"
fi
if ! mkdir -m 0700 -- "$staging"; then
    rmdir -- "$deploy_lock" || true
    fail "cannot create staging: $staging"
fi

printf '%s\t%s\n' "$deployment_mode" "$old"
REMOTE_PREPARE
}

verify_already_current_release() {
    local remote_final=$1
    local expected_tree_sha256=$2

    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$RELEASES_DIR" "$CURRENT_LINK" "$DEPLOY_LOCK" "$SERVICE" \
        "$HEALTH_URL" "$HEALTH_ATTEMPTS" "$HEALTH_INTERVAL" "$remote_final" \
        "$expected_tree_sha256" <<'REMOTE_RESUME'
set -Eeuo pipefail
IFS=$'\n\t'

releases_dir=$1
current_link=$2
deploy_lock=$3
service=$4
health_url=$5
health_attempts=$6
health_interval=$7
expected_remote=$8
expected_tree_sha256=$9

fail() {
    printf '[deploy:remote] ERROR: already-current verification: %s\n' "$*" >&2
    exit 1
}

tree_sha256() {
    local root=$1
    (
        cd -- "$root"
        find . -type f -print0 |
            LC_ALL=C sort -z |
            xargs -0 sha256sum |
            sha256sum |
            awk '{ print $1 }'
    )
}

health_check() {
    local attempt consecutive=0 service_state http_code

    for ((attempt = 1; attempt <= health_attempts; attempt++)); do
        service_state=$(systemctl is-active "$service" 2>/dev/null || true)
        http_code=$(curl --silent --output /dev/null --write-out '%{http_code}' \
            --connect-timeout 2 --max-time 5 "$health_url" || true)
        if [[ $service_state == active && $http_code == 200 ]]; then
            ((consecutive += 1))
            printf '[deploy:remote] resume health %d/%d: active + HTTP 200 (consecutive %d/3)\n' \
                "$attempt" "$health_attempts" "$consecutive"
            if (( consecutive == 3 )); then
                return 0
            fi
        else
            consecutive=0
            printf '[deploy:remote] resume health %d/%d: service=%s HTTP=%s\n' \
                "$attempt" "$health_attempts" "${service_state:-unknown}" "${http_code:-000}" >&2
        fi
        (( attempt == health_attempts )) || sleep "$health_interval"
    done
    return 1
}

[[ $(id -u) == 0 ]] || fail 'remote SSH user must be root'
for command_name in awk curl find sha256sum sort systemctl readlink xargs; do
    command -v "$command_name" >/dev/null || fail "required remote command is missing: $command_name"
done
[[ $expected_tree_sha256 =~ ^[0-9a-f]{64}$ ]] || fail 'local release tree identity is malformed'
[[ -d $releases_dir && ! -L $releases_dir ]] || fail "release root is not a real directory: $releases_dir"
case $expected_remote in
    "$releases_dir"/*) ;;
    *) fail "expected release is outside release root: $expected_remote" ;;
esac
[[ -d $expected_remote && ! -L $expected_remote ]] || fail "expected release is not a real directory: $expected_remote"
[[ -L $current_link ]] || fail "current is not a symlink: $current_link"
actual_current=$(readlink -f -- "$current_link") || fail 'cannot resolve current release'
[[ $actual_current == "$expected_remote" ]] || fail "current changed: ${actual_current:-unresolved}"
[[ ! -e $deploy_lock && ! -L $deploy_lock ]] || fail "deployment lock is present: $deploy_lock"

mapfile -d '' root_entries < <(find "$expected_remote" -mindepth 1 -maxdepth 1 -print0)
(( ${#root_entries[@]} == 4 )) || fail 'release root must contain exactly bin, _content, release.json, and SHA256SUMS'
for entry in "${root_entries[@]}"; do
    case ${entry##*/} in
        bin|_content|release.json|SHA256SUMS) ;;
        *) fail "unexpected release root entry: ${entry##*/}" ;;
    esac
done
[[ -d $expected_remote/bin && ! -L $expected_remote/bin ]] || fail 'bin is not a real directory'
[[ -d $expected_remote/_content && ! -L $expected_remote/_content ]] || fail '_content is not a real directory'
[[ -f $expected_remote/release.json && ! -L $expected_remote/release.json ]] || fail 'release.json is not a real file'
[[ -f $expected_remote/SHA256SUMS && ! -L $expected_remote/SHA256SUMS ]] || fail 'SHA256SUMS is not a real file'
[[ -f $expected_remote/bin/tour && -x $expected_remote/bin/tour && ! -L $expected_remote/bin/tour ]] || fail 'bin/tour is not a regular executable file'
symlink=$(find "$expected_remote" -type l -print -quit)
unsupported=$(find "$expected_remote" ! -type d ! -type f -print -quit)
[[ -z $symlink && -z $unsupported ]] || fail "release contains a symlink or unsupported file: ${symlink:-$unsupported}"

if ! checksum_output=$(cd -- "$expected_remote" && sha256sum -c --strict SHA256SUMS); then
    printf '%s\n' "$checksum_output" >&2
    fail 'remote SHA256SUMS verification failed'
fi
actual_tree_sha256=$(tree_sha256 "$expected_remote") || fail 'cannot compute remote release tree identity'
[[ $actual_tree_sha256 == "$expected_tree_sha256" ]] || fail "release tree identity mismatch: got $actual_tree_sha256 want $expected_tree_sha256"
service_state=$(systemctl is-active "$service" 2>/dev/null || true)
[[ $service_state == active ]] || fail "service is ${service_state:-unknown}, want active"
health_check || fail 'service did not reach three consecutive active + HTTP 200 checks'

actual_current=$(readlink -f -- "$current_link") || fail 'cannot resolve current release after health verification'
[[ $actual_current == "$expected_remote" ]] || fail "current changed during verification: ${actual_current:-unresolved}"
[[ ! -e $deploy_lock && ! -L $deploy_lock ]] || fail "deployment lock appeared during verification: $deploy_lock"
printf '[deploy:remote] deployment already current / RESUME: %s; release identity and source health PASS\n' "$expected_remote"
REMOTE_RESUME
}

upload_release() {
    local release_dir=$1
    local remote_staging=$2

    rsync -rlt --no-owner --no-group --no-perms --protect-args \
        -e "$RSYNC_SSH_COMMAND" -- \
        "$release_dir/" "$SSH_HOST:$remote_staging/"
}

validate_remote_release() {
    local remote_staging=$1

    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$remote_staging" "$SERVICE_USER" <<'REMOTE_VALIDATE'
set -Eeuo pipefail
IFS=$'\n\t'

staging=$1
service_user=$2

[[ -d $staging ]] || {
    printf '[deploy:remote] ERROR: staging is missing: %s\n' "$staging" >&2
    exit 1
}

chown -R root:root -- "$staging"
find "$staging" -type d -exec chmod 0755 {} +
find "$staging" -type f -exec chmod 0644 {} +
chmod 0755 -- "$staging/bin/tour"

[[ -f $staging/bin/tour && -x $staging/bin/tour ]] || exit 1
[[ -f $staging/release.json && -f $staging/SHA256SUMS && -d $staging/_content ]] || exit 1
[[ -z $(find "$staging" -type l -print -quit) ]] || exit 1

if ! checksum_output=$(cd -- "$staging" && sha256sum -c --strict SHA256SUMS); then
    printf '%s\n' "$checksum_output" >&2
    exit 1
fi
checksum_count=$(awk 'END { print NR }' "$staging/SHA256SUMS")
su -s /bin/sh -c 'test -x "$1" && test -r "$2" && test -r "$3"' \
    "$service_user" sh \
    "$staging/bin/tour" \
    "$staging/release.json" \
    "$staging/_content/tour/static/css/app.css"

printf '[deploy:remote] permissions verification: PASS; SHA256 verification: PASS (%s files)\n' "$checksum_count"
REMOTE_VALIDATE
}

cleanup_before_activation() {
    local remote_staging=$1

    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$remote_staging" "$DEPLOY_LOCK" <<'REMOTE_CLEANUP'
set -Eeuo pipefail
staging=$1
deploy_lock=$2

rm -rf -- "$staging"
rmdir -- "$deploy_lock"
REMOTE_CLEANUP
}

activate_release() {
    local deployment_mode=$1
    local old_release=$2
    local remote_staging=$3
    local remote_final=$4
    local link_suffix=$5

    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$RELEASES_DIR" "$CURRENT_LINK" "$DEPLOY_LOCK" "$SERVICE" "$HEALTH_URL" \
        "$HEALTH_ATTEMPTS" "$HEALTH_INTERVAL" "$deployment_mode" "$old_release" "$remote_staging" \
        "$remote_final" "$link_suffix" <<'REMOTE_ACTIVATE'
set -Eeuo pipefail
IFS=$'\n\t'

releases_dir=$1
current_link=$2
deploy_lock=$3
service=$4
health_url=$5
health_attempts=$6
health_interval=$7
deployment_mode=$8
expected_old=$9
staging=${10}
final=${11}
link_suffix=${12}
readonly NO_OLD_RELEASE='NO_OLD_RELEASE'
next_link="${current_link}.next-${link_suffix}"
rollback_link="${current_link}.rollback-${link_suffix}"

failed_before_current_switch() {
    rm -f -- "$next_link" "$rollback_link"
    rm -rf -- "$staging" "$final"
    rmdir -- "$deploy_lock" || true
    printf '[deploy:remote] RESULT=FAILED_BEFORE_CURRENT_SWITCH\n'
    exit 1
}

health_check() {
    local attempt consecutive=0 service_state http_code

    for ((attempt = 1; attempt <= health_attempts; attempt++)); do
        service_state=$(systemctl is-active "$service" 2>/dev/null || true)
        http_code=$(curl --silent --output /dev/null --write-out '%{http_code}' \
            --connect-timeout 2 --max-time 5 "$health_url" || true)
        if [[ $service_state == active && $http_code == 200 ]]; then
            ((consecutive += 1))
            printf '[deploy:remote] health %d/%d: active + HTTP 200 (consecutive %d/3)\n' \
                "$attempt" "$health_attempts" "$consecutive"
            if (( consecutive == 3 )); then
                return 0
            fi
        else
            consecutive=0
            printf '[deploy:remote] health %d/%d: service=%s HTTP=%s\n' \
                "$attempt" "$health_attempts" "${service_state:-unknown}" "${http_code:-000}" >&2
        fi
        (( attempt == health_attempts )) || sleep "$health_interval"
    done
    return 1
}

rollback() {
    local restart_ok=1

    printf '[deploy:remote] new release failed; rolling back to %s\n' "$expected_old" >&2
    rm -f -- "$rollback_link"
    ln -s -- "$expected_old" "$rollback_link" || return 1
    mv -Tf -- "$rollback_link" "$current_link" || return 1
    if ! systemctl restart "$service"; then
        restart_ok=0
    fi
    health_check && (( restart_ok ))
}

[[ -d $deploy_lock ]] || exit 1
case $deployment_mode in
    EXISTING)
        [[ -L $current_link ]] || exit 1
        actual_old=$(readlink -f -- "$current_link") || exit 1
        [[ $actual_old == "$expected_old" ]] || {
            printf '[deploy:remote] ERROR: current changed since preflight: %s\n' "$actual_old" >&2
            exit 1
        }
        case $actual_old in
            "$releases_dir"/*) ;;
            *) exit 1 ;;
        esac
        ;;
    FIRST_DEPLOYMENT)
        [[ $expected_old == NO_OLD_RELEASE && ! -e $current_link && ! -L $current_link ]] || exit 1
        ;;
    *) exit 1 ;;
esac
[[ -d $staging ]] || exit 1
[[ ! -e $final && ! -L $final ]] || exit 1
[[ ! -e $next_link && ! -L $next_link && ! -e $rollback_link && ! -L $rollback_link ]] || exit 1

if ! mv -T -- "$staging" "$final"; then
    failed_before_current_switch
fi
if ! ln -s -- "$final" "$next_link"; then
    failed_before_current_switch
fi
mv -Tf -- "$next_link" "$current_link"
printf '[deploy:remote] current switched to %s\n' "$final"

if systemctl restart "$service" && health_check; then
    rmdir -- "$deploy_lock"
    printf '[deploy:remote] deployment completed: %s\n' "$final"
    exit 0
fi

if [[ $deployment_mode == EXISTING ]]; then
    if rollback; then
        rmdir -- "$deploy_lock"
        printf '[deploy:remote] rollback completed; old release is healthy\n' >&2
        exit 20
    fi

    printf '[deploy:remote] ERROR: rollback failed; manual recovery is required\n' >&2
    printf '[deploy:remote] current=%s\n' "$(readlink -f -- "$current_link" 2>/dev/null || printf unresolved)" >&2
    systemctl status "$service" --no-pager -l >&2 || true
    journalctl -u "$service" -n 80 --no-pager >&2 || true
    exit 21
fi

printf '[deploy:remote] ERROR: FIRST_DEPLOYMENT health failure; no rollback target exists\n' >&2
printf '[deploy:remote] current=%s\n' "$(readlink -f -- "$current_link" 2>/dev/null || printf unresolved)" >&2
systemctl status "$service" --no-pager -l >&2 || true
journalctl -u "$service" -n 80 --no-pager >&2 || true
printf '[deploy:remote] RESULT=FIRST_DEPLOYMENT_HEALTH_FAILURE\n'
exit 22
REMOTE_ACTIVATE
}

check_public() {
    local attempt curl_exit=0 http_code='' transient=0

    for ((attempt = 1; attempt <= PUBLIC_CURL_RETRY_ATTEMPTS; attempt++)); do
        set +e
        http_code=$(curl --silent --output /dev/null --write-out '%{http_code}' \
            --connect-timeout "$PUBLIC_CURL_CONNECT_TIMEOUT" --max-time "$PUBLIC_CURL_MAX_TIME" "$PUBLIC_URL")
        curl_exit=$?
        set -e
        if (( curl_exit == 0 )) && [[ $http_code == 200 ]]; then
            log "public acceptance passed: HTTP 200 $PUBLIC_URL"
            log "$PUBLIC_ACCEPTANCE_HINT"
            return 0
        fi

        transient=0
        if (( curl_exit != 0 )); then
            case $curl_exit in
                6|7|16|28|35) transient=1 ;;
            esac
        elif [[ $http_code == 522 || $http_code == 525 ]]; then
            transient=1
        fi
        if (( transient && attempt < PUBLIC_CURL_RETRY_ATTEMPTS )); then
            log "public acceptance transient failure $attempt/$PUBLIC_CURL_RETRY_ATTEMPTS: curl exit $curl_exit; HTTP ${http_code:-000}; retrying in ${attempt}s"
            sleep "$attempt"
            continue
        fi
        break
    done

    if (( curl_exit != 0 )); then
        error "localhost is healthy, but public acceptance failed after $attempt attempt(s): curl exit $curl_exit; HTTP ${http_code:-000}: $PUBLIC_URL"
    else
        error "localhost is healthy, but public acceptance failed after $attempt attempt(s): curl exit 0; HTTP ${http_code:-000}: $PUBLIC_URL"
    fi
    error 'this CDN/reverse-proxy result did not roll back the healthy source release'
    error "$PUBLIC_ACCEPTANCE_HINT"
    return 1
}

main() {
    local release_input release_dir remote_name remote_final remote_staging old_release deployment_mode prepare_output
    local link_suffix activation_rc activation_output local_tree_sha256
    local staging_ready=0 activation_started=0

    interrupted() {
        trap - INT TERM HUP
        if (( staging_ready && ! activation_started )); then
            error 'deployment interrupted before activation; cleaning this staging and lock'
            cleanup_before_activation "$remote_staging" || error "cleanup failed; inspect $DEPLOY_LOCK manually"
        elif (( activation_started )); then
            manual_check_hint
        fi
        exit 1
    }
    trap interrupted INT TERM HUP

    if (( $# != 1 )); then
        usage
        return 1
    fi
    release_input=$1
    validate_local_tools || return 1
    setup_ssh_multiplex || return 1
    trap cleanup_ssh_multiplex EXIT
    remote_name=$(release_name_from_path "$release_input") || return 1
    validate_local_release "$release_input" || return 1
    release_dir=$(cd -P -- "$release_input" && pwd -P)
    local_tree_sha256=$(release_tree_sha256 "$release_dir") || {
        error 'cannot compute local release tree identity'
        return 1
    }

    log "selected production profile from release.json: $RELEASE_LOCALE"

    link_suffix="$(date -u +%Y%m%dT%H%M%SZ)-$$-${RANDOM}"
    remote_final="$RELEASES_DIR/$remote_name"
    remote_staging="$RELEASES_DIR/.${remote_name}.staging-${link_suffix}"

    log "remote preflight and lock: $SSH_HOST"
    if ! prepare_output=$(prepare_remote "$remote_staging" "$remote_final"); then
        error 'deployment stopped before upload; production current was not changed'
        error "if SSH was interrupted, check whether $DEPLOY_LOCK was left behind before retrying"
        return 1
    fi
    IFS=$'\t' read -r deployment_mode old_release <<<"$prepare_output"
    [[ ( $deployment_mode == EXISTING && -n $old_release ) || \
        ( $deployment_mode == FIRST_DEPLOYMENT && $old_release == "$NO_OLD_RELEASE" ) || \
        ( $deployment_mode == "$ALREADY_CURRENT" && -n $old_release ) ]] || {
        error 'remote preflight returned an invalid deployment mode'
        manual_check_hint
        return 1
    }
    if [[ $deployment_mode == "$ALREADY_CURRENT" ]]; then
        log "deployment already current; starting strict no-mutation resume verification: $remote_final"
        if ! verify_already_current_release "$remote_final" "$local_tree_sha256"; then
            error 'already-current release could not be proven identical and healthy; no deployment mutation was attempted'
            error "inspect current, lock, SHA256SUMS, service, and localhost health for $remote_final before retrying"
            return 1
        fi
        trap - INT TERM HUP
        log "source deployment already completed: deployment already current / RESUME $remote_final"
        check_public
        return
    fi
    staging_ready=1

    log "uploading release to staging: $remote_staging"
    if ! upload_release "$release_dir" "$remote_staging"; then
        error 'upload failed; production current was not changed'
        cleanup_before_activation "$remote_staging" || error "cleanup failed; inspect $DEPLOY_LOCK manually"
        staging_ready=0
        return 1
    fi

    log 'normalizing permissions and validating remote release'
    if ! validate_remote_release "$remote_staging"; then
        error 'remote validation failed; production current was not changed'
        cleanup_before_activation "$remote_staging" || error "cleanup failed; inspect $DEPLOY_LOCK manually"
        staging_ready=0
        return 1
    fi

    activation_started=1
    log "activating release: $remote_final"
    set +e
    activation_output=$(activate_release "$deployment_mode" "$old_release" "$remote_staging" "$remote_final" "$link_suffix")
    activation_rc=$?
    set -e
    trap - INT TERM HUP

    case $activation_rc in
        0)
            printf '%s\n' "$activation_output"
            log "source deployment succeeded: $remote_final"
            ;;
        20)
            printf '%s\n' "$activation_output"
            error 'new release failed, but the old release was rolled back and is healthy'
            return 1
            ;;
        22)
            printf '%s\n' "$activation_output"
            error 'FIRST_DEPLOYMENT health failure; no rollback was attempted and deployment evidence was preserved'
            manual_check_hint
            return 1
            ;;
        *)
            if [[ $activation_output == *'RESULT=FAILED_BEFORE_CURRENT_SWITCH'* ]]; then
                printf '%s\n' "$activation_output"
                error 'deployment failed before current was changed; this deployment staging and lock were cleaned'
            else
                manual_check_hint
            fi
            return 1
            ;;
    esac

    if [[ $deployment_mode == FIRST_DEPLOYMENT ]]; then
        log 'FIRST_DEPLOYMENT 源站已就绪；不执行 public acceptance，也不执行无旧缓存可刷新的 hostname purge'
        log "下一步：从外部主机使用 production hostname + --resolve 验收源站，通过后再启用正式 proxied DNS：$PUBLIC_URL"
        return 0
    fi
    check_public
}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
    main "$@"
fi
