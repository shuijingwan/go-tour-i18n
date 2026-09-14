#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

readonly -a SSH_OPTIONS=(
    -o BatchMode=yes
    -o ConnectTimeout=10
    -o ServerAliveInterval=5
    -o ServerAliveCountMax=3
    -o ConnectionAttempts=3
)
readonly CURL_CONNECT_TIMEOUT=5
readonly CURL_MAX_TIME=15
readonly CURL_RETRY_ATTEMPTS=5
readonly CURL_RETRY_MAX_BACKOFF=4
readonly EXPECTED_SITEMAP_URLS=105
readonly -a ACCEPTANCE_PATHS=(
    '/'
    '/tour/'
    '/tour/list'
    '/tour/welcome/1'
    '/tour/static/js/app.js'
    '/robots.txt'
    '/sitemap.xml'
)

RELEASE_LOCALE=''
RELEASES_DIR=''
CURRENT_LINK=''
DEPLOY_LOCK=''
SERVICE=''
LOOPBACK_ORIGIN=''
PUBLIC_ORIGIN=''
PRODUCTION_HOST=''
CACHE_HEADER=''
CACHE_HOME_RESULT=''
CACHE_WELCOME_RESULT=''
TEMP_DIR=''
SSH_HOST=''
CURL_NETWORK_OPTIONS=()
HTTP_REQUEST_EXIT=0
HTTP_REQUEST_STATUS=''
HTTP_REQUEST_ATTEMPTS=0
HTTP_REQUEST_STDERR=''
NETWORK_SSH_HOST=''

if [[ ${1:-} != --public ]]; then
    script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
    # shellcheck source=production-identity.sh
    source "$script_dir/production-identity.sh"
    unset script_dir
fi

error() {
    printf '[verify-production] ERROR: %s\n' "$*" >&2
}

fail_check() {
    local stage=$1 check=$2 expected=$3 actual=$4
    error "locale=${RELEASE_LOCALE:-unknown} stage=$stage check=$check expected=$expected actual=$actual"
    return 1
}

usage() {
    printf 'usage: %s <release-dir>\n' "${0##*/}" >&2
}

select_production_profile() {
    local locale=$1

    if ! load_production_identity_locale "$locale"; then
        fail_check 'release identity' locale 'locale with one valid formal production identity' "$locale"
        return 1
    fi
    SSH_HOST=$PRODUCTION_ORIGIN_SSH_ALIAS
    RELEASES_DIR=$PRODUCTION_RELEASES_ROOT
    CURRENT_LINK=$PRODUCTION_CURRENT
    DEPLOY_LOCK=$PRODUCTION_DEPLOYMENT_LOCK
    SERVICE=$PRODUCTION_SYSTEMD_SERVICE
    LOOPBACK_ORIGIN=${PRODUCTION_LOCALHOST_HEALTH_URL%/}
    PUBLIC_ORIGIN=${PRODUCTION_PUBLIC_URL%/}
    PRODUCTION_HOST=$PRODUCTION_HOSTNAME
    CACHE_HEADER=$PRODUCTION_CACHE_HEADER
}

validate_local_tools() {
    local command_name
    for command_name in awk basename cat curl mktemp python3 readlink ssh tr; do
        command -v "$command_name" >/dev/null || {
            fail_check 'release identity' tool "$command_name available" missing
            return 1
        }
    done
}

read_release_locale() {
    local release_dir=$1
    python3 - "$release_dir/release.json" <<'PY'
import json
import sys

try:
    with open(sys.argv[1], encoding="utf-8") as source:
        release = json.load(source)
except (OSError, json.JSONDecodeError) as exc:
    raise SystemExit(f"release.json error: {exc}")
locale = release.get("locale") if type(release) is dict else None
if type(locale) is not str or not locale:
    raise SystemExit(f"release.json locale must be a non-empty string, got {locale!r}")
print(locale)
PY
}

