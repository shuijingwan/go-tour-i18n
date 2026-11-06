"""Thin OS/SSH adapter for the existing Go sealed release validator.

No renderer, coverage registry, or checksum allowlist is owned here. Every
consumer validates the same Go build/publish contract before external actions.
"""
import hashlib
import json
import pathlib
import shlex
import subprocess
import tempfile
import sys
import argparse
import importlib.util

ROOT = pathlib.Path(__file__).resolve().parent.parent


class ContractError(RuntimeError):
    pass


def run(command, timeout=300, capture=False):
    result = subprocess.run([str(x) for x in command], cwd=ROOT, stdin=subprocess.DEVNULL,
                            check=False, timeout=timeout,
                            stdout=subprocess.PIPE if capture else sys.stderr,
                            stderr=subprocess.PIPE if capture else None, text=True)
    if result.returncode:
        raise ContractError("release contract command failed: %s" % (result.stderr if capture else result.returncode))
    return result.stdout if capture else None


def load(release, current=True):
    command = ["go", "run", "-mod=readonly", "./cmd/tour-i18n", "release-contract", "check",
               "--input", release, "--json"]
    if current:
        command.append("--current")
    info = json.loads(run(command, capture=True))
    if info.get("result") != "PASS" or info.get("kind")!="production" or len(info.get("input_identity", "")) != 64:
        raise ContractError("no exact release input proof")
    return info


def manifest_bytes(content):
    return "".join("%s  %s\n" % (digest, path) for path, digest in sorted(content["shared_assets"].items())).encode("utf-8")


def shared_origin(content,shared):
    expected_sha=hashlib.sha256(manifest_bytes(content)).hexdigest()
    script = r'''set -Eeuo pipefail
root=$1; lock=$2; expected=$3
[[ -d $root && ! -L $root && ! -e $lock && ! -L $lock ]]
[[ $(readlink -f -- "$root") == "$root" && -f $root/SHA256SUMS && ! -L $root/SHA256SUMS ]]
[[ $(sha256sum "$root/SHA256SUMS" | awk '{print $1}') == "$expected" ]]
cd -- "$root"
[[ -z $(find . -type l -print -quit) ]]
diff -u <(awk '{print $2}' SHA256SUMS | LC_ALL=C sort) <(find . -type f ! -name SHA256SUMS -printf '%P\n' | LC_ALL=C sort)
sha256sum -c --strict SHA256SUMS >/dev/null
sha256sum SHA256SUMS
'''
    origin = shared["shared_assets_origin_root"]
    lock = str(pathlib.PurePosixPath(origin).parent / ".assets-go-dev.deploy.lock")
    result = subprocess.run(["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10",
                             shared["aliyun_ssh_alias"], "bash", "-s", "--", origin, lock,expected_sha],
                            input=script, capture_output=True, text=True, timeout=300, check=False)
    if result.returncode or result.stdout.split()[0:1] != [expected_sha]:
        raise ContractError("shared origin is not the locale's current exact dependency")
    return {"result":"PASS","manifest_sha256":expected_sha,"shared_assets_identity":content["shared_assets_identity"]}


def shared_freshness(info, shared):
    """Invocation current export/origin/public proof, including all locales.

    Immutable historical asset receipts are never a current-state oracle.
    """
    if info["content"].get("asset_origin",shared["shared_assets_public_origin"].rstrip("/"))!=shared["shared_assets_public_origin"].rstrip("/"):
        raise ContractError("consumer asset hostname differs from production target")
    with tempfile.TemporaryDirectory(prefix="go-tour-release-assets-") as temp:
        export = pathlib.Path(temp) / "export"
        run(["go", "run", "-mod=readonly", "./cmd/tour-i18n", "assets", "export", "--output", export])
        run(["go", "run", "-mod=readonly", "./cmd/tour-i18n", "assets", "validate", "--input", export])
        expected = manifest_bytes(info["content"])
        if (export / "SHA256SUMS").read_bytes() != expected:
            raise ContractError("locale artifact shared-assets dependency is stale")
        expected_sha = hashlib.sha256(expected).hexdigest()
        shared_origin(info["content"],shared)
        run([ROOT / "scripts/verify-shared-assets-public.sh", export], timeout=1800)
        return {"result": "PASS", "shared_assets_identity": info["content"]["shared_assets_identity"],
                "manifest_sha256": expected_sha, "origin": shared["shared_assets_public_origin"]}


