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

if select_deployment_profile fr-FR 2>/dev/null; then
    fail 'unsupported locale was accepted'
fi

fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT

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
    restart) [[ ${FAKE_RESTART_FAIL:-0} == 0 ]] && exit 0 || exit 1 ;;
    is-active) printf 'active\n'; exit 0 ;;
    status|show) exit 0 ;;
esac
exit 1
SH
cat >"$fake_bin/curl" <<'SH'
#!/usr/bin/env bash
count_file=${FAKE_HEALTH_COUNTER:?}
count=0; [[ -f $count_file ]] && count=$(<"$count_file")
count=$((count + 1)); printf '%s' "$count" >"$count_file"
if (( count <= ${FAKE_HTTP_FAIL_CALLS:-0} )); then printf '500'; else printf '200'; fi
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
        /data/go-tour/current) arguments[index]=$FAKE_ZH_CURRENT ;;
        /data/go-tour/.deploy.lock) arguments[index]=$FAKE_ZH_LOCK ;;
        /data/go-tour-ja-JP/releases) arguments[index]=$FAKE_JA_RELEASES ;;
        /data/go-tour-ja-JP/current) arguments[index]=$FAKE_JA_CURRENT ;;
        /data/go-tour-ja-JP/.deploy.lock) arguments[index]=$FAKE_JA_LOCK ;;
        /data/go-tour-de-DE/releases) arguments[index]=$FAKE_DE_RELEASES ;;
        /data/go-tour-de-DE/current) arguments[index]=$FAKE_DE_CURRENT ;;
        /data/go-tour-de-DE/.deploy.lock) arguments[index]=$FAKE_DE_LOCK ;;
    esac
done
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
    esac
    TEST_RELEASES=$releases; TEST_CURRENT=$current; TEST_LOCK=$lock
}

prepare_and_activate() {
    local locale=$1 state=$2 suffix=$3 releases current lock prepared mode old staging final output rc
    setup_remote "$locale" "$state"
    releases=$TEST_RELEASES; current=$TEST_CURRENT; lock=$TEST_LOCK
    select_deployment_profile "$locale"
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
    (cd -- "$dir" && find bin _content -type f -print0 | sort -z | xargs -0 sha256sum >SHA256SUMS)
}

# The misleading directory suffix must not influence profile selection.
fake_ja="$fixture/go-tour-release-20260824-zh-CN-fake"
make_bundle "$fake_ja" ja-JP
validate_local_release "$fake_ja" >/dev/null || fail 'valid ja-JP bundle was rejected'
[[ $RELEASE_LOCALE == ja-JP && $SERVICE == go-tour-ja-JP.service ]] || \
    fail 'directory name overrode release.json locale'

fake_unsupported="$fixture/go-tour-release-20260824-zh-CN-unsupported"
make_bundle "$fake_unsupported" fr-FR
(
    remote_called=0
    prepare_remote() { remote_called=1; return 1; }
    upload_release() { remote_called=1; return 1; }
    if main "$fake_unsupported" >/dev/null 2>&1; then
        exit 1
    fi
    (( remote_called == 0 )) && [[ $RELEASE_LOCALE == fr-FR ]]
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
export FAKE_HTTP_FAIL_CALLS=0
IFS=$'\t' read -r rc current final lock _ <<<"$(prepare_and_activate de-DE absent de-first-success)"
[[ $rc == 0 && -L $current && $(readlink -f -- "$current") == "$final" && ! -e $lock ]] || \
    fail 'first deployment did not create current or clean the lock'
[[ $(<"$fixture/health-de-first-success") -ge 3 ]] || fail 'first deployment health acceptance did not require three successes'

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

printf '[deploy-test] PASS\n'
