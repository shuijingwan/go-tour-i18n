#!/usr/bin/env bash

set -Eeuo pipefail
IFS=$'\n\t'

script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repository_root=$(cd -P -- "$script_dir/.." && pwd -P)
verify_script=$script_dir/verify-shared-assets-production.sh
fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT

fail() { printf '[verify-shared-assets-production-test] FAIL: %s\n' "$*" >&2; exit 1; }
assert_contains() { [[ $1 == *"$2"* ]] || fail "output does not contain: $2"; }

fake_bin=$fixture/bin
export_dir=$fixture/formal-export
public_root=$fixture/public
real_python=$(command -v python3)
mkdir -p -- "$fake_bin" "$public_root"

(cd -- "$repository_root" && GOCACHE=/tmp/go-tour-i18n-go-build \
    go run -mod=readonly ./cmd/tour-i18n assets export --output "$export_dir") >/dev/null

cat >"$fake_bin/curl" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
headers='' body='' fail_http=0 url='' socks='' write_out=''
while (( $# )); do
    case $1 in
        -f|-sS) [[ $1 == -f ]] && fail_http=1; shift ;;
        --socks5-hostname) socks=$2; shift 2 ;;
        -D) headers=$2; shift 2 ;;
        -o) body=$2; shift 2 ;;
        -w) write_out=$2; shift 2 ;;
        *) url=$1; shift ;;
    esac
done
if [[ ${FAKE_REQUIRE_SOCKS:-0} == 1 ]]; then
    [[ $socks == 127.0.0.1:* ]] || exit 91
    printf 'curl %s\n' "$socks" >>"${FAKE_NETWORK_LOG:?}"
fi
path=${url#https://assets-go-dev.shuijingwanwq.com/}
status=200
case $path in
    tour/script.js|tour/static/img/tree.png|tour/static/partials/editor.html) status=${FAKE_BOUNDARY_STATUS:-404} ;;
    *) [[ -f $FAKE_PUBLIC_ROOT/$path ]] || status=404 ;;
