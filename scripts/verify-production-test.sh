#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
verify_script=$script_dir/verify-production.sh
fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT

fail() {
    printf '[verify-production-test] FAIL: %s\n' "$*" >&2
    exit 1
}

verify_source=$(<"$verify_script")
[[ $verify_source == *'VERIFY_PRODUCTION_NETWORK_SSH'* \
    && $verify_source == *'ControlMaster=yes'* \
    && $verify_source == *'--socks5-hostname'* \
    && $verify_source == *'cleanup_network_ssh'* ]] || \
    fail 'zgocloud invocation-scoped public network runner is incomplete'

assert_contains() {
    [[ $1 == *"$2"* ]] || fail "output does not contain: $2"
}

# Keep the verification profile independently pinned to the deployment profile values.
(
    # shellcheck source=verify-production.sh
    source "$verify_script"
    select_production_profile zh-CN
    [[ $RELEASES_DIR == /data/go-tour/releases && $CURRENT_LINK == /data/go-tour/current \
        && $DEPLOY_LOCK == /data/go-tour/.deploy.lock && $SERVICE == go-tour.service \
        && $LOOPBACK_ORIGIN == http://127.0.0.1:3999 \
        && $PUBLIC_ORIGIN == https://go-dev.shuijingwanwq.com \
        && $CACHE_HEADER == EO-Cache-Status ]]
    select_production_profile ja-JP
    [[ $RELEASES_DIR == /data/go-tour-ja-JP/releases && $CURRENT_LINK == /data/go-tour-ja-JP/current \
        && $DEPLOY_LOCK == /data/go-tour-ja-JP/.deploy.lock && $SERVICE == go-tour-ja-JP.service \
        && $LOOPBACK_ORIGIN == http://127.0.0.1:4000 \
        && $PUBLIC_ORIGIN == https://ja-go-dev.shuijingwanwq.com \
        && $CACHE_HEADER == CF-Cache-Status ]]
    select_production_profile de-DE
    [[ $RELEASES_DIR == /data/go-tour-de-DE/releases && $CURRENT_LINK == /data/go-tour-de-DE/current \
        && $DEPLOY_LOCK == /data/go-tour-de-DE/.deploy.lock && $SERVICE == go-tour-de-DE.service \
        && $LOOPBACK_ORIGIN == http://127.0.0.1:4001 \
        && $PUBLIC_ORIGIN == https://de-go-dev.shuijingwanwq.com \
        && $CACHE_HEADER == CF-Cache-Status ]]
    select_production_profile fr-FR
    [[ $RELEASES_DIR == /data/go-tour-fr-FR/releases && $CURRENT_LINK == /data/go-tour-fr-FR/current \
        && $DEPLOY_LOCK == /data/go-tour-fr-FR/.deploy.lock && $SERVICE == go-tour-fr-FR.service \
        && $LOOPBACK_ORIGIN == http://127.0.0.1:4002 \
        && $PUBLIC_ORIGIN == https://fr-go-dev.shuijingwanwq.com \
        && $PRODUCTION_HOST == fr-go-dev.shuijingwanwq.com \
        && $CACHE_HEADER == CF-Cache-Status ]]
) || fail 'verification profiles differ from the formal production values'

fake_bin=$fixture/bin
mkdir -p -- "$fake_bin"

cat >"$fake_bin/systemctl" <<'SH'
#!/usr/bin/env bash
if [[ ${1:-} == is-active ]]; then
    printf '%s\n' "${FAKE_SERVICE_STATE:-active}"
    [[ ${FAKE_SERVICE_STATE:-active} == active ]]
    exit
fi
exit 1
SH