remote_release_name() {
    local release_dir=$1 local_name remote_name
    local_name=$(basename -- "$release_dir")
    if [[ $local_name != go-tour-release-* ]]; then
        fail_check 'release identity' basename 'go-tour-release-<safe-remote-name>' "$local_name"
        return 1
    fi
    remote_name=${local_name#go-tour-release-}
    if [[ -z $remote_name || ! $remote_name =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
        fail_check 'release identity' remote-release-name 'safe [A-Za-z0-9._-] name' "$remote_name"
        return 1
    fi
    printf '%s\n' "$remote_name"
}

verify_remote_and_source() {
    local expected_remote=$1
    ssh "${SSH_OPTIONS[@]}" "$SSH_HOST" bash -s -- \
        "$RELEASE_LOCALE" "$RELEASES_DIR" "$CURRENT_LINK" "$DEPLOY_LOCK" "$SERVICE" \
        "$LOOPBACK_ORIGIN" "$expected_remote" "${ACCEPTANCE_PATHS[@]}" <<'REMOTE'
set -Eeuo pipefail
IFS=$'\n\t'

locale=$1
releases_dir=$2
current_link=$3
deploy_lock=$4
service=$5
loopback_origin=$6
expected_remote=$7
shift 7
paths=("$@")

fail_check() {
    printf '[verify-production] ERROR: locale=%s stage=%s check=%s expected=%s actual=%s\n' \
        "$locale" "$1" "$2" "$3" "$4" >&2
    exit 1
}

for command_name in curl readlink systemctl; do
    command -v "$command_name" >/dev/null || fail_check 'remote identity' tool "$command_name available" missing
done

case $expected_remote in
    "$releases_dir"/*) ;;
    *) fail_check 'remote identity' expected-release-boundary "$releases_dir/<release>" "$expected_remote" ;;
esac
[[ -d $releases_dir ]] || fail_check 'remote identity' releases-root 'existing directory' missing
[[ -d $expected_remote ]] || fail_check 'remote identity' expected-release 'existing directory' missing
[[ -L $current_link ]] || fail_check 'remote identity' current 'symlink' "$(if [[ -e $current_link ]]; then printf non-symlink; else printf missing; fi)"
actual_current=$(readlink -f -- "$current_link" 2>/dev/null || true)
[[ $actual_current == "$expected_remote" ]] || fail_check 'remote identity' current "$expected_remote" "${actual_current:-unresolved}"
[[ ! -e $deploy_lock && ! -L $deploy_lock ]] || fail_check 'remote identity' deployment-lock absent present
service_state=$(systemctl is-active "$service" 2>/dev/null || true)
[[ $service_state == active ]] || fail_check 'remote identity' service active "${service_state:-unknown}"

for path in "${paths[@]}"; do
    code=$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time 5 \
        "$loopback_origin$path" 2>/dev/null || true)
    [[ $code == 200 ]] || fail_check 'source routes' "$loopback_origin$path" 'HTTP 200' "HTTP ${code:-000}"
done
REMOTE
}

http_request() {
    local stage=$1 check=$2 url=$3 body=$4 headers=$5
    local attempt backoff curl_exit http_status reason curl_stderr="$TEMP_DIR/curl.stderr"
    shift 5

    for ((attempt = 1; attempt <= CURL_RETRY_ATTEMPTS; attempt++)); do
        : >"$curl_stderr"
        set +e
        http_status=$(curl -sS --connect-timeout "$CURL_CONNECT_TIMEOUT" --max-time "$CURL_MAX_TIME" \
            "${CURL_NETWORK_OPTIONS[@]}" -D "$headers" -o "$body" -w '%{http_code}' "$@" "$url" \
            2>"$curl_stderr")
        curl_exit=$?
        set -e
        HTTP_REQUEST_EXIT=$curl_exit
        HTTP_REQUEST_STATUS=$http_status
        HTTP_REQUEST_ATTEMPTS=$attempt
        HTTP_REQUEST_STDERR=$curl_stderr

        if (( curl_exit != 0 )); then
            case $curl_exit in
                6|7|16|28|35|97)
                    if (( attempt < CURL_RETRY_ATTEMPTS )); then
                        backoff=$attempt
                        (( backoff <= CURL_RETRY_MAX_BACKOFF )) || backoff=$CURL_RETRY_MAX_BACKOFF
                        reason="curl-exit-$curl_exit HTTP-${http_status:-000}"
                        printf '[verify-production] retry locale=%s stage=%s check=%s attempt=%d/%d reason=%s next=retry backoff=%ss\n' \
                            "$RELEASE_LOCALE" "$stage" "$check" "$attempt" "$CURL_RETRY_ATTEMPTS" "$reason" "$backoff"
                        sleep "$backoff"
                        continue
                    fi
                    ;;
            esac
            return 0
        fi
        case $http_status in
            522|525)
                if (( attempt < CURL_RETRY_ATTEMPTS )); then
                    backoff=$attempt
                    (( backoff <= CURL_RETRY_MAX_BACKOFF )) || backoff=$CURL_RETRY_MAX_BACKOFF
                    printf '[verify-production] retry locale=%s stage=%s check=%s attempt=%d/%d reason=HTTP-%s next=retry backoff=%ss\n' \
                        "$RELEASE_LOCALE" "$stage" "$check" "$attempt" "$CURL_RETRY_ATTEMPTS" "$http_status" "$backoff"
                    sleep "$backoff"
                    continue
                fi
                return 0
                ;;
        esac
        return 0
    done
}

http_result_is() {
    local stage=$1 check=$2 expected=$3
    if (( HTTP_REQUEST_EXIT != 0 )); then
        if [[ -n $HTTP_REQUEST_STDERR && -s $HTTP_REQUEST_STDERR ]]; then
            printf '[verify-production] final curl stderr (locale=%s stage=%s check=%s attempt=%d/%d):\n' \
                "$RELEASE_LOCALE" "$stage" "$check" "$HTTP_REQUEST_ATTEMPTS" "$CURL_RETRY_ATTEMPTS" >&2
            cat -- "$HTTP_REQUEST_STDERR" >&2
        fi
        fail_check "$stage" "$check" "curl exit 0 and HTTP $expected" \
            "curl exit $HTTP_REQUEST_EXIT; HTTP ${HTTP_REQUEST_STATUS:-000}; attempts $HTTP_REQUEST_ATTEMPTS/$CURL_RETRY_ATTEMPTS"
        return 1
    fi
    [[ $HTTP_REQUEST_STATUS == "$expected" ]] || {
        fail_check "$stage" "$check" "HTTP $expected" \
            "HTTP ${HTTP_REQUEST_STATUS:-000}; attempts $HTTP_REQUEST_ATTEMPTS/$CURL_RETRY_ATTEMPTS"
        return 1
    }
    if (( HTTP_REQUEST_ATTEMPTS > 1 )); then
        printf '[verify-production] recovered locale=%s stage=%s check=%s attempt=%d/%d PASS\n' \
            "$RELEASE_LOCALE" "$stage" "$check" "$HTTP_REQUEST_ATTEMPTS" "$CURL_RETRY_ATTEMPTS"
    fi
}

header_value() {
    local headers=$1 header=$2
    awk -v wanted="$header" '
        BEGIN { value = "" }
        {
            name = $1
            sub(/:$/, "", name)
            if (tolower(name) == tolower(wanted)) value = $2
        }
        END {
            sub(/\r$/, "", value)
            print value
        }
    ' "$headers"
}

verify_cache_path() {
    local path=$1 result_variable=$2 headers cache_status observation=''
    local attempt all_miss=1
    headers="$TEMP_DIR/cache-$(printf '%s' "$path" | tr '/.' '__').headers"

    for ((attempt = 1; attempt <= 3; attempt++)); do
        http_request 'CDN cache observation' "$PUBLIC_ORIGIN$path request $attempt" \
            "$PUBLIC_ORIGIN$path" /dev/null "$headers"
        http_result_is 'CDN cache observation' "$PUBLIC_ORIGIN$path request $attempt" 200 || return 1
        cache_status=$(header_value "$headers" "$CACHE_HEADER")
        case $cache_status in
            MISS|HIT|EXPIRED|REVALIDATED|UPDATING|STALE) ;;
            '')
                fail_check 'CDN cache observation' "$PUBLIC_ORIGIN$path request $attempt $CACHE_HEADER" \
                    'present allowed cache status' missing
                return 1
                ;;
            *)
                fail_check 'CDN cache observation' "$PUBLIC_ORIGIN$path request $attempt $CACHE_HEADER" \
                    'MISS|HIT|EXPIRED|REVALIDATED|UPDATING|STALE' "$cache_status"
                return 1
                ;;
        esac
        [[ $cache_status == MISS ]] || all_miss=0
        if [[ -n $observation ]]; then
            observation+=' -> '
        fi
        observation+=$cache_status
    done
    if (( all_miss )); then
        printf -v "$result_variable" '%s PASS (cache not warm yet)' "$observation"
    else
        printf -v "$result_variable" '%s PASS' "$observation"
    fi
}

verify_public_routes() {
    local path headers="$TEMP_DIR/public.headers"
    # / and /tour/welcome/1 already returned HTTP 200 during cache status observation.
    for path in '/tour/' '/tour/list' '/tour/static/js/app.js' '/robots.txt' '/sitemap.xml'; do
        http_request 'public routes' "$PUBLIC_ORIGIN$path" "$PUBLIC_ORIGIN$path" /dev/null "$headers"
        http_result_is 'public routes' "$PUBLIC_ORIGIN$path" 200 || return 1
    done
}

fetch_http_200() {
    local stage=$1 url=$2 destination=$3 headers="$TEMP_DIR/fetch.headers"
    http_request "$stage" "$url" "$url" "$destination" "$headers"
    http_result_is "$stage" "$url" 200
}

verify_html_identity() {
    local home_html="$TEMP_DIR/home.html" welcome_html="$TEMP_DIR/welcome.html"
    fetch_http_200 'html identity' "$PUBLIC_ORIGIN/" "$home_html" || return 1
    fetch_http_200 'html identity' "$PUBLIC_ORIGIN/tour/welcome/1" "$welcome_html" || return 1
    python3 - "$RELEASE_LOCALE" "$home_html" "$PUBLIC_ORIGIN/" \
        "$welcome_html" "$PUBLIC_ORIGIN/tour/welcome/1" <<'PY'
from html.parser import HTMLParser
import sys


class IdentityParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.langs = []
        self.canonicals = []

    def handle_starttag(self, tag, attrs):
        values = dict(attrs)
        if tag.lower() == "html":
            self.langs.append(values.get("lang"))
        if tag.lower() == "link" and "canonical" in values.get("rel", "").lower().split():
            self.canonicals.append(values.get("href"))


locale = sys.argv[1]


def fail(check, expected, actual):
    print(
        f"[verify-production] ERROR: locale={locale} stage=html identity check={check} "
        f"expected={expected} actual={actual}",
        file=sys.stderr,
    )
    raise SystemExit(1)


for path, expected_canonical in ((sys.argv[2], sys.argv[3]), (sys.argv[4], sys.argv[5])):
    parser = IdentityParser()
    try:
        with open(path, encoding="utf-8") as source:
            parser.feed(source.read())
    except (OSError, UnicodeError) as exc:
        fail(path, "readable UTF-8 HTML", repr(exc))
    if parser.langs != [locale]:
        fail(path + " html-lang", repr([locale]), repr(parser.langs))
    if parser.canonicals != [expected_canonical]:
        fail(path + " canonical", repr([expected_canonical]), repr(parser.canonicals))
PY
}

verify_sitemap() {
    local sitemap="$TEMP_DIR/sitemap.xml" urls_file="$TEMP_DIR/sitemap.urls" path
    local -a urls=()

    fetch_http_200 'sitemap' "$PUBLIC_ORIGIN/sitemap.xml" "$sitemap" || return 1
    if ! python3 - "$RELEASE_LOCALE" "$sitemap" "$PRODUCTION_HOST" "$EXPECTED_SITEMAP_URLS" >"$urls_file" <<'PY'
import sys
import re
import urllib.parse
import xml.etree.ElementTree as ET


locale = sys.argv[1]


def error(check, expected, actual):
    print(
        f"[verify-production] ERROR: locale={locale} stage=sitemap check={check} expected={expected} actual={actual}",
        file=sys.stderr,
    )


path, expected_host, expected_count = sys.argv[2], sys.argv[3], int(sys.argv[4])
try:
    root = ET.parse(path).getroot()
except (OSError, ET.ParseError) as exc:
    error("XML", "valid sitemap XML", repr(exc))
    raise SystemExit(1)
namespace = "{http://www.sitemaps.org/schemas/sitemap/0.9}"
urls = [(node.text or "").strip() for node in root.findall(f"{namespace}url/{namespace}loc")]
failed = False
if len(urls) != expected_count:
    error("URL-count", expected_count, len(urls))
    failed = True
expected_prefix = [f"https://{expected_host}/", f"https://{expected_host}/tour/list"]
if urls[:2] != expected_prefix:
    error("catalog-prefix", repr(expected_prefix), repr(urls[:2]))
    failed = True
duplicates = sorted({url for url in urls if urls.count(url) > 1})
for url in duplicates:
    error("duplicate-URL", "unique URL", url)
    failed = True
for url in urls:
    parsed = urllib.parse.urlsplit(url)
    if parsed.scheme != "https":
        error("scheme", "https", f"{parsed.scheme or 'missing'} URL={url}")
        failed = True
    if parsed.hostname != expected_host or parsed.netloc != expected_host:
        error("hostname", expected_host, f"{parsed.netloc or 'missing'} URL={url}")
        failed = True
course_urls = urls[2:] if urls[:2] == expected_prefix else []
if len(course_urls) != 103:
    error("course-page-count", 103, len(course_urls))
    failed = True
for url in course_urls:
    parsed = urllib.parse.urlsplit(url)
    if not re.fullmatch(r"/tour/[^/]+/[1-9][0-9]*", parsed.path) or parsed.query or parsed.fragment:
        error("course-URL", "https://<production-host>/tour/<lesson>/<positive-page>", url)
        failed = True
if failed:
    raise SystemExit(1)
sys.stdout.buffer.write(b"\0".join(url.encode("utf-8") for url in urls) + b"\0")
PY
    then
        return 1
    fi
    mapfile -d '' -t urls <"$urls_file"
    for path in "${urls[@]}"; do
        http_request sitemap "$path" "$path" /dev/null "$TEMP_DIR/sitemap-url.headers"
        # A bounded transient retry has already been exhausted.  Stop now;
        # later URLs are unverified and must never be reported as passed.
        http_result_is sitemap "$path" 200 || return 1
    done
    printf 'sitemap URLs: %d/%d\n' "${#urls[@]}" "$EXPECTED_SITEMAP_URLS"
    printf 'host mismatch: 0\n'
    printf 'HTTP failure: 0\n'
}

public_main() {
    (( $# == 4 )) || { usage; return 2; }
    RELEASE_LOCALE=$1; PUBLIC_ORIGIN=${2%/}; PRODUCTION_HOST=$3; CACHE_HEADER=$4
    for command_name in awk cat curl mktemp python3 tr; do command -v "$command_name" >/dev/null || { fail_check 'public runner' tool "$command_name available" missing; return 1; }; done
    TEMP_DIR=$(mktemp -d) || return 1
    trap 'rm -rf -- "$TEMP_DIR"' EXIT
    printf '[verify-production] public network runner: zgocloud (direct)\n'
    printf '[verify-production] progress locale=%s stage=CDN-cache-observation START probes=2x3\n' "$RELEASE_LOCALE"
    verify_cache_path '/' CACHE_HOME_RESULT || return 1
    verify_cache_path '/tour/welcome/1' CACHE_WELCOME_RESULT || return 1
    printf '[verify-production] progress locale=%s stage=CDN-cache-observation PASS\n' "$RELEASE_LOCALE"
    printf '[verify-production] progress locale=%s stage=public-routes START probes=5\n' "$RELEASE_LOCALE"
    verify_public_routes || return 1
    printf '[verify-production] public routes: 7/7 PASS\n'
    printf '[verify-production] progress locale=%s stage=public-routes PASS\n' "$RELEASE_LOCALE"
    printf '[verify-production] progress locale=%s stage=html-identity START probes=2\n' "$RELEASE_LOCALE"
    verify_html_identity || return 1
    printf '[verify-production] html identity: PASS\n'
    printf '[verify-production] progress locale=%s stage=html-identity PASS\n' "$RELEASE_LOCALE"
    printf '[verify-production] progress locale=%s stage=sitemap START URLs=%d\n' "$RELEASE_LOCALE" "$EXPECTED_SITEMAP_URLS"
    verify_sitemap || return 1
    printf '[verify-production] sitemap: 105/105 PASS\n'
    printf '[verify-production] progress locale=%s stage=sitemap PASS\n' "$RELEASE_LOCALE"
    printf '[verify-production] progress locale=%s stage=socket-boundary START probes=2\n' "$RELEASE_LOCALE"
    verify_socket_boundary || return 1
    printf '[verify-production] socket boundary: PASS\n'
    printf '[verify-production] progress locale=%s stage=socket-boundary PASS\n' "$RELEASE_LOCALE"
    printf '[verify-production] CDN /: %s\n' "$CACHE_HOME_RESULT"
    printf '[verify-production] CDN /tour/welcome/1: %s\n' "$CACHE_WELCOME_RESULT"
    printf '\nPRODUCTION MACHINE ACCEPTANCE: PASS\n'
}

run_public_acceptance() {
    load_production_identity_shared || { fail_check 'network runner' identity 'valid shared production identity' invalid; return 1; }
    NETWORK_SSH_HOST=$PRODUCTION_ZGOCLOUD_SSH_ALIAS
    ssh "${SSH_OPTIONS[@]}" "$NETWORK_SSH_HOST" bash -s -- --public \
        "$RELEASE_LOCALE" "$PUBLIC_ORIGIN" "$PRODUCTION_HOST" "$CACHE_HEADER" <"${BASH_SOURCE[0]}" || {
        fail_check 'public runner' SSH "$NETWORK_SSH_HOST direct public acceptance" failed
        return 1
    }
}

verify_socket_boundary() {
    local headers="$TEMP_DIR/socket.headers" url="$PUBLIC_ORIGIN/socket"
    http_request 'socket boundary' 'GET /socket' "$url" /dev/null "$headers"
    http_result_is 'socket boundary' 'GET /socket' 404 || return 1
    http_request 'socket boundary' 'Upgrade /socket' "$url" /dev/null "$headers" \
        --http1.1 -H 'Connection: Upgrade' -H 'Upgrade: websocket'
    http_result_is 'socket boundary' 'Upgrade /socket' 404
}

main() {
    local release_input release_dir release_name expected_remote

    (( $# == 1 )) || { usage; return 2; }
    validate_local_tools || return 1
    release_input=$1
    if [[ ! -d $release_input || -L $release_input ]]; then
        fail_check 'release identity' release-dir 'real directory (not symlink)' "$release_input"
        return 1
    fi
    release_dir=$(cd -P -- "$release_input" && pwd -P)
    if [[ ! -f $release_dir/release.json || -L $release_dir/release.json ]]; then
        fail_check 'release identity' release.json 'real regular file' missing-or-symlink
        return 1
    fi
    if ! RELEASE_LOCALE=$(read_release_locale "$release_dir"); then
        fail_check 'release identity' release.json 'valid locale metadata' invalid
        return 1
    fi
    select_production_profile "$RELEASE_LOCALE" || return 1
    release_name=$(remote_release_name "$release_dir") || return 1
    expected_remote="$RELEASES_DIR/$release_name"
    printf '[verify-production] release identity: PASS (%s -> %s)\n' "$RELEASE_LOCALE" "$expected_remote"

    TEMP_DIR=$(mktemp -d) || {
        fail_check 'release identity' temporary-directory created failed
        return 1
    }
    trap 'rm -rf -- "$TEMP_DIR"' EXIT

    if ! verify_remote_and_source "$expected_remote"; then
        error 'stage=remote/source batch check=SSH aliyun expected=completed actual=failed'
        return 1
    fi
    printf '[verify-production] remote identity: PASS\n'
    printf '[verify-production] source routes: 7/7 PASS\n'

    run_public_acceptance
}

if [[ ${1:-} == --public ]]; then
    shift
    public_main "$@"
elif [[ ${BASH_SOURCE[0]} == "$0" ]]; then
    main "$@"
fi
