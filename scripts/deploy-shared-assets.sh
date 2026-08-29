#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

readonly -a SSH_OPTIONS=(-o BatchMode=yes -o ConnectTimeout=10)
readonly PUBLIC_BASE_URL='https://assets-go-dev.shuijingwanwq.com'
readonly RECEIPT_SCHEMA='go-tour-i18n/shared-assets-production-receipt/v1'
readonly -a BOUNDARY_PATHS=(
    'tour/script.js'
    'tour/static/img/tree.png'
    'tour/static/partials/editor.html'
)
readonly SSH_HOST='aliyun'
readonly WWWROOT='/data/wwwroot'
readonly ORIGIN_ROOT='/data/wwwroot/assets-go-dev.shuijingwanwq.com'
readonly DEPLOY_LOCK='/data/wwwroot/.assets-go-dev.deploy.lock'

REMOTE_STAGING=''
REMOTE_BACKUP=''
ORIGIN_OWNER=''
ORIGIN_GROUP=''
DIRECTORY_MODE=''
FILE_MODE=''
LOCAL_MANIFEST_SHA=''
REMOTE_PREPARED=0
MUTATION_STARTED=0
BACKUP_STARTED=0

log() {
    printf '[deploy-shared-assets] %s\n' "$*"
}

error() {
    printf '[deploy-shared-assets] ERROR: %s\n' "$*" >&2
}

usage() {
    printf 'usage: %s <assets-export-dir>\n' "${0##*/}" >&2
}

manual_check_hint() {
    error '远端状态无法安全确定。不要直接重复部署；请先执行只读检查：'
    printf '%s\n' \
        "  ssh $SSH_HOST 'test -d $DEPLOY_LOCK && echo lock-present'" \
        "  ssh $SSH_HOST 'find $ORIGIN_ROOT -printf \"%y %u:%g %m %P\\n\" | sort'" \
        "  ssh $SSH_HOST 'cd $ORIGIN_ROOT && sha256sum -c --strict SHA256SUMS'" \
        "  ssh $SSH_HOST 'test -d $REMOTE_STAGING && find $REMOTE_STAGING -printf \"%y %u:%g %m %P\\n\" | sort'" \
        "  ssh $SSH_HOST 'test -d $REMOTE_BACKUP && find $REMOTE_BACKUP -printf \"%y %u:%g %m %P\\n\" | sort'" >&2
    error "deployment lock and available evidence were preserved: $DEPLOY_LOCK"
}

validate_local_tools() {
    local command_name
    for command_name in basename date find go mktemp mv python3 readlink rsync sha256sum ssh; do
        command -v "$command_name" >/dev/null || {
            error "required local command is missing: $command_name"
            return 1
        }
    done
}