cat >"$fake_bin/ssh" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'called\n' >"${FAKE_SSH_MARKER:?}"
while [[ ${1:-} == -o ]]; do shift 2; done
[[ ${1:-} == aliyun ]] || exit 90
shift
arguments=("$@")
for index in "${!arguments[@]}"; do
    value=${arguments[index]}
    case $value in
        /data/go-tour-de-DE/releases)
            arguments[index]=$FAKE_REMOTE_RELEASES
            ;;
        /data/go-tour-de-DE/releases/*)
            arguments[index]="$FAKE_REMOTE_RELEASES/${value#/data/go-tour-de-DE/releases/}"
            ;;
        /data/go-tour-de-DE/current)
            arguments[index]=$FAKE_REMOTE_CURRENT
            ;;
        /data/go-tour-de-DE/.deploy.lock)
            arguments[index]=$FAKE_REMOTE_LOCK
            ;;
        /data/go-tour-fr-FR/releases)
            arguments[index]=$FAKE_REMOTE_RELEASES
            ;;
        /data/go-tour-fr-FR/releases/*)
            arguments[index]="$FAKE_REMOTE_RELEASES/${value#/data/go-tour-fr-FR/releases/}"
            ;;
        /data/go-tour-fr-FR/current)
            arguments[index]=$FAKE_REMOTE_CURRENT
            ;;
        /data/go-tour-fr-FR/.deploy.lock)
            arguments[index]=$FAKE_REMOTE_LOCK
            ;;
    esac
done
exec "${arguments[@]}"
SH

cat >"$fake_bin/curl" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail

headers=''
body=/dev/null
write_out=''
url=''
upgrade=0
while (( $# )); do
    case $1 in
        -sS) shift ;;
        --connect-timeout|--max-time|-D|-o|-w|-H)
            option=$1
            value=$2
            shift 2
            case $option in
                -D) headers=$value ;;
                -o) body=$value ;;
                -w) write_out=$value ;;
                -H) [[ $value == 'Upgrade: websocket' ]] && upgrade=1 ;;
            esac
            ;;
        *) url=$1; shift ;;
    esac
done

status=200
cache=''
if [[ $url == http://127.0.0.1:* ]]; then
    path=/${url#*://*/}
    [[ $url == */ ]] && path=/
    [[ ${FAKE_SOURCE_FAIL_PATH:-} == "$path" ]] && status=500
