#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repository_root=$(cd -P -- "$script_dir/.." && pwd -P)
deploy_script=$script_dir/deploy-shared-assets.sh
fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT

fail() {
    printf '[deploy-shared-assets-test] FAIL: %s\n' "$*" >&2
    exit 1
}

assert_contains() {
    local text=$1 want=$2
    [[ $text == *"$want"* ]] || fail "output does not contain: $want"
}

assert_not_contains() {
    local text=$1 unwanted=$2
    [[ $text != *"$unwanted"* ]] || fail "output unexpectedly contains: $unwanted"
}

receipt_result() {
    python3 - "$1" <<'PY'
import json
import sys
print(json.load(open(sys.argv[1], encoding="utf-8"))["deployment_result"])
PY
}

fake_bin=$fixture/bin
wwwroot=$fixture/wwwroot
origin=$wwwroot/assets-go-dev.shuijingwanwq.com
lock=$wwwroot/.assets-go-dev.deploy.lock
export_dir=$fixture/formal-export
connection_log=$fixture/connection.log
calculate_counter=$fixture/calculate.counter
mkdir -p -- "$fake_bin" "$wwwroot"

cat >"$fake_bin/ssh" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'ssh\t%s\n' "$*" >>"${FAKE_CONNECTION_LOG:-/dev/null}"
while [[ ${1:-} == -o ]]; do shift 2; done
shift
arguments=("$@")
for index in "${!arguments[@]}"; do
    if [[ ${arguments[index]} == /data/wwwroot* ]]; then
        arguments[index]=${arguments[index]/\/data\/wwwroot/$FAKE_WWWROOT}
    fi
    if [[ ${FAKE_REWRITE_ORIGIN_OUTSIDE:-0} == 1 && ${arguments[index]} == "$FAKE_WWWROOT/assets-go-dev.shuijingwanwq.com" ]]; then
        arguments[index]=$FAKE_OUTSIDE_ORIGIN
    fi
done
if [[ ${FAKE_PRECREATE_STAGING:-0} == 1 ]]; then
    for argument in "${arguments[@]}"; do
        [[ $argument == "$FAKE_WWWROOT"/.assets-go-dev.staging-* ]] && mkdir -p -- "$argument"
    done
