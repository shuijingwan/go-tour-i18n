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
}

assert_profile zh-CN /data/go-tour/releases /data/go-tour/current \
    /data/go-tour/.deploy.lock go-tour.service http://127.0.0.1:3999/ \
    https://go-dev.shuijingwanwq.com/
assert_profile ja-JP /data/go-tour-ja-JP/releases /data/go-tour-ja-JP/current \
    /data/go-tour-ja-JP/.deploy.lock go-tour-ja-JP.service http://127.0.0.1:4000/ \
    https://ja-go-dev.shuijingwanwq.com/

if select_deployment_profile fr-FR 2>/dev/null; then
    fail 'unsupported locale was accepted'
fi

fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT

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
remote_called=0
prepare_remote() { remote_called=1; return 1; }
upload_release() { remote_called=1; return 1; }
if main "$fake_unsupported" >/dev/null 2>&1; then
    fail 'unsupported bundle locale was accepted'
fi
(( remote_called == 0 )) || fail 'unsupported locale reached a remote operation'

[[ $RELEASE_LOCALE == fr-FR ]] || fail 'bundle locale was not read from release.json'

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

printf '[deploy-test] PASS\n'
