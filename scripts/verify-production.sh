#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

readonly SSH_HOST='aliyun'
readonly -a SSH_OPTIONS=(-o BatchMode=yes -o ConnectTimeout=10)
readonly CURL_CONNECT_TIMEOUT=5
readonly CURL_MAX_TIME=15
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

error() {
    printf '[verify-production] ERROR: %s\n' "$*" >&2
}

fail_check() {
    local stage=$1 check=$2 expected=$3 actual=$4
    error "stage=$stage check=$check expected=$expected actual=$actual"
    return 1
}

usage() {
    printf 'usage: %s <release-dir>\n' "${0##*/}" >&2
}

select_production_profile() {
    local locale=$1

    case $locale in
        zh-CN)
            RELEASES_DIR='/data/go-tour/releases'
            CURRENT_LINK='/data/go-tour/current'
            DEPLOY_LOCK='/data/go-tour/.deploy.lock'
            SERVICE='go-tour.service'
            LOOPBACK_ORIGIN='http://127.0.0.1:3999'
            PUBLIC_ORIGIN='https://go-dev.shuijingwanwq.com'
            PRODUCTION_HOST='go-dev.shuijingwanwq.com'
            CACHE_HEADER='EO-Cache-Status'
            ;;
        ja-JP)
            RELEASES_DIR='/data/go-tour-ja-JP/releases'
            CURRENT_LINK='/data/go-tour-ja-JP/current'
            DEPLOY_LOCK='/data/go-tour-ja-JP/.deploy.lock'
            SERVICE='go-tour-ja-JP.service'
            LOOPBACK_ORIGIN='http://127.0.0.1:4000'
            PUBLIC_ORIGIN='https://ja-go-dev.shuijingwanwq.com'
            PRODUCTION_HOST='ja-go-dev.shuijingwanwq.com'
            CACHE_HEADER='CF-Cache-Status'
            ;;
        de-DE)
            RELEASES_DIR='/data/go-tour-de-DE/releases'
            CURRENT_LINK='/data/go-tour-de-DE/current'
            DEPLOY_LOCK='/data/go-tour-de-DE/.deploy.lock'
            SERVICE='go-tour-de-DE.service'
            LOOPBACK_ORIGIN='http://127.0.0.1:4001'
            PUBLIC_ORIGIN='https://de-go-dev.shuijingwanwq.com'
            PRODUCTION_HOST='de-go-dev.shuijingwanwq.com'
            CACHE_HEADER='CF-Cache-Status'
            ;;
        *)
            fail_check 'release identity' locale 'supported production locale (zh-CN, ja-JP, de-DE)' "$locale"
            return 1
            ;;
    esac
}