else
    path=${url#"$FAKE_PUBLIC_ORIGIN"}
    [[ -n $path ]] || path=/
    if [[ ${FAKE_PUBLIC_FAIL_PATH:-} == "$path" ]]; then
        status=503
    elif [[ $path == /socket ]]; then
        if (( upgrade )); then status=${FAKE_SOCKET_UPGRADE_STATUS:-404}
        else status=${FAKE_SOCKET_NORMAL_STATUS:-404}
        fi
    elif [[ ${FAKE_SITEMAP_FAIL_URL:-} == "$url" ]]; then
        status=500
    fi

    if [[ $status == 200 && ( $path == / || $path == /tour/welcome/1 ) \
        && -n ${FAKE_CACHE_HTTP_STATUS:-} ]]; then
        status=$FAKE_CACHE_HTTP_STATUS
    fi
    if [[ $status == 200 && ( $path == / || $path == /tour/welcome/1 ) ]]; then
        key=root
        [[ $path == /tour/welcome/1 ]] && key=welcome
        counter="$FAKE_STATE_DIR/cache-$key"
        count=0
        [[ -f $counter ]] && count=$(<"$counter")
        count=$((count + 1))
        printf '%s' "$count" >"$counter"
        IFS=',' read -r -a cache_sequence <<<"${FAKE_CACHE_SEQUENCE:-MISS,HIT,HIT}"
        sequence_index=$((count - 1))
        if (( sequence_index >= ${#cache_sequence[@]} )); then
            sequence_index=$((${#cache_sequence[@]} - 1))
        fi
        cache=${cache_sequence[sequence_index]}
        [[ $cache == MISSING ]] && cache=''
    fi
fi

if [[ -n $headers ]]; then
    printf 'HTTP/2 %s Test\r\n' "$status" >"$headers"
    if [[ -n $cache ]]; then
        if [[ $FAKE_PUBLIC_ORIGIN == https://go-dev.shuijingwanwq.com ]]; then
            printf 'EO-Cache-Status: %s\r\n' "$cache" >>"$headers"
        else
            printf 'CF-Cache-Status: %s\r\n' "$cache" >>"$headers"
        fi
    fi
    printf '\r\n' >>"$headers"
fi

if [[ $status == 200 && $body != /dev/null ]]; then
    case $path in
        /)
            lang=${FAKE_HTML_LANG:-de-DE}
            canonical="$FAKE_PUBLIC_ORIGIN/"
            [[ ${FAKE_HTML_MODE:-} == LANG ]] && lang=en
            [[ ${FAKE_HTML_MODE:-} == CANONICAL ]] && canonical=https://wrong.example/
            printf '<!doctype html><html lang="%s"><head><link rel="canonical" href="%s"></head></html>\n' \
                "$lang" "$canonical" >"$body"
            ;;
        /tour/welcome/1)
            lang=${FAKE_HTML_LANG:-de-DE}
            canonical="$FAKE_PUBLIC_ORIGIN/tour/welcome/1"
            [[ ${FAKE_HTML_MODE:-} == LANG ]] && lang=en
            [[ ${FAKE_HTML_MODE:-} == CANONICAL ]] && canonical=https://wrong.example/tour/welcome/1
            printf '<!doctype html><html lang="%s"><head><link rel="canonical" href="%s"></head></html>\n' \
                "$lang" "$canonical" >"$body"
            ;;
        /sitemap.xml)
            fake_pages=102
            [[ ${FAKE_SITEMAP_MODE:-} == COUNT ]] && fake_pages=101
            {
                printf '<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n'
                printf '<url><loc>%s/</loc></url>\n' "$FAKE_PUBLIC_ORIGIN"
                printf '<url><loc>%s/tour/list</loc></url>\n' "$FAKE_PUBLIC_ORIGIN"
                printf '<url><loc>%s/tour/welcome/1</loc></url>\n' "$FAKE_PUBLIC_ORIGIN"
                for ((index = 1; index <= fake_pages; index++)); do
                    host=$FAKE_PUBLIC_ORIGIN
                    [[ ${FAKE_SITEMAP_MODE:-} == HOST && $index == 1 ]] && host=https://wrong.example
                    printf '<url><loc>%s/tour/fake/%d</loc></url>\n' "$host" "$index"
                done
                printf '</urlset>\n'
            } >"$body"
            ;;
        *) printf 'body\n' >"$body" ;;
    esac
fi

[[ -z $write_out ]] || printf '%s' "$status"
SH

chmod 0755 -- "$fake_bin"/*

# zh-CN uses the documented EdgeOne header rather than the Cloudflare header.
(
    # shellcheck source=verify-production.sh
    source "$verify_script"
    TEMP_DIR=$fixture/edgeone-state
    mkdir -p -- "$TEMP_DIR"
    PUBLIC_ORIGIN=https://go-dev.shuijingwanwq.com
    CACHE_HEADER=EO-Cache-Status
    export FAKE_PUBLIC_ORIGIN=$PUBLIC_ORIGIN FAKE_STATE_DIR=$TEMP_DIR
    PATH="$fake_bin:$PATH"
    verify_cache_path / CACHE_HOME_RESULT
    [[ $CACHE_HOME_RESULT == 'MISS -> HIT -> HIT PASS' ]]
) || fail 'zh-CN EdgeOne cache status observation failed'

release_dir=$fixture/go-tour-release-20260829-de-DE-8937fdc
mkdir -p -- "$release_dir"
printf '{"locale":"de-DE"}\n' >"$release_dir/release.json"
de_release_dir=$release_dir
fr_release_dir=$fixture/go-tour-release-20260830-fr-FR-bd8df0a
mkdir -p -- "$fr_release_dir"
printf '{"locale":"fr-FR"}\n' >"$fr_release_dir/release.json"

setup_case() {
    local current_mode=${1:-expected} locale=${2:-de-DE} release_name
    if [[ $locale == fr-FR ]]; then
        release_dir=$fr_release_dir
        export FAKE_PUBLIC_ORIGIN=https://fr-go-dev.shuijingwanwq.com FAKE_HTML_LANG=fr-FR
    else
        release_dir=$de_release_dir
        export FAKE_PUBLIC_ORIGIN=https://de-go-dev.shuijingwanwq.com FAKE_HTML_LANG=de-DE
    fi
    rm -rf -- "$fixture/remote" "$fixture/state"
    mkdir -p -- "$fixture/remote/releases" "$fixture/state"
    release_name=${release_dir##*/}; release_name=${release_name#go-tour-release-}
    mkdir -p -- "$fixture/remote/releases/$release_name"
    if [[ $current_mode == expected ]]; then
        ln -s -- "$fixture/remote/releases/$release_name" "$fixture/remote/current"
    else
        mkdir -p -- "$fixture/remote/releases/other"
        ln -s -- "$fixture/remote/releases/other" "$fixture/remote/current"
    fi
    export FAKE_REMOTE_RELEASES=$fixture/remote/releases
    export FAKE_REMOTE_CURRENT=$fixture/remote/current
    export FAKE_REMOTE_LOCK=$fixture/remote/.deploy.lock
    export FAKE_STATE_DIR=$fixture/state
    export FAKE_SSH_MARKER=$fixture/ssh-called
    rm -f -- "$FAKE_SSH_MARKER"
    unset FAKE_SOURCE_FAIL_PATH FAKE_PUBLIC_FAIL_PATH FAKE_HTML_MODE FAKE_SITEMAP_MODE
    unset FAKE_SITEMAP_FAIL_URL FAKE_SOCKET_NORMAL_STATUS FAKE_SOCKET_UPGRADE_STATUS
    unset FAKE_CACHE_SEQUENCE FAKE_CACHE_HTTP_STATUS FAKE_SERVICE_STATE
}

run_verify() {
    env PATH="$fake_bin:$PATH" "$verify_script" "$release_dir" 2>&1
}

expect_failure() {
    local description=$1 expected=$2 output status
    set +e
    output=$(run_verify)
    status=$?
    set -e
    (( status != 0 )) || fail "$description was accepted"
    assert_contains "$output" "$expected"
}

setup_case
output=$(run_verify) || fail 'de-DE complete success failed'
assert_contains "$output" '[verify-production] source routes: 7/7 PASS'
assert_contains "$output" '[verify-production] public routes: 7/7 PASS'
assert_contains "$output" '[verify-production] sitemap: 105/105 PASS'
assert_contains "$output" '[verify-production] CDN /: MISS -> HIT -> HIT PASS'
assert_contains "$output" 'PRODUCTION MACHINE ACCEPTANCE: PASS'

setup_case expected fr-FR
output=$(run_verify) || fail 'fr-FR complete success failed'
assert_contains "$output" '[verify-production] release identity: PASS (fr-FR -> /data/go-tour-fr-FR/releases/'
assert_contains "$output" '[verify-production] CDN /: MISS -> HIT -> HIT PASS'
assert_contains "$output" 'PRODUCTION MACHINE ACCEPTANCE: PASS'

setup_case other
expect_failure 'mismatched current release' 'stage=remote identity check=current'

setup_case
export FAKE_SOURCE_FAIL_PATH=/tour/list
expect_failure 'source route failure' 'stage=source routes'

setup_case
export FAKE_PUBLIC_FAIL_PATH=/tour/list
expect_failure 'public route failure' 'stage=public routes'

setup_case
export FAKE_HTML_MODE=LANG
expect_failure 'HTML lang failure' 'stage=html identity'

setup_case
export FAKE_HTML_MODE=CANONICAL
expect_failure 'HTML canonical failure' 'canonical'

setup_case
export FAKE_SITEMAP_MODE=COUNT
expect_failure 'sitemap count failure' 'check=URL-count'

setup_case
export FAKE_SITEMAP_MODE=HOST
expect_failure 'sitemap hostname failure' 'check=hostname'

setup_case
export FAKE_SITEMAP_FAIL_URL=$FAKE_PUBLIC_ORIGIN/tour/fake/7
expect_failure 'sitemap URL HTTP failure' 'check=https://de-go-dev.shuijingwanwq.com/tour/fake/7'

setup_case
export FAKE_SOCKET_NORMAL_STATUS=200
expect_failure 'ordinary socket boundary failure' 'check=GET /socket'

setup_case
export FAKE_SOCKET_UPGRADE_STATUS=200
expect_failure 'upgrade socket boundary failure' 'check=Upgrade /socket'

for cache_case in \
    'HIT,HIT,HIT|HIT -> HIT -> HIT PASS' \
    'MISS,MISS,MISS|MISS -> MISS -> MISS PASS (cache not warm yet)' \
    'MISS,MISS,HIT|MISS -> MISS -> HIT PASS' \
    'EXPIRED,REVALIDATED,UPDATING|EXPIRED -> REVALIDATED -> UPDATING PASS' \
    'STALE,HIT,MISS|STALE -> HIT -> MISS PASS'
do
    setup_case
    export FAKE_CACHE_SEQUENCE=${cache_case%%|*}
    output=$(run_verify) || fail "allowed cache observation failed: $FAKE_CACHE_SEQUENCE"
    assert_contains "$output" "[verify-production] CDN /: ${cache_case#*|}"
done

for rejected_cache_status in BYPASS DYNAMIC UNKNOWN_STATUS; do
    setup_case
    export FAKE_CACHE_SEQUENCE="$rejected_cache_status,HIT,HIT"
    expect_failure "$rejected_cache_status cache status" "actual=$rejected_cache_status"
done

setup_case
export FAKE_CACHE_SEQUENCE='MISSING,HIT,HIT'
expect_failure 'missing cache status header' 'actual=missing'

setup_case
export FAKE_CACHE_HTTP_STATUS=503
expect_failure 'cache observation HTTP failure' 'expected=HTTP 200 actual=HTTP 503'

unsupported=$fixture/go-tour-release-20260829-it-IT-unsupported
mkdir -p -- "$unsupported"
printf '{"locale":"it-IT"}\n' >"$unsupported/release.json"
release_dir=$unsupported
rm -f -- "$FAKE_SSH_MARKER"
expect_failure 'unsupported locale' 'locale with one valid formal production identity'
[[ ! -e $FAKE_SSH_MARKER ]] || fail 'unsupported locale reached SSH'

printf '[verify-production-test] PASS\n'
