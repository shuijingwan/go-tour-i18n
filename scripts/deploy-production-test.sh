#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
# shellcheck source=deploy-production.sh
source "$script_dir/deploy-production.sh"

fail() {
    printf '[deploy-test] FAIL: %s\n' "$*" >&2
    exit 1
}

assert_profile() {
    local locale=$1 releases=$2 current=$3 lock=$4 service=$5 health=$6 public=$7
    select_deployment_profile "$locale" || fail "profile rejected: $locale"
    [[ $RELEASES_DIR == "$releases" ]] || fail "$locale releases profile"
    [[ $CURRENT_LINK == "$current" ]] || fail "$locale current profile"
    [[ $DEPLOY_LOCK == "$lock" ]] || fail "$locale lock profile"
    [[ $SERVICE == "$service" ]] || fail "$locale service profile"
    [[ $HEALTH_URL == "$health" ]] || fail "$locale health profile"
    [[ $PUBLIC_URL == "$public" ]] || fail "$locale public profile"
    [[ $EXPECTED_DEPLOYMENT_MODE == EXISTING ]] || fail "$locale lifecycle profile"
    [[ $PUBLIC_ACCEPTANCE_HINT == 'inspect the CDN/reverse-proxy cache and refresh it manually if needed' ]] || fail "$locale public acceptance hint"
}

assert_profile zh-CN /data/go-tour/releases /data/go-tour/current \
    /data/go-tour/.deploy.lock go-tour.service http://127.0.0.1:3999/ \
    https://go-dev.shuijingwanwq.com/
assert_profile ja-JP /data/go-tour-ja-JP/releases /data/go-tour-ja-JP/current \
    /data/go-tour-ja-JP/.deploy.lock go-tour-ja-JP.service http://127.0.0.1:4000/ \
    https://ja-go-dev.shuijingwanwq.com/
assert_profile de-DE /data/go-tour-de-DE/releases /data/go-tour-de-DE/current \
    /data/go-tour-de-DE/.deploy.lock go-tour-de-DE.service http://127.0.0.1:4001/ \
    https://de-go-dev.shuijingwanwq.com/
assert_profile fr-FR /data/go-tour-fr-FR/releases /data/go-tour-fr-FR/current \
    /data/go-tour-fr-FR/.deploy.lock go-tour-fr-FR.service http://127.0.0.1:4002/ \
    https://fr-go-dev.shuijingwanwq.com/
assert_profile ko-KR /data/go-tour-ko-KR/releases /data/go-tour-ko-KR/current \
    /data/go-tour-ko-KR/.deploy.lock go-tour-ko-KR.service http://127.0.0.1:4003/ \
    https://ko-go-dev.shuijingwanwq.com/

if select_deployment_profile zz-ZZ 2>/dev/null; then
    fail 'unsupported locale was accepted'
fi

main_source=$(declare -f main)
[[ $main_source == *'deployment_mode == FIRST_DEPLOYMENT'* && $main_source == *'不执行 public acceptance'* ]] || \
    fail 'FIRST_DEPLOYMENT does not explicitly skip pre-DNS public acceptance/purge'
[[ $main_source == *'check_public'* ]] || fail 'EXISTING_DEPLOYMENT public acceptance was removed'

fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT

TMPDIR=$fixture setup_ssh_multiplex || fail 'SSH multiplex setup failed'
control_dir=$SSH_CONTROL_DIR
[[ -d $control_dir && $SSH_CONTROL_PATH == "$control_dir/control" && $RSYNC_SSH_COMMAND == *ControlPath* ]] || \
    fail 'SSH multiplex setup did not create invocation-scoped options'
cleanup_ssh_multiplex
[[ ! -e $control_dir && -z $SSH_CONTROL_DIR && -z $SSH_CONTROL_PATH ]] || \
    fail 'SSH multiplex cleanup left invocation-scoped state'