fi
if [[ ${FAKE_REPLACE_ORIGIN_SYMLINK_BEFORE_UPDATE:-0} == 1 && ${#arguments[@]} -ge 12 ]]; then
    mapped_origin="$FAKE_WWWROOT/assets-go-dev.shuijingwanwq.com"
    mv -- "$mapped_origin" "$FAKE_WWWROOT/origin-displaced"
    ln -s -- "$FAKE_WWWROOT/other-site" "$mapped_origin"
fi
export SHARED_ASSETS_REMOTE_TEST_MODE=1
phase=other
has_origin=0
has_staging=0
has_lock=0
for argument in "${arguments[@]}"; do
    [[ $argument != "$FAKE_WWWROOT/assets-go-dev.shuijingwanwq.com" ]] || has_origin=1
    [[ $argument != "$FAKE_WWWROOT"/.assets-go-dev.staging-* ]] || has_staging=1
    [[ $argument != "$FAKE_WWWROOT/.assets-go-dev.deploy.lock" ]] || has_lock=1
done
if (( has_origin && has_staging && ! has_lock )); then
    phase=calculate
elif [[ ${#arguments[@]} == 8 ]]; then
    phase=backup
elif [[ ${#arguments[@]} -ge 12 ]]; then
    phase=update
fi
printf 'phase=%s\n' "$phase" >>"${FAKE_CONNECTION_LOG:-/dev/null}"
if [[ $phase == calculate && ${FAKE_CALCULATE_FAILURES:-0} -gt 0 ]]; then
    count=0
    [[ ! -f $FAKE_CALCULATE_COUNTER ]] || count=$(<"$FAKE_CALCULATE_COUNTER")
    count=$((count + 1))
    printf '%s\n' "$count" >"$FAKE_CALCULATE_COUNTER"
    (( count > FAKE_CALCULATE_FAILURES )) || exit 255
fi
if [[ ${FAKE_SSH_LOSE_PREPARE_RESULT:-0} == 1 && ${#arguments[@]} == 7 ]]; then
    "${arguments[@]}" >/dev/null
    exit 255
fi
if [[ ${FAKE_SSH_LOSE_BACKUP_RESULT:-0} == 1 && ${#arguments[@]} == 8 ]]; then
    "${arguments[@]}" >/dev/null
    exit 255
fi
if [[ $phase == update && ${FAKE_SSH_FAIL_UPDATE_BEFORE_EXEC:-0} == 1 ]]; then
    exit 255
fi
exec "${arguments[@]}"
SH

cat >"$fake_bin/rsync" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'rsync\t%s\n' "$*" >>"${FAKE_CONNECTION_LOG:-/dev/null}"
source_path=${@: -2:1}
destination=${@: -1}
remote_upload=0
if [[ $destination == *:* ]]; then
    remote_upload=1
    destination=${destination#*:}
fi
if [[ $destination == /data/wwwroot* ]]; then
    destination=${destination/\/data\/wwwroot/$FAKE_WWWROOT}
fi
if (( remote_upload )) && [[ ${FAKE_RSYNC_FAIL_UPLOAD:-0} == 1 ]]; then
    exit 31
fi
if [[ $source_path == *'.assets-go-dev.staging-'* && ${FAKE_RSYNC_FAIL_MUTATION:-0} == 1 ]]; then
    exit 32
fi
if [[ $source_path == *'.bak.'* && ${FAKE_RSYNC_FAIL_ROLLBACK:-0} == 1 ]]; then
    exit 33
fi
delete=()
for argument in "$@"; do
    [[ $argument == --delete ]] && delete=(--delete)
done
mkdir -p -- "$destination"
/usr/bin/rsync -rlt "${delete[@]}" -- "$source_path" "$destination"
if (( remote_upload )) && [[ ${FAKE_CORRUPT_STAGING:-0} == 1 ]]; then
    target=$(find "$destination" -type f ! -name SHA256SUMS -print -quit)
    printf 'corrupt staging\n' >>"$target"
fi
if (( remote_upload )) && [[ ${FAKE_REWRITE_STAGING_AND_MANIFEST:-0} == 1 ]]; then
    target=$(find "$destination" -type f ! -name SHA256SUMS -print -quit)
    printf 'self-consistent remote replacement\n' >"$target"
    (cd -- "${destination%/}" && find . -type f ! -name SHA256SUMS -printf '%P\0' | LC_ALL=C sort -z | xargs -0 sha256sum >SHA256SUMS)
fi
if (( remote_upload )) && [[ ${FAKE_REPLACE_STAGING_SYMLINK:-0} == 1 ]]; then
    link_path=${destination%/}
    rm -rf -- "$link_path"
    ln -s -- "$FAKE_WWWROOT/other-site" "$link_path"
fi
if [[ $source_path == *'.assets-go-dev.staging-'* && ${FAKE_CORRUPT_ORIGIN_AFTER_SYNC:-0} == 1 ]]; then
    target=$(find "$destination" -type f ! -name SHA256SUMS -print -quit)
    printf 'corrupt origin\n' >>"$target"
fi
SH

cat >"$fake_bin/cp" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
source_path=${@: -2:1}
destination=${@: -1}
if [[ $destination == *'.bak.'* && ${FAKE_CP_FAIL_BACKUP:-0} == 1 ]]; then
    exit 41
fi
exec /bin/cp "$@"
SH

chmod 0755 -- "$fake_bin/ssh" "$fake_bin/rsync" "$fake_bin/cp"

(cd -- "$repository_root" && GOCACHE=/tmp/go-tour-i18n-go-build \
    go run -mod=readonly ./cmd/tour-i18n assets export --output "$export_dir") >/dev/null

rehash_tree() {
    local tree=$1
    (cd -- "$tree" && find . -type f ! -name SHA256SUMS -printf '%P\0' | \
        LC_ALL=C sort -z | xargs -0 sha256sum >SHA256SUMS)
}

normalize_tree() {
    local tree=$1
    find "$tree" -type d -exec chmod 0755 {} +
    find "$tree" -type f -exec chmod 0644 {} +
}

reset_remote() {
    rm -rf -- "$origin" "$lock" "$wwwroot/origin-displaced"
    find "$wwwroot" -mindepth 1 -maxdepth 1 \
        \( -name '.assets-go-dev.staging-*' -o -name 'assets-go-dev.shuijingwanwq.com.bak.*' \) \
        -exec rm -rf -- {} +
    mkdir -p -- "$origin" "$wwwroot/other-site"
    /bin/cp -a -- "$export_dir/." "$origin/"
    printf 'keep\n' >"$wwwroot/other-site/marker"
    normalize_tree "$origin"
}

deploy_env() {
    env \
        PATH="$fake_bin:$PATH" \
        FAKE_WWWROOT="$wwwroot" \
        FAKE_CONNECTION_LOG="$connection_log" \
        FAKE_CALCULATE_COUNTER="$calculate_counter" \
        TMPDIR="$fixture" \
        "$@"
}

run_deploy() {
    deploy_env "$deploy_script" "$export_dir" 2>&1
}

# Local argument and export validation failures never reach a remote command.
if "$deploy_script" >/dev/null 2>&1; then fail 'no arguments accepted'; fi
if "$deploy_script" "$export_dir" extra >/dev/null 2>&1; then fail 'multiple arguments accepted'; fi
if "$deploy_script" "$fixture/missing" >/dev/null 2>&1; then fail 'missing input accepted'; fi
ln -s -- "$export_dir" "$fixture/export-link"
if "$deploy_script" "$fixture/export-link" >/dev/null 2>&1; then fail 'symlink input accepted'; fi

bad=$fixture/bad
/bin/cp -a -- "$export_dir" "$bad"
rm -- "$bad/SHA256SUMS"
if "$deploy_script" "$bad" >/dev/null 2>&1; then fail 'missing SHA256SUMS accepted'; fi
rm -rf -- "$bad"; /bin/cp -a -- "$export_dir" "$bad"
printf 'tamper\n' >>"$bad/tour/static/css/app.css"
if "$deploy_script" "$bad" >/dev/null 2>&1; then fail 'SHA mismatch accepted'; fi
rm -rf -- "$bad"; /bin/cp -a -- "$export_dir" "$bad"
ln -s -- tour/static/css/app.css "$bad/link"
if "$deploy_script" "$bad" >/dev/null 2>&1; then fail 'inner symlink accepted'; fi
rm -rf -- "$bad"; /bin/cp -a -- "$export_dir" "$bad"
mkfifo -- "$bad/unsupported-fifo"
if "$deploy_script" "$bad" >/dev/null 2>&1; then fail 'unsupported filesystem entry accepted'; fi
rm -rf -- "$bad"; /bin/cp -a -- "$export_dir" "$bad"
printf 'extra\n' >"$bad/extra.txt"; rehash_tree "$bad"
if "$deploy_script" "$bad" >/dev/null 2>&1; then fail 'unauthorized extra file accepted'; fi
rm -rf -- "$bad"; /bin/cp -a -- "$export_dir" "$bad"
rm -- "$bad/images/site-logo.png"; rehash_tree "$bad"
if "$deploy_script" "$bad" >/dev/null 2>&1; then fail 'incomplete file set accepted'; fi
rm -rf -- "$bad"; /bin/cp -a -- "$export_dir" "$bad"
printf '%064d  ../outside\n' 0 >"$bad/SHA256SUMS"
if "$deploy_script" "$bad" >/dev/null 2>&1; then fail 'manifest path traversal accepted'; fi

# No-op does not create a backup or purge list.
reset_remote
output=$(run_deploy) || fail 'no-op deployment failed'
assert_contains "$output" 'NO CHANGES'
assert_contains "$output" 'verification receipt:'
assert_contains "$output" 'verify-shared-assets-production.sh'
assert_not_contains "$output" 'https://assets-go-dev.shuijingwanwq.com/'
receipt=$export_dir.verification-receipt.json
[[ -f $receipt && ! -L $receipt ]] || fail 'no-op did not write a regular verification receipt'
[[ $(receipt_result "$receipt") == NO_CHANGES ]] || fail 'no-op receipt has wrong deployment result'
[[ ! -e $lock ]] || fail 'no-op left deployment lock'
[[ -z $(find "$wwwroot" -maxdepth 1 -name 'assets-go-dev.shuijingwanwq.com.bak.*' -print -quit) ]] || fail 'no-op created backup'

# SSH and rsync share one invocation-scoped multiplex transport and stable defaults.
first_control_path=$(grep -o 'ControlPath=[^ ]*' "$connection_log" | cut -d= -f2- | head -1)
[[ -n $first_control_path ]] || fail 'SSH ControlPath was not configured'
[[ ! -e ${first_control_path%/control} ]] || fail 'invocation-scoped SSH control directory was not cleaned'
for option in BatchMode=yes ConnectTimeout=10 ServerAliveInterval=5 ServerAliveCountMax=3 ConnectionAttempts=3 ControlMaster=auto ControlPersist=60; do
    grep -F "ssh" "$connection_log" | grep -F "$option" >/dev/null || fail "ssh missing stable option: $option"
    grep -F "rsync" "$connection_log" | grep -F "$option" >/dev/null || fail "rsync transport missing stable option: $option"
done
if grep -F 'ssh' "$connection_log" | grep -v -F "ControlPath=$first_control_path" >/dev/null; then fail 'ssh commands did not share one ControlPath'; fi
grep -F 'rsync' "$connection_log" | grep -F "ControlPath=$first_control_path" >/dev/null || fail 'rsync did not share the SSH ControlPath'
rm -f -- "$connection_log"
reset_remote
output=$(run_deploy) || fail 'second no-op deployment failed'
second_control_path=$(grep -o 'ControlPath=[^ ]*' "$connection_log" | cut -d= -f2- | head -1)
[[ -n $second_control_path && $second_control_path != "$first_control_path" ]] || fail 'concurrent-safe invocation ControlPath was reused'
[[ ! -e ${second_control_path%/control} ]] || fail 'second SSH control directory was not cleaned'

# calculate_changes is read-only and may recover from two transient SSH failures.
rm -f -- "$connection_log" "$calculate_counter"
reset_remote
output=$(deploy_env FAKE_CALCULATE_FAILURES=2 "$deploy_script" "$export_dir" 2>&1) || fail 'read-only calculate_changes retry did not recover'
[[ $(grep -c '^phase=calculate$' "$connection_log") == 3 ]] || fail 'calculate_changes did not use the bounded three attempts'
assert_contains "$output" 'attempt 1/3'
assert_contains "$output" 'attempt 2/3'

# Existing lock and staging fail closed.
reset_remote; mkdir -- "$lock"
if output=$(run_deploy); then fail 'existing lock accepted'; fi
assert_contains "$output" 'deployment lock exists'
reset_remote
if output=$(deploy_env FAKE_PRECREATE_STAGING=1 "$deploy_script" "$export_dir" 2>&1); then fail 'existing staging accepted'; fi
assert_contains "$output" 'remote staging already exists'
reset_remote
if output=$(deploy_env FAKE_SSH_LOSE_PREPARE_RESULT=1 "$deploy_script" "$export_dir" 2>&1); then fail 'uncertain prepare result accepted'; fi
assert_contains "$output" 'completion could not be confirmed'
[[ -d $lock && -n $(find "$wwwroot" -maxdepth 1 -name '.assets-go-dev.staging-*' -print -quit) ]] || fail 'uncertain prepare result did not preserve lock and staging'

# Upload and staging failures leave origin byte-identical and clean pre-mutation evidence.
reset_remote; before=$(sha256sum "$origin/tour/static/css/app.css")
if output=$(deploy_env FAKE_RSYNC_FAIL_UPLOAD=1 "$deploy_script" "$export_dir" 2>&1); then fail 'upload failure accepted'; fi
[[ $(sha256sum "$origin/tour/static/css/app.css") == "$before" && ! -e $lock ]] || fail 'upload failure changed origin or left lock'
reset_remote; before=$(sha256sum "$origin/tour/static/css/app.css")
if output=$(deploy_env FAKE_CORRUPT_STAGING=1 "$deploy_script" "$export_dir" 2>&1); then fail 'staging corruption accepted'; fi
[[ $(sha256sum "$origin/tour/static/css/app.css") == "$before" && ! -e $lock ]] || fail 'staging failure changed origin or left lock'
reset_remote
if output=$(deploy_env FAKE_REWRITE_STAGING_AND_MANIFEST=1 "$deploy_script" "$export_dir" 2>&1); then fail 'self-consistent staging replacement accepted'; fi
assert_contains "$output" 'manifest differs from the validated local export'
[[ ! -e $lock ]] || fail 'known staging manifest failure left lock'
reset_remote
if output=$(deploy_env FAKE_REPLACE_STAGING_SYMLINK=1 "$deploy_script" "$export_dir" 2>&1); then fail 'staging symlink replacement accepted'; fi
assert_contains "$output" 'remote staging validation failed'
[[ -L $(find "$wwwroot" -maxdepth 1 -name '.assets-go-dev.staging-*' -print -quit) && -d $lock ]] || fail 'staging symlink uncertainty did not preserve lock and evidence'

# Backup failure leaves origin unchanged and removes pre-mutation staging/lock.
reset_remote
printf 'old app css\n' >"$origin/tour/static/css/app.css"; rehash_tree "$origin"; before=$(sha256sum "$origin/tour/static/css/app.css")
if output=$(deploy_env FAKE_CP_FAIL_BACKUP=1 "$deploy_script" "$export_dir" 2>&1); then fail 'backup failure accepted'; fi
[[ $(sha256sum "$origin/tour/static/css/app.css") == "$before" && ! -e $lock ]] || fail 'backup failure changed origin or left lock'
reset_remote
printf 'old app css\n' >"$origin/tour/static/css/app.css"; rehash_tree "$origin"
rm -f -- "$connection_log"
if output=$(deploy_env FAKE_SSH_LOSE_BACKUP_RESULT=1 "$deploy_script" "$export_dir" 2>&1); then fail 'uncertain backup result accepted'; fi
assert_contains "$output" 'backup state could not be safely confirmed'
[[ $(grep -c '^phase=backup$' "$connection_log") == 1 ]] || fail 'uncertain backup was automatically retried'
[[ -d $lock && -n $(find "$wwwroot" -maxdepth 1 -name '.assets-go-dev.staging-*' -print -quit) && -n $(find "$wwwroot" -maxdepth 1 -name 'assets-go-dev.shuijingwanwq.com.bak.*' -print -quit) ]] || fail 'uncertain backup result did not preserve evidence'

# An uncertain update connection is never retried and preserves pre-command evidence.
reset_remote
printf 'old app css\n' >"$origin/tour/static/css/app.css"; rehash_tree "$origin"
rm -f -- "$connection_log"
if output=$(deploy_env FAKE_SSH_FAIL_UPDATE_BEFORE_EXEC=1 "$deploy_script" "$export_dir" 2>&1); then fail 'uncertain update result accepted'; fi
assert_contains "$output" '远端状态无法安全确定'
[[ $(grep -c '^phase=update$' "$connection_log") == 1 ]] || fail 'uncertain update was automatically retried'
[[ -d $lock && -n $(find "$wwwroot" -maxdepth 1 -name '.assets-go-dev.staging-*' -print -quit) && -n $(find "$wwwroot" -maxdepth 1 -name 'assets-go-dev.shuijingwanwq.com.bak.*' -print -quit) ]] || fail 'uncertain update did not preserve lock, staging, and backup'

# Added files: old 9-file origin becomes the exact formal 11-file tree.
reset_remote
rm -- "$origin/tour/static/go-dev/course-ad.css" "$origin/tour/static/go-dev/course-ad.js"
rehash_tree "$origin"
output=$(run_deploy) || fail 'added-file deployment failed'
assert_contains "$output" '/tour/static/go-dev/course-ad.css'
assert_contains "$output" '/tour/static/go-dev/course-ad.js'
assert_contains "$output" '/SHA256SUMS'
diff -qr -- "$export_dir" "$origin" >/dev/null || fail 'added-file deployment final tree mismatch'
(cd -- "$origin" && sha256sum -c --strict SHA256SUMS >/dev/null) || fail 'added-file origin SHA failed'

# Modified file emits only that asset plus the necessarily changed SHA256SUMS.
reset_remote
printf 'old app css\n' >"$origin/tour/static/css/app.css"; rehash_tree "$origin"
output=$(run_deploy) || fail 'modified-file deployment failed'
assert_contains "$output" '/tour/static/css/app.css'
assert_contains "$output" '/SHA256SUMS'
assert_contains "$output" 'Cloudflare HUMAN GATE'
[[ $(receipt_result "$receipt") == DEPLOYED ]] || fail 'deployment receipt has wrong deployment result'
python3 - "$receipt" <<'PY' || fail 'deployment receipt omitted modified path'
import json
import sys
if "tour/static/css/app.css" not in json.load(open(sys.argv[1], encoding="utf-8"))["changed_paths"]:
    raise SystemExit(1)
PY
assert_not_contains "$output" '/images/site-logo.png'
diff -qr -- "$export_dir" "$origin" >/dev/null || fail 'modified-file final tree mismatch'

# Deleted old public file is removed, reported for purge, and other sites are untouched.
reset_remote
mkdir -p -- "$origin/legacy"; printf 'legacy\n' >"$origin/legacy/old.css"; rehash_tree "$origin"; normalize_tree "$origin"
output=$(run_deploy) || fail 'deleted-file deployment failed'
assert_contains "$output" '/legacy/old.css'
[[ ! -e $origin/legacy/old.css ]] || fail 'deleted old asset remains in origin'
[[ $(<"$wwwroot/other-site/marker") == keep ]] || fail 'deployment changed another site'
diff -qr -- "$export_dir" "$origin" >/dev/null || fail 'deleted-file final tree mismatch'

# Mutation validation failure rolls back to the complete backup.
reset_remote
printf 'old app css\n' >"$origin/tour/static/css/app.css"; rehash_tree "$origin"; /bin/cp -a -- "$origin" "$fixture/before-rollback"
if output=$(deploy_env FAKE_RSYNC_FAIL_MUTATION=1 "$deploy_script" "$export_dir" 2>&1); then fail 'origin sync failure accepted'; fi
assert_contains "$output" 'rollback succeeded'
diff -qr -- "$fixture/before-rollback" "$origin" >/dev/null || fail 'sync-failure rollback did not restore old origin'
[[ ! -e $lock ]] || fail 'sync-failure rollback left lock'

rm -rf -- "$fixture/before-rollback"
reset_remote
printf 'old app css\n' >"$origin/tour/static/css/app.css"; rehash_tree "$origin"; /bin/cp -a -- "$origin" "$fixture/before-rollback"
if output=$(deploy_env FAKE_CORRUPT_ORIGIN_AFTER_SYNC=1 "$deploy_script" "$export_dir" 2>&1); then fail 'mutation validation failure accepted'; fi
assert_contains "$output" 'rollback succeeded'
diff -qr -- "$fixture/before-rollback" "$origin" >/dev/null || fail 'rollback did not restore old origin'
[[ ! -e $lock ]] || fail 'successful rollback left lock'

# Rollback failure preserves lock, staging, backup, and reports unknown state.
rm -rf -- "$fixture/before-rollback"
reset_remote
printf 'old app css\n' >"$origin/tour/static/css/app.css"; rehash_tree "$origin"
if output=$(deploy_env FAKE_CORRUPT_ORIGIN_AFTER_SYNC=1 FAKE_RSYNC_FAIL_ROLLBACK=1 "$deploy_script" "$export_dir" 2>&1); then fail 'rollback failure accepted'; fi
assert_contains "$output" '远端状态无法安全确定'
[[ -d $lock ]] || fail 'rollback failure removed lock'
[[ -n $(find "$wwwroot" -maxdepth 1 -name '.assets-go-dev.staging-*' -print -quit) ]] || fail 'rollback failure removed staging'
[[ -n $(find "$wwwroot" -maxdepth 1 -name 'assets-go-dev.shuijingwanwq.com.bak.*' -print -quit) ]] || fail 'rollback failure removed backup'

# Origin replacement after verified backup is rejected before rsync --delete and preserves evidence.
reset_remote
printf 'old app css\n' >"$origin/tour/static/css/app.css"; rehash_tree "$origin"
if output=$(deploy_env FAKE_REPLACE_ORIGIN_SYMLINK_BEFORE_UPDATE=1 "$deploy_script" "$export_dir" 2>&1); then fail 'origin symlink replacement accepted'; fi
assert_contains "$output" '远端状态无法安全确定'
[[ -L $origin && -d $wwwroot/origin-displaced && -d $lock ]] || fail 'origin TOCTOU did not preserve displaced origin and lock'
[[ $(<"$wwwroot/other-site/marker") == keep ]] || fail 'origin symlink replacement changed target site'

# Signal policy: pre-mutation cleanup; post-mutation evidence preservation.
reset_remote
mkdir -- "$lock" "$wwwroot/.assets-go-dev.staging-interrupt-before"
set +e
( export PATH="$fake_bin:$PATH" FAKE_WWWROOT="$wwwroot" SHARED_ASSETS_REMOTE_TEST_MODE=1; source "$deploy_script"; REMOTE_STAGING='/data/wwwroot/.assets-go-dev.staging-interrupt-before'; REMOTE_PREPARED=1; MUTATION_STARTED=0; on_signal ) >/dev/null 2>&1
status=$?
set -e
[[ $status == 130 && ! -e $lock && ! -e $wwwroot/.assets-go-dev.staging-interrupt-before ]] || fail 'pre-mutation interrupt cleanup failed'
reset_remote
mkdir -- "$lock" "$wwwroot/.assets-go-dev.staging-interrupt-after" "$wwwroot/assets-go-dev.shuijingwanwq.com.bak.interrupt-after"
set +e
( export PATH="$fake_bin:$PATH" FAKE_WWWROOT="$wwwroot" SHARED_ASSETS_REMOTE_TEST_MODE=1; source "$deploy_script"; REMOTE_STAGING='/data/wwwroot/.assets-go-dev.staging-interrupt-after'; REMOTE_BACKUP='/data/wwwroot/assets-go-dev.shuijingwanwq.com.bak.interrupt-after'; REMOTE_PREPARED=1; MUTATION_STARTED=1; on_signal ) >/dev/null 2>&1
status=$?
set -e
[[ $status == 130 && -d $lock && -d $wwwroot/.assets-go-dev.staging-interrupt-after && -d $wwwroot/assets-go-dev.shuijingwanwq.com.bak.interrupt-after ]] || fail 'post-mutation interrupt removed evidence'

# A test profile outside its fixed wwwroot fails before touching either tree.
reset_remote
outside=$fixture/outside/assets-go-dev.shuijingwanwq.com
mkdir -p -- "$outside"; /bin/cp -a -- "$export_dir/." "$outside/"; normalize_tree "$outside"
if output=$(deploy_env FAKE_REWRITE_ORIGIN_OUTSIDE=1 FAKE_OUTSIDE_ORIGIN="$outside" "$deploy_script" "$export_dir" 2>&1); then fail 'origin outside fixed wwwroot accepted'; fi
[[ $(<"$wwwroot/other-site/marker") == keep ]] || fail 'boundary failure changed another site'

# The production script contains no Cloudflare client or public acceptance call.
if grep -Eiq 'api\.cloudflare|wrangler|cloudflared|purge_cache|curl .*assets-go-dev' "$deploy_script"; then
    fail 'deployment script contains Cloudflare API/CLI or public acceptance logic'
fi

# Receipt generation is valid JSON even when its absolute export path has JSON-special characters.
special_export="$fixture/export \"quoted\" \\ path"
(
    source "$deploy_script"
    LOCAL_MANIFEST_SHA=$(sha256sum -- "$export_dir/SHA256SUMS"); LOCAL_MANIFEST_SHA=${LOCAL_MANIFEST_SHA%% *}
    write_verification_receipt "$special_export" DEPLOYED SHA256SUMS tour/static/css/app.css
) >/dev/null || fail 'special-character receipt generation failed'
special_receipt=$special_export.verification-receipt.json
python3 - "$special_receipt" "$special_export" <<'PY' || fail 'special-character receipt is invalid JSON'
import json
import sys
receipt = json.load(open(sys.argv[1], encoding="utf-8"))
if receipt["export_dir"] != sys.argv[2] or receipt["changed_paths"] != ["SHA256SUMS", "tour/static/css/app.css"]:
    raise SystemExit(1)
PY

printf '[deploy-shared-assets-test] PASS\n'
