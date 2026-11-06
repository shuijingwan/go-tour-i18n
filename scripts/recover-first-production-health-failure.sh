#!/usr/bin/env bash
# Narrow exact late-health recovery. No partial rollback or pointer removal.
set -Eeuo pipefail
IFS=$'\n\t'
script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
source "$script_dir/production-identity.sh"
error() { printf '[first-production-recovery] ERROR: %s\n' "$*" >&2; }
main() {
    (( $# == 1 )) || { error 'usage: recover-first-production-health-failure.sh <sealed-release-dir>'; return 2; }
    local release info input_id locale receipt manifest_sha shared_sha remote_name proof
    [[ -d $1 && ! -L $1 ]] || return 1
    release=$(readlink -f -- "$1"); receipt="$release.first-production-state.json"
    info=$(python3 "$script_dir/release_contract.py" check --input "$release") || return 1
    input_id=$(python3 -c 'import json,sys;print(json.load(sys.stdin)["input_identity"])' <<<"$info")
    locale=$(python3 -c 'import json,sys;print(json.load(sys.stdin)["locale"])' <<<"$info")
    shared_sha=$(python3 -c 'import hashlib,json,sys;a=json.load(sys.stdin)["content"]["shared_assets"];print(hashlib.sha256("".join("%s  %s\n"%(h,p) for p,h in sorted(a.items())).encode()).hexdigest())' <<<"$info")
    [[ -f $receipt && ! -L $receipt ]] || { error 'current failed execution state required; historical receipt is immutable'; return 1; }
    python3 - "$receipt" "$locale" "$input_id" <<'PY_STATE'
import json,sys
s=json.load(open(sys.argv[1])); stages=s.get('stages',{})
assert s.get('schema')=='go-learning/first-production-execution/v2'
assert s.get('locale')==sys.argv[2] and s.get('INPUT_IDENTITY')==sys.argv[3]
assert s.get('result')=='failed' and s.get('failure',{}).get('stage')=='deploy'
assert s.get('MUTATION_STATUS')=='UNKNOWN'
assert all(stages.get(k,{}).get('result')=='PASS' and stages[k].get('input_identity')==sys.argv[3] for k in ('preflight','infrastructure','playground-origin'))
assert not any(k in stages for k in ('deploy','cloudflare-dns','public-machine','browser'))
PY_STATE
    load_production_identity_locale "$locale"; load_production_identity_shared
    [[ $PRODUCTION_STATE == first-production && $PRODUCTION_ORIGIN_SSH_ALIAS == "$PRODUCTION_ALIYUN_SSH_ALIAS" ]] || return 1
    remote_name=${release##*/};remote_name=${remote_name#go-tour-release-}
    [[ $remote_name =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]] || return 1
    proof=$(ssh -o BatchMode=yes -o ConnectTimeout=10 "$PRODUCTION_ORIGIN_SSH_ALIAS" bash -s -- \
        "$PRODUCTION_RELEASES_ROOT/$remote_name" "$PRODUCTION_CURRENT" "$PRODUCTION_DEPLOYMENT_LOCK" \
        "$PRODUCTION_SYSTEMD_SERVICE" "$PRODUCTION_LOCALHOST_HEALTH_URL" "$input_id" \
        "$PRODUCTION_SHARED_ASSETS_ORIGIN_ROOT" "$shared_sha" <<'REMOTE_RECOVERY'
set -Eeuo pipefail
failed=$1;current=$2;lock=$3;service=$4;health=$5;expected=$6;assets=$7;expected_shared=$8
assets_lock="${assets%/*}/.assets-go-dev.deploy.lock"
[[ $(id -u) == 0 && -L $current && $(readlink -f "$current") == "$failed" ]]
[[ -d $lock && ! -L $lock && -d $assets_lock && ! -L $assets_lock ]]
[[ -f $lock/consumer-token && ! -L $lock/consumer-token && -f $assets_lock/locale-consumer && ! -L $assets_lock/locale-consumer ]]
[[ $(cat "$lock/consumer-token") == "$(cat "$assets_lock/locale-consumer")" ]]
proof=$("$failed/bin/tour" -verify-bundle)
[[ $(python3 -c 'import json,sys;print(json.load(sys.stdin)["input_identity"])' <<<"$proof") == "$expected" ]]
[[ $(sha256sum "$assets/SHA256SUMS" | awk '{print $1}') == "$expected_shared" ]]
(cd -- "$assets" && sha256sum -c --strict SHA256SUMS >/dev/null)
route=$(python3 -c 'import json,sys;print(json.load(sys.stdin)["content"]["canonical_routes"][0])' <<<"$proof")
health="${health%/}$route"
for attempt in 1 2 3; do
  [[ $(systemctl is-active "$service") == active ]]
  [[ $(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time 5 "$health") == 200 ]]
  [[ $attempt == 3 ]] || sleep 1
done
pid=$(systemctl show "$service" -p MainPID);pid=${pid#MainPID=}
[[ $pid =~ ^[1-9][0-9]*$ && $(readlink -f "/proc/$pid/exe") == "$failed/bin/tour" ]]
# Unlock only this proven lease. An unhealthy/changed/partial state stops above.
rm -- "$lock/consumer-token" "$assets_lock/locale-consumer"
rmdir -- "$assets_lock" "$lock"
printf 'RECOVERED_FIRST_DEPLOYMENT_LATE_HEALTH=%s\n' "$expected"
REMOTE_RECOVERY
    ) || { error 'exact late-health proof failed; keep current, locks and all evidence; no automatic rollback'; return 1; }
    [[ $proof == "RECOVERED_FIRST_DEPLOYMENT_LATE_HEALTH=$input_id" ]] || return 1
    python3 - "$receipt" "$input_id" <<'PY_KNOWN'
import datetime,json,os,pathlib,sys
p=pathlib.Path(sys.argv[1]);s=json.loads(p.read_text())
assert s['INPUT_IDENTITY']==sys.argv[2] and s['MUTATION_STATUS']=='UNKNOWN'
s['stages']['deploy']={'result':'PASS','input_identity':sys.argv[2],'completed_at':datetime.datetime.now(datetime.timezone.utc).isoformat()}
s['MUTATION_STATUS']='KNOWN';s['RECOVERY']='EXACT_LATE_HEALTH'
tmp=p.with_name(p.name+'.recovery-tmp-%d'%os.getpid());tmp.write_text(json.dumps(s,sort_keys=True,separators=(',',':'))+'\n');os.replace(tmp,p)
PY_KNOWN
    printf '[first-production-recovery] PASS: exact bundle/assets/PID proved; resume same first-production input.\n'
}
main "$@"