validate_local_tools() {
    local command_name
    for command_name in awk basename curl mktemp python3 readlink ssh tr; do
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
        "$RELEASES_DIR" "$CURRENT_LINK" "$DEPLOY_LOCK" "$SERVICE" \
        "$LOOPBACK_ORIGIN" "$expected_remote" "${ACCEPTANCE_PATHS[@]}" <<'REMOTE'
set -Eeuo pipefail
IFS=$'\n\t'

releases_dir=$1
current_link=$2
deploy_lock=$3
service=$4
loopback_origin=$5
expected_remote=$6
shift 6
paths=("$@")

fail_check() {
    printf '[verify-production] ERROR: stage=%s check=%s expected=%s actual=%s\n' \
        "$1" "$2" "$3" "$4" >&2
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
    local url=$1 body=$2 headers=$3
    shift 3
    curl -sS --connect-timeout "$CURL_CONNECT_TIMEOUT" --max-time "$CURL_MAX_TIME" \
        -D "$headers" -o "$body" -w '%{http_code}' "$@" "$url" || true
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
    local path=$1 result_variable=$2 headers code cache_status observation=''
    local attempt all_miss=1
    headers="$TEMP_DIR/cache-$(printf '%s' "$path" | tr '/.' '__').headers"

    for ((attempt = 1; attempt <= 3; attempt++)); do
        code=$(http_request "$PUBLIC_ORIGIN$path" /dev/null "$headers")
        [[ $code == 200 ]] || {
            fail_check 'CDN cache observation' "$PUBLIC_ORIGIN$path request $attempt" 'HTTP 200' "HTTP ${code:-000}"
            return 1
        }
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
    local path code headers="$TEMP_DIR/public.headers"
    # / and /tour/welcome/1 already returned HTTP 200 during cache status observation.
    for path in '/tour/' '/tour/list' '/tour/static/js/app.js' '/robots.txt' '/sitemap.xml'; do
        code=$(http_request "$PUBLIC_ORIGIN$path" /dev/null "$headers")
        [[ $code == 200 ]] || {
            fail_check 'public routes' "$PUBLIC_ORIGIN$path" 'HTTP 200' "HTTP ${code:-000}"
            return 1
        }
    done
}

fetch_http_200() {
    local stage=$1 url=$2 destination=$3 code headers="$TEMP_DIR/fetch.headers"
    code=$(http_request "$url" "$destination" "$headers")
    [[ $code == 200 ]] || {
        fail_check "$stage" "$url" 'HTTP 200' "HTTP ${code:-000}"
        return 1
    }
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


def fail(check, expected, actual):
    print(
        f"[verify-production] ERROR: stage=html identity check={check} "
        f"expected={expected} actual={actual}",
        file=sys.stderr,
    )
    raise SystemExit(1)


locale = sys.argv[1]
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
    local sitemap="$TEMP_DIR/sitemap.xml" urls_file="$TEMP_DIR/sitemap.urls" path code
    local failures=0
    local -a urls=()

    fetch_http_200 'sitemap' "$PUBLIC_ORIGIN/sitemap.xml" "$sitemap" || return 1
    if ! python3 - "$sitemap" "$PRODUCTION_HOST" "$EXPECTED_SITEMAP_URLS" >"$urls_file" <<'PY'
import sys
import re
import urllib.parse
import xml.etree.ElementTree as ET


def error(check, expected, actual):
    print(
        f"[verify-production] ERROR: stage=sitemap check={check} expected={expected} actual={actual}",
        file=sys.stderr,
    )


path, expected_host, expected_count = sys.argv[1], sys.argv[2], int(sys.argv[3])
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
        code=$(http_request "$path" /dev/null "$TEMP_DIR/sitemap-url.headers")
        if [[ $code != 200 ]]; then
            error "stage=sitemap check=$path expected=HTTP 200 actual=HTTP ${code:-000}"
            failures=$((failures + 1))
        fi
    done
    printf 'sitemap URLs: %d/%d\n' "${#urls[@]}" "$EXPECTED_SITEMAP_URLS"
    printf 'host mismatch: 0\n'
    printf 'HTTP failure: %d\n' "$failures"
    (( failures == 0 )) || return 1
}

verify_socket_boundary() {
    local code headers="$TEMP_DIR/socket.headers" url="$PUBLIC_ORIGIN/socket"
    code=$(http_request "$url" /dev/null "$headers")
    [[ $code == 404 ]] || {
        fail_check 'socket boundary' 'GET /socket' 'HTTP 404' "HTTP ${code:-000}"
        return 1
    }
    code=$(http_request "$url" /dev/null "$headers" --http1.1 -H 'Connection: Upgrade' -H 'Upgrade: websocket')
    [[ $code == 404 ]] || {
        fail_check 'socket boundary' 'Upgrade /socket' 'HTTP 404' "HTTP ${code:-000}"
        return 1
    }
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

    verify_cache_path '/' CACHE_HOME_RESULT || return 1
    verify_cache_path '/tour/welcome/1' CACHE_WELCOME_RESULT || return 1

    verify_public_routes || return 1
    printf '[verify-production] public routes: 7/7 PASS\n'

    verify_html_identity || return 1
    printf '[verify-production] html identity: PASS\n'

    verify_sitemap || return 1
    printf '[verify-production] sitemap: 105/105 PASS\n'

    verify_socket_boundary || return 1
    printf '[verify-production] socket boundary: PASS\n'
    printf '[verify-production] CDN /: %s\n' "$CACHE_HOME_RESULT"
    printf '[verify-production] CDN /tour/welcome/1: %s\n' "$CACHE_WELCOME_RESULT"
    printf '\nPRODUCTION MACHINE ACCEPTANCE: PASS\n'
}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
    main "$@"
fi
