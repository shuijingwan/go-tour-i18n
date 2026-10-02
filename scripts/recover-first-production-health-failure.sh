#!/usr/bin/env bash

# Recover only the preserved state left by deploy-production.sh exit 22.  This
# is deliberately not a general lock remover or deployment retry mechanism.
set -Eeuo pipefail
IFS=$'\n\t'

script_dir=$(cd -P -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
# shellcheck source=production-identity.sh
source "$script_dir/production-identity.sh"
unset script_dir

error() { printf '[first-production-recovery] ERROR: %s\n' "$*" >&2; }
usage() { printf 'usage: %s <failed-first-production-release-dir>\n' "${0##*/}" >&2; }

main() {
    local release_input release receipt locale remote_name expected_remote
    local ssh_host releases current lock service health local_manifest_sha remote_result
    (( $# == 1 )) || { usage; return 2; }
    release_input=$1
    for command_name in basename python3 readlink sha256sum ssh; do command -v "$command_name" >/dev/null || { error "missing command: $command_name"; return 1; }; done
    [[ -d $release_input && ! -L $release_input ]] || { error 'failed release must be a real local directory'; return 1; }
    release=$(readlink -f -- "$release_input")
    remote_name=${release##*/}; remote_name=${remote_name#go-tour-release-}
    [[ ${release##*/} == go-tour-release-* && $remote_name =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]] || { error 'unsafe failed release name'; return 1; }
    [[ -f $release/SHA256SUMS && ! -L $release/SHA256SUMS ]] || { error 'failed release has no regular SHA256SUMS'; return 1; }
    local_manifest_sha=$(sha256sum "$release/SHA256SUMS"); local_manifest_sha=${local_manifest_sha%% *}
    receipt="${release}.first-production-receipt.json"
    [[ -f $receipt && ! -L $receipt ]] || { error 'failed release has no regular first-production receipt'; return 1; }
    locale=$(python3 - "$release/release.json" "$receipt" <<'PY'
import json,os,sys
try:
    release=json.load(open(sys.argv[1],encoding='utf-8'))
    receipt=json.load(open(sys.argv[2],encoding='utf-8'))
except (OSError,json.JSONDecodeError) as exc: raise SystemExit(exc)
locale=release.get('locale')
stages=receipt.get('stages')
failure=receipt.get('failure')
if not isinstance(locale,str) or not locale: raise SystemExit('release locale is invalid')
expected_release=os.path.basename(os.path.abspath(os.path.dirname(sys.argv[1]))).removeprefix('go-tour-release-')
if not (receipt.get('schema') == 'go-tour-i18n/first-production-receipt/v1' and receipt.get('locale') == locale and receipt.get('release') == expected_release and receipt.get('result') == 'failed'):
    raise SystemExit('receipt identity/result is not a failed first-production receipt')
if not isinstance(failure,dict) or failure.get('stage') != 'deploy': raise SystemExit('receipt is not an explicit deploy failure')
if not isinstance(stages,dict) or any(stages.get(stage,{}).get('result') != 'PASS' for stage in ('preflight','infrastructure','playground-origin')):
    raise SystemExit('receipt does not prove the required pre-deploy stages passed')
if any(stage in stages for stage in ('cloudflare-dns','public-machine','browser')):
    raise SystemExit('receipt records post-deploy stages and is not recoverable as first-deployment health failure')
print(locale)
PY
) || { error 'failed release/receipt is not the formal recoverable FIRST_DEPLOYMENT evidence'; return 1; }
    load_production_identity_locale "$locale" || { error 'formal production identity is invalid'; return 1; }
    [[ $PRODUCTION_STATE == first-production ]] || { error "production_state must be first-production, got $PRODUCTION_STATE"; return 1; }
    ssh_host=$PRODUCTION_ORIGIN_SSH_ALIAS; releases=$PRODUCTION_RELEASES_ROOT; current=$PRODUCTION_CURRENT
    lock=$PRODUCTION_DEPLOYMENT_LOCK; service=$PRODUCTION_SYSTEMD_SERVICE; health=$PRODUCTION_LOCALHOST_HEALTH_URL
    expected_remote="$releases/$remote_name"
    remote_result=$(ssh -o BatchMode=yes -o ConnectTimeout=10 "$ssh_host" bash -s -- \
        "$releases" "$current" "$lock" "$service" "$health" "$expected_remote" "$local_manifest_sha" <<'REMOTE'
set -Eeuo pipefail
releases=$1; current=$2; lock=$3; service=$4; health=$5; failed=$6; expected_manifest_sha=$7
fail() { printf '[first-production-recovery:remote] ERROR: %s\n' "$*" >&2; exit 1; }
[[ $(id -u) == 0 ]] || fail 'remote SSH user must be root'
[[ -d $releases && ! -L $releases && -d $failed && ! -L $failed ]] || fail 'release roots are not exact real directories'
[[ -L $current && $(readlink -f -- "$current") == "$failed" ]] || fail 'current does not point exactly to the receipt failed release'
[[ -d $lock && ! -L $lock ]] || fail 'deployment lock is not the expected directory'
healthy=1
for health_attempt in 1 2 3; do
  state=$(systemctl is-active "$service" 2>/dev/null || true)
  code=$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time 5 "$health" || true)
  if [[ $state != active || $code != 200 ]]; then healthy=0; break; fi
  [[ $health_attempt == 3 ]] || sleep 1
done
if (( healthy )); then
  [[ -f $failed/SHA256SUMS && ! -L $failed/SHA256SUMS ]] || fail 'healthy recovered release has no regular SHA256SUMS'
  actual_manifest_sha=$(sha256sum "$failed/SHA256SUMS"); actual_manifest_sha=${actual_manifest_sha%% *}
  [[ $actual_manifest_sha == "$expected_manifest_sha" ]] || fail 'healthy recovered release manifest differs from the local failed release'
  (cd -- "$failed" && sha256sum -c --strict SHA256SUMS >/dev/null) || fail 'healthy recovered release checksum verification failed'
  [[ -z $(find "$failed" -type l -print -quit) ]] || fail 'healthy recovered release contains a symlink'
  [[ -z $(find "$failed" ! -type d ! -type f -print -quit) ]] || fail 'healthy recovered release contains an unsupported entry'
  main_pid=$(systemctl show "$service" -p MainPID); main_pid=${main_pid#MainPID=}
  [[ $main_pid =~ ^[1-9][0-9]*$ ]] || fail 'healthy recovered service has no valid MainPID'
  [[ $(readlink -f -- "/proc/$main_pid/exe") == "$failed/bin/tour" ]] || fail 'healthy recovered service does not execute the receipt release binary'
  rmdir -- "$lock" || fail 'could not remove the verified empty deployment lock after late health recovery'
  printf '[first-production-recovery:remote] RECOVERED_FIRST_DEPLOYMENT_LATE_HEALTH=%s\n' "$failed"
  exit 0
fi
recovery="$current.recovery-$$"
[[ ! -e $recovery && ! -L $recovery ]] || fail 'recovery path already exists'
restore() {
  if [[ -L $recovery && ! -e $current && ! -L $current ]]; then mv -Tf -- "$recovery" "$current" || true; fi
}
trap 'restore; exit 1' INT TERM HUP
mv -T -- "$current" "$recovery"
if ! rmdir -- "$lock"; then restore; fail 'could not remove the verified empty deployment lock; current restored'; fi
rm -f -- "$recovery"
trap - INT TERM HUP
printf '[first-production-recovery:remote] RECOVERED_FIRST_DEPLOYMENT_HEALTH_FAILURE=%s\n' "$failed"
REMOTE
    ) || { error 'remote recovery validation failed'; return 1; }
    printf '%s\n' "$remote_result"
    case $remote_result in
        *RECOVERED_FIRST_DEPLOYMENT_LATE_HEALTH=*)
            python3 - "$receipt" <<'PY'
import datetime,json,os,pathlib,sys
path=pathlib.Path(sys.argv[1])
receipt=json.loads(path.read_text(encoding='utf-8'))
if receipt.get('schema') != 'go-tour-i18n/first-production-receipt/v1' or receipt.get('result') != 'failed':
    raise SystemExit('receipt changed before late-health recovery could be recorded')
failure=receipt.get('failure')
stages=receipt.get('stages')
if not isinstance(failure,dict) or failure.get('stage') != 'deploy' or not isinstance(stages,dict):
    raise SystemExit('receipt no longer records the expected deploy failure')
stages['deploy']={
    'completed_at': datetime.datetime.now(datetime.timezone.utc).replace(microsecond=0).isoformat().replace('+00:00','Z'),
    'result': 'PASS',
}
temporary=path.with_name(path.name+'.tmp-%d' % os.getpid())
temporary.write_text(json.dumps(receipt,ensure_ascii=False,sort_keys=True,separators=(',',':'))+'\n',encoding='utf-8')
os.chmod(temporary,0o644)
os.replace(temporary,path)
PY
            printf '[first-production-recovery] PASS: receipt release recovered to stable health; deploy PASS recorded without replacing the immutable release.\n'
            printf 'next: scripts/first-production.sh %s\n' "$release"
            ;;
        *RECOVERED_FIRST_DEPLOYMENT_HEALTH_FAILURE=*)
            printf '[first-production-recovery] PASS: failed release preserved; publish a new immutable release, then run first-production on that new release.\n'
            ;;
        *)
            error 'remote recovery returned no recognized completion identity'
            return 1
            ;;
    esac
}

main "$@"