esac
cache=''
if [[ -n ${FAKE_STATUS_PATH:-} && $path == "$FAKE_STATUS_PATH" ]]; then
    counter=${FAKE_STATUS_COUNTER:?}
    count=0; [[ -f $counter ]] && count=$(<"$counter")
    count=$((count + 1)); printf '%s' "$count" >"$counter"
    IFS=',' read -r -a status_sequence <<<"${FAKE_STATUS_SEQUENCE:?}"
    sequence_index=$((count - 1))
    if (( sequence_index >= ${#status_sequence[@]} )); then
        sequence_index=$((${#status_sequence[@]} - 1))
    fi
    status=${status_sequence[sequence_index]}
fi
if [[ -n $headers ]]; then
    if [[ ${FAKE_CACHE_STATUS:-} == ERROR ]]; then cache=BYPASS
    elif [[ $status == 200 ]]; then
        counter=${FAKE_CURL_COUNTER:?}
        count=0; [[ -f $counter ]] && count=$(<"$counter")
        count=$((count + 1)); printf '%s' "$count" >"$counter"
        (( count % 2 == 1 )) && cache=MISS || cache=HIT
    fi
    printf 'HTTP/2 %s OK\r\n' "$status" >"$headers"
    [[ -n $cache ]] && printf 'CF-Cache-Status: %s\r\n' "$cache" >>"$headers"
    printf '\r\n' >>"$headers"
fi
if [[ $status == 200 && -n $body && $body != /dev/null ]]; then cp -- "$FAKE_PUBLIC_ROOT/$path" "$body"; fi
if (( fail_http )) && [[ $status != 200 ]]; then exit 22; fi
[[ -z $write_out ]] || printf '%s' "$status"
SH
chmod 0755 -- "$fake_bin/curl"

cat >"$fake_bin/ssh" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
arguments=("$@")
control=''
for value in "${arguments[@]}"; do
    [[ $value == ControlPath=* ]] && control=${value#ControlPath=}
done
if [[ " ${arguments[*]} " == *' -O exit '* ]]; then
    printf 'cleanup\n' >>"${FAKE_NETWORK_LOG:?}"
    [[ -f $control.pid ]] && kill "$(<"$control.pid")" 2>/dev/null || true
    rm -f -- "$control" "$control.pid"
    exit 0
fi
if [[ " ${arguments[*]} " == *' -N '* && " ${arguments[*]} " == *' -D '* ]]; then
    [[ ${arguments[-1]} == zgocloud ]] || exit 92
    [[ ${FAKE_SSH_NETWORK_FAIL:-0} != 1 ]] || exit 93
    printf 'setup\n' >>"${FAKE_NETWORK_LOG:?}"
    : >"$control"
    exit 0
fi
exit 94
SH
chmod 0755 -- "$fake_bin/ssh"

cat >"$fake_bin/python3" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
if [[ ${1:-} == - ]]; then
    source=$(cat)
    if [[ $source == *'listener.bind(("127.0.0.1", 0))'* ]]; then
        printf '45678\n'
    else
        printf '%s' "$source" | "$FAKE_REAL_PYTHON" "$@"
    fi
else
    exec "$FAKE_REAL_PYTHON" "$@"
fi
SH
chmod 0755 -- "$fake_bin/python3"

make_receipt() {
    local destination=$1 result=$2 changed=$3 base=${4:-https://assets-go-dev.shuijingwanwq.com} digest
    digest=$(sha256sum -- "$export_dir/SHA256SUMS"); digest=${digest%% *}
    printf '{\n  "schema": "go-tour-i18n/shared-assets-production-receipt/v1",\n  "export_dir": "%s",\n  "manifest_sha256": "%s",\n  "deployment_result": "%s",\n  "production_base_url": "%s",\n  "changed_paths": [%s],\n  "boundary_paths": ["tour/script.js","tour/static/img/tree.png","tour/static/partials/editor.html"]\n}\n' \
        "$export_dir" "$digest" "$result" "$base" "$changed" >"$destination"
}

reset_public() {
    rm -rf -- "$public_root"; mkdir -p -- "$public_root"
    cp -a -- "$export_dir/." "$public_root/"
}

run_verify() {
    local receipt=$1
    rm -f -- "$fixture/curl-count" "$fixture/network.log" "$fixture/status-count"
    env PATH="$fake_bin:$PATH" FAKE_PUBLIC_ROOT="$public_root" FAKE_CURL_COUNTER="$fixture/curl-count" \
        FAKE_REQUIRE_SOCKS=1 FAKE_NETWORK_LOG="$fixture/network.log" FAKE_REAL_PYTHON="$real_python" \
        FAKE_STATUS_PATH="${FAKE_STATUS_PATH:-}" FAKE_STATUS_SEQUENCE="${FAKE_STATUS_SEQUENCE:-}" FAKE_STATUS_COUNTER="$fixture/status-count" \
        "$verify_script" "$receipt" 2>&1
}

network_count() {
    local kind=$1
    [[ -f $fixture/network.log ]] && grep -c "^$kind" "$fixture/network.log" || true
}

status_count() {
    [[ -f $fixture/status-count ]] && cat "$fixture/status-count" || printf '0\n'
}

clear_status_sequence() {
    unset FAKE_STATUS_PATH FAKE_STATUS_SEQUENCE
}

set +e
output=$("$verify_script" 2>&1); status=$?
set -e
[[ $status == 2 ]] || fail 'zero-argument invocation did not exit 2'
assert_contains "$output" 'usage:'
set +e
output=$("$verify_script" one two 2>&1); status=$?
set -e
[[ $status == 2 ]] || fail 'two-argument invocation did not exit 2'
assert_contains "$output" 'usage:'

receipt=$export_dir.verification-receipt.json
reset_public
make_receipt "$receipt" DEPLOYED '"SHA256SUMS","tour/static/css/app.css"'
rm -f -- "$fixture/curl-count"
output=$(run_verify "$receipt") || fail 'DEPLOYED verification failed'
assert_contains "$output" 'CACHE VERIFIED: MISS -> HIT'
assert_contains "$output" 'SHARED ASSETS PRODUCTION VERIFICATION: PASSED'
assert_contains "$output" 'public network runner: zgocloud'
[[ $(network_count setup) == 1 ]] || fail 'DEPLOYED did not establish exactly one network ControlMaster'
[[ $(network_count curl) == 18 ]] || fail 'DEPLOYED public requests did not all reuse SOCKS'
[[ $(network_count cleanup) == 1 ]] || fail 'DEPLOYED did not clean up network ControlMaster'

make_receipt "$receipt" NO_CHANGES ''
output=$(run_verify "$receipt") || fail 'NO_CHANGES verification failed'
assert_contains "$output" 'SKIP CACHE PURGE VERIFICATION: NO CHANGES'
[[ $(network_count setup) == 1 ]] || fail 'NO_CHANGES did not establish exactly one network ControlMaster'
[[ $(network_count curl) == 14 ]] || fail 'NO_CHANGES public requests did not all reuse SOCKS'
[[ $(network_count cleanup) == 1 ]] || fail 'NO_CHANGES did not clean up network ControlMaster'

reset_public; make_receipt "$receipt" NO_CHANGES ''
export FAKE_STATUS_PATH=images/go-logo-white.svg FAKE_STATUS_SEQUENCE=525,200
output=$(run_verify "$receipt") || fail 'asset 525 then 200 did not pass'
assert_contains "$output" 'transient HTTP 525, retry 1/2: https://assets-go-dev.shuijingwanwq.com/images/go-logo-white.svg'
[[ $(status_count) == 2 ]] || fail 'asset 525 then 200 did not use exactly two attempts'
clear_status_sequence

reset_public; make_receipt "$receipt" NO_CHANGES ''
export FAKE_STATUS_PATH=images/go-logo-white.svg FAKE_STATUS_SEQUENCE=522,522,200
output=$(run_verify "$receipt") || fail 'asset 522,522 then 200 did not pass'
[[ $(status_count) == 3 ]] || fail 'asset 522,522 then 200 did not use exactly three attempts'
clear_status_sequence

reset_public; make_receipt "$receipt" NO_CHANGES ''
export FAKE_STATUS_PATH=images/go-logo-white.svg FAKE_STATUS_SEQUENCE=525,525,525
if run_verify "$receipt" >/dev/null; then fail 'three transient 525 responses accepted'; fi
[[ $(status_count) == 3 ]] || fail 'three transient 525 responses exceeded attempt limit'
clear_status_sequence

reset_public; make_receipt "$receipt" NO_CHANGES ''
export FAKE_STATUS_PATH=images/go-logo-white.svg FAKE_STATUS_SEQUENCE=500,200
if run_verify "$receipt" >/dev/null; then fail 'HTTP 500 was retried or accepted'; fi
[[ $(status_count) == 1 ]] || fail 'HTTP 500 was retried'
clear_status_sequence

make_receipt "$receipt" NO_CHANGES ''
rm -f -- "$fixture/network.log"
if env PATH="$fake_bin:$PATH" FAKE_PUBLIC_ROOT="$public_root" FAKE_CURL_COUNTER="$fixture/curl-count" \
    FAKE_REQUIRE_SOCKS=1 FAKE_NETWORK_LOG="$fixture/network.log" FAKE_REAL_PYTHON="$real_python" FAKE_SSH_NETWORK_FAIL=1 \
    "$verify_script" "$receipt" >/dev/null 2>&1; then
    fail 'network runner setup failure accepted'
fi
[[ ! -s $fixture/network.log ]] || fail 'network runner setup failure fell back to local curl'

make_receipt "$receipt" DEPLOYED '"../unsafe"'
if run_verify "$receipt" >/dev/null; then fail 'unsafe changed path accepted'; fi

make_receipt "$receipt" NO_CHANGES ''
sed -i '/manifest_sha256/d' "$receipt"
if run_verify "$receipt" >/dev/null; then fail 'receipt with missing field accepted'; fi

make_receipt "$receipt" NO_CHANGES ''
mv -- "$receipt" "$fixture/receipt-real.json"
ln -s -- "$fixture/receipt-real.json" "$receipt"
if run_verify "$receipt" >/dev/null; then fail 'receipt symlink accepted'; fi
rm -- "$receipt"; mv -- "$fixture/receipt-real.json" "$receipt"

make_receipt "$receipt" NO_CHANGES ''
sed -i 's/"manifest_sha256": "[0-9a-f]*/"manifest_sha256": "0000000000000000000000000000000000000000000000000000000000000000/' "$receipt"
if run_verify "$receipt" >/dev/null; then fail 'stale receipt manifest identity accepted'; fi

stale_export=$fixture/stale-export
cp -a -- "$export_dir" "$stale_export"
printf 'stale export\n' >>"$stale_export/tour/static/css/app.css"
(cd -- "$stale_export" && find . -type f ! -name SHA256SUMS -printf '%P\0' | LC_ALL=C sort -z | xargs -0 sha256sum >SHA256SUMS)
saved_export=$export_dir; export_dir=$stale_export; receipt=$export_dir.verification-receipt.json
make_receipt "$receipt" NO_CHANGES ''
if run_verify "$receipt" >/dev/null; then fail 'export stale from repository source accepted'; fi
export_dir=$saved_export; receipt=$export_dir.verification-receipt.json

reset_public; make_receipt "$receipt" DEPLOYED '"tour/static/css/app.css"'
if env PATH="$fake_bin:$PATH" FAKE_PUBLIC_ROOT="$public_root" FAKE_CURL_COUNTER="$fixture/curl-count" FAKE_CACHE_STATUS=ERROR FAKE_REAL_PYTHON="$real_python" "$verify_script" "$receipt" >/dev/null 2>&1; then fail 'cache status failure accepted'; fi

reset_public; printf 'wrong public asset\n' >"$public_root/tour/static/css/app.css"
make_receipt "$receipt" NO_CHANGES ''
if run_verify "$receipt" >/dev/null; then fail 'public SHA mismatch accepted'; fi
[[ $(network_count cleanup) == 1 ]] || fail 'public SHA mismatch did not clean up network ControlMaster'

reset_public; printf 'wrong public asset\n' >"$public_root/images/go-logo-white.svg"
make_receipt "$receipt" NO_CHANGES ''
if run_verify "$receipt" >/dev/null; then fail 'first public SHA mismatch accepted'; fi
[[ $(network_count curl) == 1 ]] || fail 'public SHA mismatch retried the logical request'

reset_public; make_receipt "$receipt" NO_CHANGES ''
export FAKE_STATUS_PATH=tour/script.js FAKE_STATUS_SEQUENCE=525,404
output=$(run_verify "$receipt") || fail 'boundary 525 then 404 did not pass'
[[ $(status_count) == 2 ]] || fail 'boundary 525 then 404 did not use exactly two attempts'
clear_status_sequence

reset_public; make_receipt "$receipt" NO_CHANGES ''
export FAKE_STATUS_PATH=tour/script.js FAKE_STATUS_SEQUENCE=525,200
if run_verify "$receipt" >/dev/null; then fail 'boundary 525 then 200 accepted'; fi
[[ $(status_count) == 2 ]] || fail 'boundary 525 then 200 did not fail on second response'
clear_status_sequence

reset_public; make_receipt "$receipt" DEPLOYED '"tour/static/css/app.css"'
export FAKE_STATUS_PATH=tour/static/css/app.css FAKE_STATUS_SEQUENCE=525,200,200
output=$(run_verify "$receipt") || fail 'cache transient then MISS/HIT did not pass'
assert_contains "$output" 'CACHE VERIFIED: MISS -> HIT'
[[ $(status_count) == 4 ]] || fail 'cache transient request did not preserve two successful observations'
clear_status_sequence

reset_public; make_receipt "$receipt" NO_CHANGES ''
if env PATH="$fake_bin:$PATH" FAKE_PUBLIC_ROOT="$public_root" FAKE_CURL_COUNTER="$fixture/curl-count" FAKE_BOUNDARY_STATUS=200 FAKE_REAL_PYTHON="$real_python" "$verify_script" "$receipt" >/dev/null 2>&1; then fail 'boundary non-404 accepted'; fi

make_receipt "$receipt" NO_CHANGES '' 'https://assets.invalid'
if run_verify "$receipt" >/dev/null; then fail 'unsupported base URL accepted'; fi

printf '[verify-shared-assets-production-test] PASS\n'
