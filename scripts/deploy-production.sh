#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

readonly SSH_HOST='aliyun'
readonly SERVICE_USER='go-tour'
readonly HEALTH_ATTEMPTS=12
readonly HEALTH_INTERVAL=3
readonly -a SSH_OPTIONS=(-o BatchMode=yes -o ConnectTimeout=10)

RELEASE_LOCALE=''
RELEASES_DIR=''
CURRENT_LINK=''
DEPLOY_LOCK=''
SERVICE=''
HEALTH_URL=''
PUBLIC_URL=''
PUBLIC_ACCEPTANCE_HINT=''

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

    case $locale in
        zh-CN)
            RELEASES_DIR='/data/go-tour/releases'
            CURRENT_LINK='/data/go-tour/current'
            DEPLOY_LOCK='/data/go-tour/.deploy.lock'
            SERVICE='go-tour.service'
            HEALTH_URL='http://127.0.0.1:3999/'
            PUBLIC_URL='https://go-dev.shuijingwanwq.com/'
            PUBLIC_ACCEPTANCE_HINT='inspect the CDN/reverse-proxy cache and refresh it manually if needed'
            ;;
        ja-JP)
            RELEASES_DIR='/data/go-tour-ja-JP/releases'
            CURRENT_LINK='/data/go-tour-ja-JP/current'
            DEPLOY_LOCK='/data/go-tour-ja-JP/.deploy.lock'
            SERVICE='go-tour-ja-JP.service'
            HEALTH_URL='http://127.0.0.1:4000/'
            PUBLIC_URL='https://ja-go-dev.shuijingwanwq.com/'
            PUBLIC_ACCEPTANCE_HINT='inspect the CDN/reverse-proxy cache and refresh it manually if needed'
            ;;
        de-DE)
            RELEASES_DIR='/data/go-tour-de-DE/releases'
            CURRENT_LINK='/data/go-tour-de-DE/current'
            DEPLOY_LOCK='/data/go-tour-de-DE/.deploy.lock'
            SERVICE='go-tour-de-DE.service'
            HEALTH_URL='http://127.0.0.1:4001/'
            PUBLIC_URL='https://de-go-dev.shuijingwanwq.com/'
            PUBLIC_ACCEPTANCE_HINT='inspect the CDN/reverse-proxy cache and refresh it manually if needed'
            ;;
        *)
            error "unsupported production locale in release.json: $locale"
            return 1
            ;;
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

    for command_name in basename curl date find python3 rsync sha256sum ssh; do
        command -v "$command_name" >/dev/null || {
            error "required local command is missing: $command_name"
            return 1
        }
    done
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
    local release_dir entry entry_name symlink unsupported
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

    if ! (cd -- "$release_dir" && sha256sum -c --strict SHA256SUMS); then
        error 'local SHA256 verification failed'
        return 1
    fi
    log "local release preflight passed: $release_dir"
}

prepare_remote() {
    local remote_staging=$1
    local remote_final=$2

    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$RELEASES_DIR" "$CURRENT_LINK" "$DEPLOY_LOCK" "$SERVICE" \
        "$remote_staging" "$remote_final" <<'REMOTE_PREPARE'
set -Eeuo pipefail
IFS=$'\n\t'

releases_dir=$1
current_link=$2
deploy_lock=$3
service=$4
staging=$5
final=$6

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
    old=''
else
    fail "current exists but is not a symlink: $current_link"
fi
systemctl cat "$service" >/dev/null || fail "systemd service does not exist: $service"
[[ $deployment_mode != EXISTING || $final != "$old" ]] || fail 'new release is already current'
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

upload_release() {
    local release_dir=$1
    local remote_staging=$2

    rsync -rlt --no-owner --no-group --no-perms --protect-args \
        -e 'ssh -o BatchMode=yes -o ConnectTimeout=10' -- \
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

(cd -- "$staging" && sha256sum -c --strict SHA256SUMS)
su -s /bin/sh -c 'test -x "$1" && test -r "$2" && test -r "$3"' \
    "$service_user" sh \
    "$staging/bin/tour" \
    "$staging/release.json" \
    "$staging/_content/tour/static/css/app.css"

printf '[deploy:remote] permissions and SHA256 verification passed\n'
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
        [[ -z $expected_old && ! -e $current_link && ! -L $current_link ]] || exit 1
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
    local http_code

    http_code=$(curl --silent --output /dev/null --write-out '%{http_code}' \
        --connect-timeout 5 --max-time 15 "$PUBLIC_URL" || true)
    if [[ $http_code != 200 ]]; then
        error "localhost is healthy, but public acceptance returned HTTP ${http_code:-000}: $PUBLIC_URL"
        error 'this CDN/reverse-proxy result did not roll back the healthy source release'
        error "$PUBLIC_ACCEPTANCE_HINT"
        return 1
    fi
    log "public acceptance passed: HTTP 200 $PUBLIC_URL"
    log "$PUBLIC_ACCEPTANCE_HINT"
}

main() {
    local release_input release_dir remote_name remote_final remote_staging old_release deployment_mode prepare_output
    local link_suffix activation_rc activation_output
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
    remote_name=$(release_name_from_path "$release_input") || return 1
    validate_local_release "$release_input" || return 1
    release_dir=$(cd -P -- "$release_input" && pwd -P)

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
    [[ ( $deployment_mode == EXISTING && -n $old_release ) || ( $deployment_mode == FIRST_DEPLOYMENT && -z $old_release ) ]] || {
        error 'remote preflight returned an invalid deployment mode'
        manual_check_hint
        return 1
    }
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

    check_public
}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
    main "$@"
fi