is_safe_logical_path() {
    local path=$1
    [[ $path =~ ^[A-Za-z0-9._/-]+$ && $path != /* && $path != *'..'* && $path != *'//'* && $path != *'\\'* ]]
}

write_verification_receipt() {
    local export_dir=$1 result=$2 receipt temporary path
    shift 2
    local -a changed_paths=("$@")

    [[ $result == NO_CHANGES || $result == DEPLOYED ]] || {
        error 'cannot create receipt for an unknown deployment result'
        return 1
    }
    for path in "${changed_paths[@]}"; do
        is_safe_logical_path "$path" || {
            error "cannot create receipt with unsafe changed path: $path"
            return 1
        }
    done
    if [[ $result == NO_CHANGES && ${#changed_paths[@]} -ne 0 ]]; then
        error 'NO_CHANGES receipt cannot contain changed paths'
        return 1
    fi
    if [[ $result == DEPLOYED && ${#changed_paths[@]} -eq 0 ]]; then
        error 'DEPLOYED receipt must contain changed paths'
        return 1
    fi
    receipt="$export_dir.verification-receipt.json"
    temporary=$(mktemp "$receipt.tmp.XXXXXX") || return 1
    python3 - "$temporary" "$RECEIPT_SCHEMA" "$export_dir" "$LOCAL_MANIFEST_SHA" "$result" "$PUBLIC_BASE_URL" \
        "${BOUNDARY_PATHS[0]}" "${BOUNDARY_PATHS[1]}" "${BOUNDARY_PATHS[2]}" "${changed_paths[@]}" <<'PY' || {
import json
import sys

temporary, schema, export_dir, manifest_sha256, result, base_url, *paths = sys.argv[1:]
boundary_paths = paths[:3]
changed_paths = paths[3:]
with open(temporary, "w", encoding="utf-8") as receipt:
    json.dump({
        "schema": schema,
        "export_dir": export_dir,
        "manifest_sha256": manifest_sha256,
        "deployment_result": result,
        "production_base_url": base_url,
        "changed_paths": changed_paths,
        "boundary_paths": boundary_paths,
    }, receipt, ensure_ascii=False, sort_keys=True, separators=(",", ":"))
    receipt.write("\n")
PY
        rm -f -- "$temporary"
        return 1
    }
    chmod 0644 -- "$temporary"
    mv -f -- "$temporary" "$receipt"
    printf 'verification receipt: %s\n' "$receipt"
    printf 'next command: scripts/verify-shared-assets-production.sh %s\n' "$receipt"
}

validate_local_export() {
    local input_path=$1 export_dir script_dir repository_root symlink unsupported

    if [[ ! -d $input_path || -L $input_path ]]; then
        error "assets export path must be a real directory, not a symlink: $input_path"
        return 1
    fi
    export_dir=$(cd -P -- "$input_path" && pwd -P)
    [[ -f $export_dir/SHA256SUMS && ! -L $export_dir/SHA256SUMS ]] || {
        error 'assets export must contain a regular SHA256SUMS'
        return 1
    }
    symlink=$(find "$export_dir" -type l -print -quit)
    unsupported=$(find "$export_dir" ! -type d ! -type f -print -quit)
    if [[ -n $symlink || -n $unsupported ]]; then
        error "assets export contains a symlink or unsupported entry: ${symlink:-$unsupported}"
        return 1
    fi
    script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
    repository_root=$(cd -P -- "$script_dir/.." && pwd -P)
    if ! (cd -- "$repository_root" && GOCACHE=${GOCACHE:-/tmp/go-tour-i18n-go-build} \
        go run -mod=readonly ./cmd/tour-i18n assets validate --input "$export_dir" >/dev/null); then
        error 'input is not the repository formal shared-assets export'
        return 1
    fi
    if ! (cd -- "$export_dir" && sha256sum -c --strict SHA256SUMS >/dev/null); then
        error 'local SHA256 verification failed'
        return 1
    fi
    printf '%s\n' "$export_dir"
}

prepare_remote() {
    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$WWWROOT" "$ORIGIN_ROOT" "$DEPLOY_LOCK" "$REMOTE_STAGING" <<'REMOTE_PREPARE'
set -Eeuo pipefail
IFS=$'\n\t'
wwwroot=$1
origin=$2
lock=$3
staging=$4

fail() { printf '[deploy-shared-assets:remote] ERROR: %s\n' "$*" >&2; exit 1; }
validate_tree() {
    local root=$1
    [[ -d $root && ! -L $root && -f $root/SHA256SUMS && ! -L $root/SHA256SUMS ]] || return 1
    [[ -z $(find "$root" -type l -print -quit) ]] || return 1
    [[ -z $(find "$root" ! -type d ! -type f -print -quit) ]] || return 1
    awk 'NF != 2 || length($1) != 64 || $1 !~ /^[0-9a-f]+$/ || $2 !~ /^[A-Za-z0-9._\/-]+$/ || $2 ~ /^\// || $2 ~ /(^|\/)\.\.?($|\/)/ || $2 ~ /\/\// || $2 ~ /\\/ {exit 1}' "$root/SHA256SUMS" || return 1
    diff -u \
        <(awk '{print $2}' "$root/SHA256SUMS" | LC_ALL=C sort) \
        <(cd -- "$root" && find . -type f ! -name SHA256SUMS -printf '%P\n' | LC_ALL=C sort) >/dev/null || return 1
    (cd -- "$root" && sha256sum -c --strict SHA256SUMS >/dev/null) || return 1
}

[[ $(id -u) == 0 || ${SHARED_ASSETS_REMOTE_TEST_MODE:-0} == 1 ]] || fail 'remote SSH user must be root'
[[ -d $wwwroot && ! -L $wwwroot ]] || fail "wwwroot must be a real directory: $wwwroot"
[[ -d $origin && ! -L $origin ]] || fail "origin must be a real directory: $origin"
[[ $(readlink -f -- "$wwwroot") == "$wwwroot" && $(readlink -f -- "$origin") == "$origin" ]] || fail 'wwwroot or origin has a non-canonical path component'
[[ $origin == "$wwwroot/assets-go-dev.shuijingwanwq.com" ]] || fail 'origin is not the fixed assets-go-dev path'
[[ $lock == "$wwwroot/.assets-go-dev.deploy.lock" ]] || fail 'lock is not the fixed assets-go-dev lock path'
[[ $(dirname -- "$origin") == "$wwwroot" ]] || fail 'origin is outside the fixed wwwroot'
[[ $(dirname -- "$lock") == "$wwwroot" && $(dirname -- "$staging") == "$wwwroot" ]] || fail 'lock or staging is outside the fixed wwwroot'
[[ $(basename -- "$staging") == .assets-go-dev.staging-* ]] || fail 'unsafe staging name'
[[ ! -e $staging && ! -L $staging ]] || fail "remote staging already exists: $staging"
for command_name in awk basename chown chmod cmp cp diff dirname find mkdir readlink rm rmdir rsync sha256sum sort stat; do
    command -v "$command_name" >/dev/null || fail "required remote command is missing: $command_name"
done

owner=$(stat -c %U -- "$origin")
group=$(stat -c %G -- "$origin")
directory_mode=$(stat -c %a -- "$origin")
[[ -f $origin/SHA256SUMS && ! -L $origin/SHA256SUMS ]] || fail 'origin SHA256SUMS is missing or invalid'
file_mode=$(stat -c %a -- "$origin/SHA256SUMS")
[[ -z $(find "$origin" -type l -print -quit) ]] || fail 'origin contains a symlink'
[[ -z $(find "$origin" ! -type d ! -type f -print -quit) ]] || fail 'origin contains an unsupported entry'
validate_tree "$origin" || fail 'existing origin file set or SHA256SUMS is invalid'
[[ $(find "$origin" -type d -printf '%u:%g %m\n' | LC_ALL=C sort -u) == "$owner:$group $directory_mode" ]] || fail 'origin directory permissions are not uniform'
[[ $(find "$origin" -type f -printf '%u:%g %m\n' | LC_ALL=C sort -u) == "$owner:$group $file_mode" ]] || fail 'origin file permissions are not uniform'

if ! mkdir -- "$lock"; then
    fail "deployment lock exists; another or unfinished deployment needs manual inspection: $lock"
fi
if ! mkdir -m 0700 -- "$staging"; then
    rmdir -- "$lock" || true
    fail "cannot create staging: $staging"
fi
[[ -d $lock && ! -L $lock && $(readlink -f -- "$lock") == "$lock" ]] || fail 'deployment lock is not a canonical real directory'
[[ -d $staging && ! -L $staging && $(readlink -f -- "$staging") == "$staging" ]] || fail 'staging is not a canonical real directory'
printf '%s\t%s\t%s\t%s\n' "$owner" "$group" "$directory_mode" "$file_mode"
REMOTE_PREPARE
}

upload_export() {
    local export_dir=$1
    rsync -rlt --no-owner --no-group --no-perms --protect-args \
        -e 'ssh -o BatchMode=yes -o ConnectTimeout=10' -- \
        "$export_dir/" "$SSH_HOST:$REMOTE_STAGING/"
}

validate_remote_staging() {
    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$WWWROOT" "$REMOTE_STAGING" "$ORIGIN_OWNER" "$ORIGIN_GROUP" \
        "$DIRECTORY_MODE" "$FILE_MODE" "$LOCAL_MANIFEST_SHA" <<'REMOTE_VALIDATE'
set -Eeuo pipefail
IFS=$'\n\t'
wwwroot=$1
tree=$2
owner=$3
group=$4
directory_mode=$5
file_mode=$6
expected_manifest_sha=$7

fail() { printf '[deploy-shared-assets:remote] ERROR: %s\n' "$*" >&2; exit 1; }
validate_tree() {
    local root=$1
    [[ -d $root && ! -L $root && -f $root/SHA256SUMS && ! -L $root/SHA256SUMS ]] || return 1
    [[ -z $(find "$root" -type l -print -quit) ]] || return 1
    [[ -z $(find "$root" ! -type d ! -type f -print -quit) ]] || return 1
    awk 'NF != 2 || length($1) != 64 || $1 !~ /^[0-9a-f]+$/ || $2 !~ /^[A-Za-z0-9._\/-]+$/ || $2 ~ /^\// || $2 ~ /(^|\/)\.\.?($|\/)/ || $2 ~ /\/\// || $2 ~ /\\/ {exit 1}' "$root/SHA256SUMS" || return 1
    diff -u \
        <(awk '{print $2}' "$root/SHA256SUMS" | LC_ALL=C sort) \
        <(cd -- "$root" && find . -type f ! -name SHA256SUMS -printf '%P\n' | LC_ALL=C sort) >/dev/null || return 1
    (cd -- "$root" && sha256sum -c --strict SHA256SUMS) || return 1
}

[[ $(dirname -- "$tree") == "$wwwroot" && $(basename -- "$tree") == .assets-go-dev.staging-* ]] || fail 'staging path escaped fixed boundary'
[[ -d $tree && ! -L $tree && $(readlink -f -- "$tree") == "$tree" ]] || fail 'staging is not a canonical real directory'
[[ $(sha256sum -- "$tree/SHA256SUMS" | awk '{print $1}') == "$expected_manifest_sha" ]] || fail 'remote staging manifest differs from the validated local export'
chown -R "$owner:$group" -- "$tree"
find "$tree" -type d -exec chmod "$directory_mode" {} +
find "$tree" -type f -exec chmod "$file_mode" {} +
validate_tree "$tree" || fail 'remote staging validation failed'
[[ $(find "$tree" -type d -printf '%u:%g %m\n' | LC_ALL=C sort -u) == "$owner:$group $directory_mode" ]] || fail 'staging directory permissions differ from origin model'
[[ $(find "$tree" -type f -printf '%u:%g %m\n' | LC_ALL=C sort -u) == "$owner:$group $file_mode" ]] || fail 'staging file permissions differ from origin model'
printf '[deploy-shared-assets:remote] staging permissions, file set, and SHA256 verified\n'
REMOTE_VALIDATE
}

calculate_changes() {
    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$WWWROOT" "$ORIGIN_ROOT" "$REMOTE_STAGING" <<'REMOTE_CHANGES'
set -Eeuo pipefail
IFS=$'\n\t'
wwwroot=$1
origin=$2
staging=$3
[[ $origin == "$wwwroot/assets-go-dev.shuijingwanwq.com" ]]
[[ $(dirname -- "$staging") == "$wwwroot" && $(basename -- "$staging") == .assets-go-dev.staging-* ]]
[[ -d $origin && ! -L $origin && $(readlink -f -- "$origin") == "$origin" ]]
[[ -d $staging && ! -L $staging && $(readlink -f -- "$staging") == "$staging" ]]
while IFS= read -r -d '' path; do
    [[ -n $path && $path =~ ^[A-Za-z0-9._/-]+$ && $path != /* && $path != *'..'* && $path != *'//'* && $path != *'\'* ]]
    if [[ ! -e $origin/$path ]]; then
        printf '%s\n' "$path"
    elif [[ ! -e $staging/$path ]]; then
        printf '%s\n' "$path"
    elif ! cmp -s -- "$origin/$path" "$staging/$path"; then
        printf '%s\n' "$path"
    fi
done < <({ cd -- "$origin" && find . -type f -printf '%P\0'; cd -- "$staging" && find . -type f -printf '%P\0'; } | LC_ALL=C sort -zu)
REMOTE_CHANGES
}

create_backup() {
    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$WWWROOT" "$ORIGIN_ROOT" "$DEPLOY_LOCK" "$REMOTE_STAGING" "$REMOTE_BACKUP" <<'REMOTE_BACKUP'
set -Eeuo pipefail
IFS=$'\n\t'
wwwroot=$1
origin=$2
lock=$3
staging=$4
backup=$5
fail() { printf '[deploy-shared-assets:remote] ERROR: %s\n' "$*" >&2; exit 1; }
[[ $origin == "$wwwroot/assets-go-dev.shuijingwanwq.com" ]] || fail 'origin is not the fixed assets-go-dev path'
[[ $lock == "$wwwroot/.assets-go-dev.deploy.lock" ]] || fail 'lock is not the fixed assets-go-dev lock path'
[[ $(dirname -- "$staging") == "$wwwroot" && $(basename -- "$staging") == .assets-go-dev.staging-* ]] || fail 'staging path escaped fixed boundary'
[[ -d $lock && -d $staging ]] || fail 'deployment lock or staging disappeared before backup'
[[ ! -L $lock && ! -L $staging && ! -L $origin ]] || fail 'lock, staging, or origin became a symlink before backup'
[[ $(readlink -f -- "$lock") == "$lock" && $(readlink -f -- "$staging") == "$staging" && $(readlink -f -- "$origin") == "$origin" ]] || fail 'lock, staging, or origin is no longer canonical'
[[ $(dirname -- "$backup") == "$wwwroot" && $(basename -- "$backup") == assets-go-dev.shuijingwanwq.com.bak.* ]] || fail 'backup path escaped fixed boundary'
[[ ! -e $backup && ! -L $backup ]] || fail "backup already exists: $backup"
if ! cp -a -- "$origin" "$backup"; then
    rm -rf -- "$backup"
    printf '[deploy-shared-assets:remote] ERROR: cannot create complete origin backup\n' >&2
    exit 10
fi
[[ -d $backup && ! -L $backup && $(readlink -f -- "$backup") == "$backup" ]] || {
    rm -rf -- "$backup"
    fail 'backup is not a canonical real directory'
}
if ! diff -qr -- "$origin" "$backup" >/dev/null || ! (cd -- "$backup" && sha256sum -c --strict SHA256SUMS); then
    rm -rf -- "$backup"
    printf '[deploy-shared-assets:remote] ERROR: backup verification failed\n' >&2
    exit 10
fi
printf '[deploy-shared-assets:remote] complete backup verified: %s\n' "$backup"
REMOTE_BACKUP
}

cleanup_before_mutation() {
    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$WWWROOT" "$DEPLOY_LOCK" "$REMOTE_STAGING" <<'REMOTE_CLEANUP'
set -Eeuo pipefail
wwwroot=$1
lock=$2
staging=$3
[[ $(dirname -- "$lock") == "$wwwroot" && $(dirname -- "$staging") == "$wwwroot" ]]
[[ $lock == "$wwwroot/.assets-go-dev.deploy.lock" ]]
[[ $(basename -- "$staging") == .assets-go-dev.staging-* ]]
[[ -d $lock && ! -L $lock && $(readlink -f -- "$lock") == "$lock" ]]
[[ -d $staging && ! -L $staging && $(readlink -f -- "$staging") == "$staging" ]]
rm -rf -- "$staging"
rmdir -- "$lock"
REMOTE_CLEANUP
}

update_origin() {
    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$WWWROOT" "$ORIGIN_ROOT" "$DEPLOY_LOCK" "$REMOTE_STAGING" "$REMOTE_BACKUP" \
        "$ORIGIN_OWNER" "$ORIGIN_GROUP" "$DIRECTORY_MODE" "$FILE_MODE" <<'REMOTE_UPDATE'
set -Eeuo pipefail
IFS=$'\n\t'
wwwroot=$1
origin=$2
lock=$3
staging=$4
backup=$5
owner=$6
group=$7
directory_mode=$8
file_mode=$9

validate_tree() {
    local root=$1
    [[ -d $root && ! -L $root && -f $root/SHA256SUMS && ! -L $root/SHA256SUMS ]] || return 1
    [[ -z $(find "$root" -type l -print -quit) ]] || return 1
    [[ -z $(find "$root" ! -type d ! -type f -print -quit) ]] || return 1
    awk 'NF != 2 || length($1) != 64 || $1 !~ /^[0-9a-f]+$/ || $2 !~ /^[A-Za-z0-9._\/-]+$/ || $2 ~ /^\// || $2 ~ /(^|\/)\.\.?($|\/)/ || $2 ~ /\/\// || $2 ~ /\\/ {exit 1}' "$root/SHA256SUMS" || return 1
    diff -u \
        <(awk '{print $2}' "$root/SHA256SUMS" | LC_ALL=C sort) \
        <(cd -- "$root" && find . -type f ! -name SHA256SUMS -printf '%P\n' | LC_ALL=C sort) >/dev/null || return 1
    (cd -- "$root" && sha256sum -c --strict SHA256SUMS) || return 1
}
normalize_and_validate() {
    local root=$1
    chown -R "$owner:$group" -- "$root" || return 1
    find "$root" -type d -exec chmod "$directory_mode" {} + || return 1
    find "$root" -type f -exec chmod "$file_mode" {} + || return 1
    validate_tree "$root" || return 1
    [[ $(find "$root" -type d -printf '%u:%g %m\n' | LC_ALL=C sort -u) == "$owner:$group $directory_mode" ]] || return 1
    [[ $(find "$root" -type f -printf '%u:%g %m\n' | LC_ALL=C sort -u) == "$owner:$group $file_mode" ]] || return 1
}
rollback() {
    [[ -d $origin && ! -L $origin && $(readlink -f -- "$origin") == "$origin" ]] || return 1
    [[ -d $backup && ! -L $backup && $(readlink -f -- "$backup") == "$backup" ]] || return 1
    rsync -rlt --delete -- "$backup/" "$origin/" || return 1
    normalize_and_validate "$origin" || return 1
    diff -qr -- "$backup" "$origin" >/dev/null || return 1
}

[[ $(dirname -- "$origin") == "$wwwroot" && $(dirname -- "$staging") == "$wwwroot" && $(dirname -- "$backup") == "$wwwroot" ]] || exit 21
[[ $origin == "$wwwroot/assets-go-dev.shuijingwanwq.com" && $lock == "$wwwroot/.assets-go-dev.deploy.lock" ]] || exit 21
[[ $(basename -- "$staging") == .assets-go-dev.staging-* && $(basename -- "$backup") == assets-go-dev.shuijingwanwq.com.bak.* ]] || exit 21
[[ -d $lock && ! -L $lock && $(readlink -f -- "$lock") == "$lock" ]] || exit 21
[[ -d $origin && ! -L $origin && $(readlink -f -- "$origin") == "$origin" ]] || exit 21
[[ -d $staging && ! -L $staging && $(readlink -f -- "$staging") == "$staging" ]] || exit 21
[[ -d $backup && ! -L $backup && $(readlink -f -- "$backup") == "$backup" ]] || exit 21

printf '[deploy-shared-assets:remote] production mutation starting\n'
if rsync -rlt --delete -- "$staging/" "$origin/" && normalize_and_validate "$origin"; then
    rm -rf -- "$staging"
    rmdir -- "$lock"
    printf '[deploy-shared-assets:remote] RESULT=DEPLOYED backup=%s\n' "$backup"
    exit 0
fi

printf '[deploy-shared-assets:remote] deployment validation failed; attempting rollback\n' >&2
if rollback; then
    rm -rf -- "$staging"
    rmdir -- "$lock"
    printf '[deploy-shared-assets:remote] RESULT=ROLLED_BACK backup=%s\n' "$backup" >&2
    exit 20
fi

printf '[deploy-shared-assets:remote] RESULT=UNKNOWN rollback failed; preserving lock, staging, and backup\n' >&2
exit 21
REMOTE_UPDATE
}

on_signal() {
    trap - HUP INT TERM
    if (( REMOTE_PREPARED )) && (( ! MUTATION_STARTED )) && (( ! BACKUP_STARTED )); then
        cleanup_before_mutation >/dev/null 2>&1 || manual_check_hint
    elif (( MUTATION_STARTED || BACKUP_STARTED )); then
        manual_check_hint
    fi
    exit 130
}

main() {
    local export_dir token prepare_output update_status backup_status changes_output path
    local -a changed_paths=()

    (( $# == 1 )) || { usage; return 2; }
    validate_local_tools
    export_dir=$(validate_local_export "$1")
    LOCAL_MANIFEST_SHA=$(sha256sum -- "$export_dir/SHA256SUMS")
    LOCAL_MANIFEST_SHA=${LOCAL_MANIFEST_SHA%% *}
    [[ $LOCAL_MANIFEST_SHA =~ ^[0-9a-f]{64}$ ]] || {
        error 'cannot determine local SHA256SUMS digest'
        return 1
    }

    token=$(date -u +%Y%m%dT%H%M%SZ)-$$-$RANDOM
    [[ $token =~ ^[0-9]{8}T[0-9]{6}Z-[0-9]+-[0-9]+$ ]] || {
        error 'cannot derive a safe deployment token'
        return 1
    }
    REMOTE_STAGING="$WWWROOT/.assets-go-dev.staging-$token"
    REMOTE_BACKUP="$WWWROOT/assets-go-dev.shuijingwanwq.com.bak.$token"
    trap on_signal HUP INT TERM

    if ! prepare_output=$(prepare_remote); then
        error 'remote preflight failed or its completion could not be confirmed'
        manual_check_hint
        return 1
    fi
    REMOTE_PREPARED=1
    IFS=$'\t' read -r ORIGIN_OWNER ORIGIN_GROUP DIRECTORY_MODE FILE_MODE <<<"$prepare_output"
    [[ -n $ORIGIN_OWNER && -n $ORIGIN_GROUP && $DIRECTORY_MODE =~ ^[0-7]{3,4}$ && $FILE_MODE =~ ^[0-7]{3,4}$ ]] || {
        error 'remote permission preflight returned invalid data'
        cleanup_before_mutation || manual_check_hint
        return 1
    }

    if ! upload_export "$export_dir"; then
        error 'upload failed before production mutation'
        cleanup_before_mutation || manual_check_hint
        return 1
    fi
    if ! validate_remote_staging; then
        error 'remote staging validation failed before production mutation'
        cleanup_before_mutation || manual_check_hint
        return 1
    fi

    if ! changes_output=$(calculate_changes); then
        error 'cannot calculate changed assets before production mutation'
        cleanup_before_mutation || manual_check_hint
        return 1
    fi
    if [[ -n $changes_output ]]; then
        mapfile -t changed_paths <<<"$changes_output"
    fi
    if (( ${#changed_paths[@]} == 0 )); then
        cleanup_before_mutation
        log 'NO CHANGES: production origin already matches the formal export; no backup or purge is required'
        write_verification_receipt "$export_dir" NO_CHANGES
        return 0
    fi
    BACKUP_STARTED=1
    set +e
    create_backup
    backup_status=$?
    set -e
    case $backup_status in
        0)
            BACKUP_STARTED=0
            ;;
        10)
            BACKUP_STARTED=0
            error 'backup failed before production mutation; origin was not changed'
            cleanup_before_mutation || manual_check_hint
            return 1
            ;;
        *)
            error 'backup state could not be safely confirmed; origin mutation was not started'
            manual_check_hint
            return 1
            ;;
    esac

    MUTATION_STARTED=1
    set +e
    update_origin
    update_status=$?
    set -e
    case $update_status in
        0) ;;
        20)
            error 'origin deployment failed and rollback succeeded; old production content was restored'
            return 1
            ;;
        *)
            manual_check_hint
            return 1
            ;;
    esac

    log 'SHARED ASSETS ORIGIN DEPLOYMENT COMPLETED'
    write_verification_receipt "$export_dir" DEPLOYED "${changed_paths[@]}"
    printf '\nCloudflare HUMAN GATE:\n'
    printf '请在 Cloudflare Dashboard 对以下 URL 执行 Custom Purge：\n'
    for path in "${changed_paths[@]}"; do
        printf '%s/%s\n' "$PUBLIC_BASE_URL" "$path"
    done
    printf '完成后运行上述 verification receipt 对应的唯一后续命令。\n'
}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
    main "$@"
fi