fake_bin=$fixture/bin
mkdir -p -- "$fake_bin"
cat >"$fake_bin/id" <<'SH'
#!/usr/bin/env bash
[[ ${1:-} == -u ]] && { printf '0\n'; exit 0; }
exec /usr/bin/id "$@"
SH
cat >"$fake_bin/systemctl" <<'SH'
#!/usr/bin/env bash
case $1 in
    cat) exit 0 ;;
    restart)
        [[ -z ${FAKE_RESTART_MARKER:-} ]] || : >"$FAKE_RESTART_MARKER"
        [[ ${FAKE_RESTART_FAIL:-0} == 0 ]] && exit 0 || exit 1
        ;;
    is-active)
        printf '%s\n' "${FAKE_SERVICE_STATE:-active}"
        [[ ${FAKE_SERVICE_STATE:-active} == active ]]
        ;;
    status|show) exit 0 ;;
esac
exit 1
SH
cat >"$fake_bin/curl" <<'SH'
#!/usr/bin/env bash
url=${!#}
if [[ $url == https://* ]]; then
    count_file=${FAKE_PUBLIC_COUNTER:?}
    status_sequence=${FAKE_PUBLIC_HTTP_SEQUENCE:-200}
    exit_sequence=${FAKE_PUBLIC_EXIT_SEQUENCE:-0}
else
    count_file=${FAKE_HEALTH_COUNTER:?}
    status_sequence=${FAKE_HEALTH_HTTP_SEQUENCE:-}
    exit_sequence=0
fi
count=0; [[ -f $count_file ]] && count=$(<"$count_file")
count=$((count + 1)); printf '%s' "$count" >"$count_file"
if [[ -n $status_sequence ]]; then
    IFS=, read -r -a statuses <<<"$status_sequence"
    index=$((count - 1)); (( index < ${#statuses[@]} )) || index=$((${#statuses[@]} - 1))
    status=${statuses[index]}
else
    if (( count <= ${FAKE_HTTP_FAIL_CALLS:-0} )); then status=500; else status=200; fi
fi
IFS=, read -r -a exits <<<"$exit_sequence"
index=$((count - 1)); (( index < ${#exits[@]} )) || index=$((${#exits[@]} - 1))
printf '%s' "$status"
exit "${exits[index]}"
SH
cat >"$fake_bin/sleep" <<'SH'
#!/usr/bin/env bash
exit 0
SH
cat >"$fake_bin/ssh" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
while [[ ${1:-} == -o ]]; do shift 2; done
shift
arguments=("$@")
for index in "${!arguments[@]}"; do
    case ${arguments[index]} in
        /data/go-tour/releases) arguments[index]=$FAKE_ZH_RELEASES ;;
        /data/go-tour/releases/*) arguments[index]="$FAKE_ZH_RELEASES/${arguments[index]#/data/go-tour/releases/}" ;;
        /data/go-tour/current) arguments[index]=$FAKE_ZH_CURRENT ;;
        /data/go-tour/.deploy.lock) arguments[index]=$FAKE_ZH_LOCK ;;
        /data/go-tour-ja-JP/releases) arguments[index]=$FAKE_JA_RELEASES ;;
        /data/go-tour-ja-JP/releases/*) arguments[index]="$FAKE_JA_RELEASES/${arguments[index]#/data/go-tour-ja-JP/releases/}" ;;
        /data/go-tour-ja-JP/current) arguments[index]=$FAKE_JA_CURRENT ;;
        /data/go-tour-ja-JP/.deploy.lock) arguments[index]=$FAKE_JA_LOCK ;;
        /data/go-tour-de-DE/releases) arguments[index]=$FAKE_DE_RELEASES ;;
        /data/go-tour-de-DE/releases/*) arguments[index]="$FAKE_DE_RELEASES/${arguments[index]#/data/go-tour-de-DE/releases/}" ;;
        /data/go-tour-de-DE/current) arguments[index]=$FAKE_DE_CURRENT ;;
        /data/go-tour-de-DE/.deploy.lock) arguments[index]=$FAKE_DE_LOCK ;;
        /data/go-tour-fr-FR/releases) arguments[index]=$FAKE_FR_RELEASES ;;
        /data/go-tour-fr-FR/releases/*) arguments[index]="$FAKE_FR_RELEASES/${arguments[index]#/data/go-tour-fr-FR/releases/}" ;;
        /data/go-tour-fr-FR/current) arguments[index]=$FAKE_FR_CURRENT ;;
        /data/go-tour-fr-FR/.deploy.lock) arguments[index]=$FAKE_FR_LOCK ;;
    esac
done
if [[ ${FAKE_REQUIRE_NONEMPTY_ACTIVATION_ARGS:-0} == 1 && ${#arguments[@]} == 15 ]]; then
    for index in {3..14}; do
        [[ -n ${arguments[index]} ]] || {
            printf '[deploy-test:ssh] first deployment SSH positional argument regression: empty activation argument %s\n' "$index" >&2
            exit 97
        }
    done
fi
exec "${arguments[@]}"
SH
chmod 0755 -- "$fake_bin"/*

setup_remote() {
    local locale=$1 state=$2 root releases current lock old
    root=$fixture/remote-$locale
    releases=$root/releases; current=$root/current; lock=$root/.deploy.lock
    rm -rf -- "$root"; mkdir -p -- "$releases"
    if [[ $state == existing ]]; then
        old=$releases/old; mkdir -- "$old"; ln -s -- "$old" "$current"
    elif [[ $state == outside ]]; then
        mkdir -p -- "$fixture/outside"; ln -s -- "$fixture/outside" "$current"
    elif [[ $state == regular ]]; then
        printf 'not a link\n' >"$current"
    fi
    case $locale in
        zh-CN) export FAKE_ZH_RELEASES=$releases FAKE_ZH_CURRENT=$current FAKE_ZH_LOCK=$lock ;;
        ja-JP) export FAKE_JA_RELEASES=$releases FAKE_JA_CURRENT=$current FAKE_JA_LOCK=$lock ;;
        de-DE) export FAKE_DE_RELEASES=$releases FAKE_DE_CURRENT=$current FAKE_DE_LOCK=$lock ;;
        fr-FR) export FAKE_FR_RELEASES=$releases FAKE_FR_CURRENT=$current FAKE_FR_LOCK=$lock ;;
    esac
    TEST_RELEASES=$releases; TEST_CURRENT=$current; TEST_LOCK=$lock
}

prepare_and_activate() {
    local locale=$1 state=$2 suffix=$3 releases current lock prepared mode old staging final output rc
    setup_remote "$locale" "$state"
    releases=$TEST_RELEASES; current=$TEST_CURRENT; lock=$TEST_LOCK
    select_deployment_profile "$locale"
    [[ $state != absent ]] || EXPECTED_DEPLOYMENT_MODE=FIRST_DEPLOYMENT
    staging="$releases/.new.staging-$suffix"; final="$releases/new-$suffix"
    prepared=$(prepare_remote "$staging" "$final") || return 1
    IFS=$'\t' read -r mode old <<<"$prepared"
    mkdir -p -- "$staging"
    export FAKE_HEALTH_COUNTER=$fixture/health-$suffix
    rm -f -- "$FAKE_HEALTH_COUNTER"
    set +e
    output=$(activate_release "$mode" "$old" "$staging" "$final" "$suffix")
    rc=$?
    set -e
    printf '%s\t%s\t%s\t%s\t%s\n' "$rc" "$current" "$final" "$lock" "$output"
}

make_bundle() {
    local dir=$1 locale=$2
    mkdir -p -- "$dir/bin" "$dir/_content/tour/static/css"
    printf '#!/bin/sh\n' >"$dir/bin/tour"
    chmod 0755 -- "$dir/bin/tour"
    printf 'css\n' >"$dir/_content/tour/static/css/app.css"
    python3 - "$dir" "$locale" <<'PY'
import json, pathlib, sys
root, locale = pathlib.Path(sys.argv[1]), sys.argv[2]
common = {"locale": locale, "published_at": "2026-08-24T00:00:00Z", "upstream_commit": "a" * 40,
          "upstream_commit_time": "2026-08-24T00:00:00Z", "pages": 103, "articles": 7}
release = {**common, "schema_version": 2, "execution_transport": "http-playground-proxy",
           "execution_provider": "play.golang.org", "local_socket_enabled": False,
           "goos": "linux", "goarch": "amd64", "eligible_examples": 0, "translation_units": 103}
(root / "release.json").write_text(json.dumps(release), encoding="utf-8")
(root / "_content/tour/site-metadata.json").write_text(json.dumps(common), encoding="utf-8")
PY
    (cd -- "$dir" && find bin _content release.json -type f -print0 | sort -z | xargs -0 sha256sum >SHA256SUMS)
}

setup_already_current() {
    local locale=$1 suffix=$2

    TEST_BUNDLE=$fixture/go-tour-release-$suffix
    make_bundle "$TEST_BUNDLE" "$locale"
    setup_remote "$locale" absent
    TEST_FINAL=$TEST_RELEASES/$suffix
    cp -a -- "$TEST_BUNDLE" "$TEST_FINAL"
    ln -s -- "$TEST_FINAL" "$TEST_CURRENT"
    export FAKE_HEALTH_COUNTER=$fixture/health-resume-$suffix
    export FAKE_PUBLIC_COUNTER=$fixture/public-resume-$suffix
    export FAKE_RESTART_MARKER=$fixture/restart-resume-$suffix
    export FAKE_SERVICE_STATE=active FAKE_HTTP_FAIL_CALLS=0
    export FAKE_PUBLIC_HTTP_SEQUENCE=200 FAKE_PUBLIC_EXIT_SEQUENCE=0
    rm -f -- "$FAKE_HEALTH_COUNTER" "$FAKE_PUBLIC_COUNTER" "$FAKE_RESTART_MARKER"
}

# The misleading directory suffix must not influence profile selection.
fake_ja="$fixture/go-tour-release-20260824-zh-CN-fake"
make_bundle "$fake_ja" ja-JP
validate_local_release "$fake_ja" >/dev/null || fail 'valid ja-JP bundle was rejected'
[[ $RELEASE_LOCALE == ja-JP && $SERVICE == go-tour-ja-JP.service ]] || \
    fail 'directory name overrode release.json locale'

fake_unsupported="$fixture/go-tour-release-20260824-zh-CN-unsupported"
make_bundle "$fake_unsupported" zz-ZZ
(
    remote_called=0
    prepare_remote() { remote_called=1; return 1; }
    upload_release() { remote_called=1; return 1; }
    if main "$fake_unsupported" >/dev/null 2>&1; then
        exit 1
    fi
    (( remote_called == 0 ))
) || fail 'unsupported locale was accepted or reached a remote operation'

mismatched="$fixture/go-tour-release-20260824-ja-JP-mismatch"
make_bundle "$mismatched" ja-JP
python3 - "$mismatched/_content/tour/site-metadata.json" <<'PY'
import json, pathlib, sys
path = pathlib.Path(sys.argv[1])
data = json.loads(path.read_text(encoding="utf-8"))
data["locale"] = "zh-CN"
path.write_text(json.dumps(data), encoding="utf-8")
PY
(cd -- "$mismatched" && find bin _content -type f -print0 | sort -z | xargs -0 sha256sum >SHA256SUMS)
if validate_local_release "$mismatched" >/dev/null 2>&1; then
    fail 'bundle metadata locale inconsistent with the selected profile was accepted'
fi

export PATH="$fake_bin:$PATH"

# Existing deployments for both supported existing profiles still activate with the 3-success health rule.
for locale in zh-CN ja-JP; do
    export FAKE_HTTP_FAIL_CALLS=0
    IFS=$'\t' read -r rc current final lock _ <<<"$(prepare_and_activate "$locale" existing "$locale-existing")"
    [[ $rc == 0 && -L $current && $(readlink -f -- "$current") == "$final" && ! -e $lock ]] || \
        fail "$locale existing deployment did not activate correctly"
    [[ $(<"$fixture/health-$locale-existing") -ge 3 ]] || fail "$locale health acceptance did not require three successes"
done

# First deployment permits an absent current link, creates it atomically, and uses the same health acceptance.
# The fake SSH rejects any empty activation argument, modelling OpenSSH command flattening.
export FAKE_HTTP_FAIL_CALLS=0 FAKE_REQUIRE_NONEMPTY_ACTIVATION_ARGS=1
IFS=$'\t' read -r rc current final lock _ <<<"$(prepare_and_activate de-DE absent de-first-success)"
[[ $rc == 0 && -L $current && $(readlink -f -- "$current") == "$final" && ! -e $lock ]] || \
    fail 'first deployment SSH positional argument regression: activation did not create current or clean the lock'
[[ $(<"$fixture/health-de-first-success") -ge 3 ]] || fail 'first deployment health acceptance did not require three successes'
unset FAKE_REQUIRE_NONEMPTY_ACTIVATION_ARGS

# The new fr-FR profile follows the same first-deployment activation and health gate.
export FAKE_HTTP_FAIL_CALLS=0
IFS=$'\t' read -r rc current final lock _ <<<"$(prepare_and_activate fr-FR absent fr-first-success)"
[[ $rc == 0 && -L $current && $(readlink -f -- "$current") == "$final" && ! -e $lock ]] || \
    fail 'fr-FR first deployment did not activate with its exact profile'
[[ $(<"$fixture/health-fr-first-success") -ge 3 ]] || fail 'fr-FR health acceptance did not require three successes'

# Existing deployment failure retains the existing rollback behavior.
export FAKE_HTTP_FAIL_CALLS=12
IFS=$'\t' read -r rc current final lock _ <<<"$(prepare_and_activate zh-CN existing zh-rollback)"
[[ $rc == 20 && $(readlink -f -- "$current") == "${current%/current}/releases/old" && ! -e $lock ]] || \
    fail 'existing deployment rollback did not restore old release'

# First deployment failure never invents a rollback target and preserves current plus deployment lock for inspection.
export FAKE_HTTP_FAIL_CALLS=99
IFS=$'\t' read -r rc current final lock output <<<"$(prepare_and_activate de-DE absent de-first-failure)"
[[ $rc == 22 && -L $current && $(readlink -f -- "$current") == "$final" && -d $lock ]] || \
    fail 'first deployment health failure changed current or removed evidence'

# A present non-symlink current and a symlink outside releases both fail before a lock is created.
for state in regular outside; do
    setup_remote de-DE "$state"
    select_deployment_profile de-DE
    if prepare_remote "$TEST_RELEASES/.staging-$state" "$TEST_RELEASES/new-$state" >/dev/null 2>&1; then
        fail "invalid current state $state was accepted"
    fi
    [[ ! -e $TEST_LOCK ]] || fail "invalid current state $state created a lock"
done

# A live locale never falls back to FIRST_DEPLOYMENT when current is missing.
setup_remote fr-FR absent
select_deployment_profile fr-FR
if prepare_remote "$TEST_RELEASES/.staging-live-absent" "$TEST_RELEASES/new-live-absent" >/dev/null 2>&1; then
    fail 'live locale with missing current was accepted as FIRST_DEPLOYMENT'
fi
[[ ! -e $TEST_LOCK ]] || fail 'live locale lifecycle mismatch created a lock'

# Public acceptance retries only the formally transient transport/status set.
PUBLIC_URL=https://example.test/
PUBLIC_ACCEPTANCE_HINT='test hint'
for transient_exit in 6 7 16 28 35; do
    export FAKE_PUBLIC_COUNTER=$fixture/public-exit-$transient_exit
    export FAKE_PUBLIC_HTTP_SEQUENCE=000,200 FAKE_PUBLIC_EXIT_SEQUENCE=$transient_exit,0
    rm -f -- "$FAKE_PUBLIC_COUNTER"
    check_public >/dev/null 2>&1 || fail "transient curl exit $transient_exit was not retried"
    [[ $(<"$FAKE_PUBLIC_COUNTER") == 2 ]] || fail "transient curl exit $transient_exit retry count"
done
for transient_status in 522 525; do
    export FAKE_PUBLIC_COUNTER=$fixture/public-http-$transient_status
    export FAKE_PUBLIC_HTTP_SEQUENCE=$transient_status,200 FAKE_PUBLIC_EXIT_SEQUENCE=0,0
    rm -f -- "$FAKE_PUBLIC_COUNTER"
    check_public >/dev/null 2>&1 || fail "transient HTTP $transient_status was not retried"
    [[ $(<"$FAKE_PUBLIC_COUNTER") == 2 ]] || fail "transient HTTP $transient_status retry count"
done

export FAKE_PUBLIC_COUNTER=$fixture/public-persistent
export FAKE_PUBLIC_HTTP_SEQUENCE=000 FAKE_PUBLIC_EXIT_SEQUENCE=28
rm -f -- "$FAKE_PUBLIC_COUNTER"
if persistent_output=$(check_public 2>&1); then
    fail 'persistent transient public failure was accepted'
fi
[[ $(<"$FAKE_PUBLIC_COUNTER") == 3 ]] || fail 'public retry exceeded or missed the three-attempt bound'
[[ $persistent_output == *'curl exit 28; HTTP 000'* ]] || fail 'public failure lost curl exit/status evidence'

export FAKE_PUBLIC_COUNTER=$fixture/public-semantic
export FAKE_PUBLIC_HTTP_SEQUENCE=503,200 FAKE_PUBLIC_EXIT_SEQUENCE=0,0
rm -f -- "$FAKE_PUBLIC_COUNTER"
if check_public >/dev/null 2>&1; then
    fail 'nontransient HTTP 503 was retried and accepted'
fi
[[ $(<"$FAKE_PUBLIC_COUNTER") == 1 ]] || fail 'nontransient HTTP failure did not fail immediately'

# An exactly identical, already-current live release takes the read-only resume path.
setup_already_current de-DE 20260911-de-DE-resume-pass
mutation_marker=$fixture/mutation-resume-pass
if ! resume_output=$(
    upload_release() { : >"$mutation_marker"; return 97; }
    validate_remote_release() { : >"$mutation_marker"; return 97; }
    activate_release() { : >"$mutation_marker"; return 97; }
    cleanup_before_activation() { : >"$mutation_marker"; return 97; }
    main "$TEST_BUNDLE"
  2>&1); then
    printf '%s\n' "$resume_output" >&2
    fail 'identical already-current release did not resume'
fi
[[ $resume_output == *'deployment already current / RESUME'* ]] || fail 'resume result was not explicit'
[[ $(readlink -f -- "$TEST_CURRENT") == "$TEST_FINAL" ]] || fail 'resume changed current'
[[ ! -e $mutation_marker && ! -e $FAKE_RESTART_MARKER ]] || fail 'resume invoked a production mutation'
[[ $(<"$FAKE_HEALTH_COUNTER") -ge 3 && $(<"$FAKE_PUBLIC_COUNTER") == 1 ]] || \
    fail 'resume skipped source health or public acceptance'
if find "$TEST_RELEASES" -mindepth 1 -maxdepth 1 -name '.*.staging-*' -print -quit | grep -q .; then
    fail 'resume created staging'
fi

assert_resume_fails_without_mutation() {
    local label=$1 output mutation rc
    mutation=$fixture/mutation-$label

    if output=$(
      (
        upload_release() { : >"$mutation"; return 97; }
        validate_remote_release() { : >"$mutation"; return 97; }
        activate_release() { : >"$mutation"; return 97; }
        cleanup_before_activation() { : >"$mutation"; return 97; }
        if main "$TEST_BUNDLE"; then main_rc=0; else main_rc=$?; fi
        trap - EXIT
        exit "$main_rc"
      ) 2>&1
    ); then
        rc=0
    else
        rc=$?
    fi
    (( rc != 0 )) || fail "$label unsafe resume was accepted"
    [[ ! -e $mutation && ! -e $FAKE_RESTART_MARKER ]] || fail "$label failure invoked a production mutation"
    if find "$TEST_RELEASES" -mindepth 1 -maxdepth 1 -name '.*.staging-*' -print -quit | grep -q .; then
        fail "$label failure created staging"
    fi
}

setup_already_current de-DE 20260911-de-DE-resume-checksum
printf 'remote drift\n' >"$TEST_FINAL/_content/tour/static/css/app.css"
assert_resume_fails_without_mutation checksum-mismatch

setup_already_current de-DE 20260911-de-DE-resume-lock
mkdir -- "$TEST_LOCK"
assert_resume_fails_without_mutation lock-present

setup_already_current de-DE 20260911-de-DE-resume-service
export FAKE_SERVICE_STATE=inactive
assert_resume_fails_without_mutation service-inactive

setup_already_current de-DE 20260911-de-DE-resume-health
export FAKE_HTTP_FAIL_CALLS=99
assert_resume_fails_without_mutation localhost-non-200

# FIRST_DEPLOYMENT is never eligible for the already-current live-maintenance recovery.
setup_already_current de-DE 20260911-de-DE-first-cannot-resume
select_deployment_profile de-DE
EXPECTED_DEPLOYMENT_MODE=FIRST_DEPLOYMENT
if prepare_remote "$TEST_RELEASES/.first-resume.staging" "$TEST_FINAL" >/dev/null 2>&1; then
    fail 'FIRST_DEPLOYMENT accepted already-current recovery'
fi
[[ ! -e $TEST_LOCK ]] || fail 'FIRST_DEPLOYMENT lifecycle mismatch created a lock'

printf '[deploy-test] PASS\n'
