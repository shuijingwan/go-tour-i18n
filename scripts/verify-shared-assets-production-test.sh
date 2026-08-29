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
mkdir -p -- "$fake_bin" "$public_root"

(cd -- "$repository_root" && GOCACHE=/tmp/go-tour-i18n-go-build \
    go run -mod=readonly ./cmd/tour-i18n assets export --output "$export_dir") >/dev/null

cat >"$fake_bin/curl" <<'SH'
#!/usr/bin/env bash
set -Eeuo pipefail
headers='' body='' fail_http=0 url=''
while (( $# )); do
    case $1 in
        -f|-sS) [[ $1 == -f ]] && fail_http=1; shift ;;
        -D) headers=$2; shift 2 ;;
        -o) body=$2; shift 2 ;;
        *) url=$1; shift ;;
    esac
done
path=${url#https://assets-go-dev.shuijingwanwq.com/}
status=200
case $path in
    tour/script.js|tour/static/img/tree.png|tour/static/partials/editor.html) status=${FAKE_BOUNDARY_STATUS:-404} ;;
    *) [[ -f $FAKE_PUBLIC_ROOT/$path ]] || status=404 ;;
esac
cache=''
if [[ -n $headers ]]; then
    if [[ ${FAKE_CACHE_STATUS:-} == ERROR ]]; then cache=BYPASS
    else
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
SH
chmod 0755 -- "$fake_bin/curl"

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
    env PATH="$fake_bin:$PATH" FAKE_PUBLIC_ROOT="$public_root" FAKE_CURL_COUNTER="$fixture/curl-count" \
        "$verify_script" "$receipt" 2>&1
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

make_receipt "$receipt" NO_CHANGES ''
rm -f -- "$fixture/curl-count"
output=$(run_verify "$receipt") || fail 'NO_CHANGES verification failed'
assert_contains "$output" 'SKIP CACHE PURGE VERIFICATION: NO CHANGES'

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
if env PATH="$fake_bin:$PATH" FAKE_PUBLIC_ROOT="$public_root" FAKE_CURL_COUNTER="$fixture/curl-count" FAKE_CACHE_STATUS=ERROR "$verify_script" "$receipt" >/dev/null 2>&1; then fail 'cache status failure accepted'; fi

reset_public; printf 'wrong public asset\n' >"$public_root/tour/static/css/app.css"
make_receipt "$receipt" NO_CHANGES ''
if run_verify "$receipt" >/dev/null; then fail 'public SHA mismatch accepted'; fi

reset_public; make_receipt "$receipt" NO_CHANGES ''
if env PATH="$fake_bin:$PATH" FAKE_PUBLIC_ROOT="$public_root" FAKE_CURL_COUNTER="$fixture/curl-count" FAKE_BOUNDARY_STATUS=200 "$verify_script" "$receipt" >/dev/null 2>&1; then fail 'boundary non-404 accepted'; fi

make_receipt "$receipt" NO_CHANGES '' 'https://assets.invalid'
if run_verify "$receipt" >/dev/null; then fail 'unsupported base URL accepted'; fi

printf '[verify-shared-assets-production-test] PASS\n'