def prior_state(profile, info):
    """Read-only exact current pointer, payload, executable and lock proof."""
    script = r'''set -Eeuo pipefail
current=$1; releases=$2; lock=$3; service=$4; mode=$5; locale=$6; origin=$7
[[ ! -e $lock && ! -L $lock ]]
if [[ ! -e $current && ! -L $current ]]; then
  [[ $mode == first-production ]]; printf '{"pointer":null,"input_identity":null,"mutation_status":"NONE"}\n'; exit 0
fi
[[ -L $current ]]
target=$(readlink -f -- "$current")
[[ $target == "$releases/"* && -d $target && ! -L $target ]]
schema=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["schema_version"])' "$target/release.json")
if [[ $schema == 3 ]]; then
  proof=$("$target/bin/tour" -verify-bundle)
  python3 -c 'import json,sys;a=json.load(sys.stdin)["content"];assert a["locale"]==sys.argv[1] and a["origin"]==sys.argv[2]' "$locale" "$origin" <<<"$proof"
elif [[ $schema == 2 ]]; then
  # Historical predecessor identity only. This does not authorize publishing or
  # rolling back to a Tour-only artifact under the new current release contract.
  [[ -z $(find "$target" -type l -print -quit) ]]
  python3 -c 'import json,sys;assert json.load(open(sys.argv[1]))["locale"]==sys.argv[2]' "$target/release.json" "$locale"
  cd -- "$target"
  awk 'NF!=2 || length($1)!=64 || $1 !~ /^[0-9a-f]+$/ || $2 ~ /(^\/|(^|\/)\.\.?(\/|$)|\\)/ {exit 1}' SHA256SUMS
  diff -u <(awk '{print $2}' SHA256SUMS | LC_ALL=C sort) <(find . -type f ! -name SHA256SUMS -printf '%P\n' | LC_ALL=C sort)
  sha256sum -c --strict SHA256SUMS >/dev/null
else exit 1; fi
pid=$(systemctl show "$service" -p MainPID); pid=${pid#MainPID=}
[[ $(systemctl is-active "$service") == active ]]
[[ $pid =~ ^[1-9][0-9]*$ && $(readlink -f -- "/proc/$pid/exe") == "$target/bin/tour" ]]
python3 - "$target" <<'PY'
import hashlib,json,pathlib,sys
p=pathlib.Path(sys.argv[1]); f=p/'_content/site/release-content.json'; a=json.loads(f.read_text()) if f.is_file() else {}
print(json.dumps({'pointer':str(p),'input_identity':hashlib.sha256((p/'SHA256SUMS').read_bytes()).hexdigest(),'content_identity':a.get('identity'),'shared_assets_identity':a.get('shared_assets_identity'),'mutation_status':'KNOWN','prior_contract':'sealed' if a else 'historical'}))
PY
'''
    result = subprocess.run(["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", profile["origin_ssh_alias"],
                             "bash", "-s", "--", profile["current"], profile["releases_root"],
                             profile["deployment_lock"], profile["systemd_service"], profile["production_state"],profile["locale"],info["origin"]],
                            input=script, text=True, capture_output=True, timeout=300, check=False)
    if result.returncode:
        raise ContractError("prior state is unknown, locked, or not a sealed Site release; stop for recovery")
    return json.loads(result.stdout)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=("preflight", "prior", "check"))
    parser.add_argument("--input", required=True)
    args = parser.parse_args()
    try:
        info = load(args.input, current=True)
        if args.action == "check": print(json.dumps(info)); return 0
        spec = importlib.util.spec_from_file_location("release_contract_identity", ROOT / "scripts/production-identity.py")
        identity_module = importlib.util.module_from_spec(spec); spec.loader.exec_module(identity_module)
        identity = identity_module.load_identity(ROOT / "production/identity.json")
        profile = next(p for p in identity["locales"] if p["locale"] == info["locale"])
        proof = {"shared_assets":shared_freshness(info, identity["shared"]),"prior":prior_state(profile,info)} if args.action == "preflight" else prior_state(profile, info)
        print(json.dumps(proof, sort_keys=True)); return 0
    except (ContractError,OSError,ValueError,StopIteration,subprocess.SubprocessError) as exc:
        print("release contract FAILED: " + str(exc),file=sys.stderr); return 1

if __name__ == "__main__": raise SystemExit(main())
