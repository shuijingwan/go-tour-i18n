#!/usr/bin/env python3
"""Origin/public adapters for the single sealed Site release contract."""
import argparse
import importlib.util
import pathlib
import sys
ROOT=pathlib.Path(__file__).resolve().parent.parent
def load(name,filename):
    spec=importlib.util.spec_from_file_location(name,ROOT/"scripts"/filename)
    value=importlib.util.module_from_spec(spec);spec.loader.exec_module(value);return value
CONTRACT=load("production_verifier_contract","release_contract.py")
HTTP=load("production_verifier_http","release_http.py")
IDENTITY=load("production_verifier_identity","production-identity.py")

def verify(release,direct=False):
    info=CONTRACT.load(release,current=True)
    identity=IDENTITY.load_identity(ROOT/"production/identity.json")
    profile=next(p for p in identity["locales"] if p["locale"]==info["locale"])
    CONTRACT.shared_freshness(info,identity["shared"])
    prior=CONTRACT.prior_state(profile,info)
    if prior.get("input_identity")!=info["input_identity"]:
        raise CONTRACT.ContractError("origin current pointer/payload/PID differs from exact deployment input")
    # Origin HTTP uses the same derived model on the formal origin runner.
    model=HTTP.model(info,release)
    code=pathlib.Path(HTTP.__file__).read_text()
    import json,shlex,subprocess
    payload={"model":model,"base":profile["localhost_health_url"].rstrip("/"),"cache_header":None,"origin_ip":None}
    # The runner entry accepts a base only for origin HTTP; canonical remains
    # the formal HTTPS identity in the model.
    result=subprocess.run(["ssh","-o","BatchMode=yes","-o","ConnectTimeout=10",profile["origin_ssh_alias"],
                           "python3 -c "+shlex.quote(code)+" --remote"],input=json.dumps(payload),text=True,check=False,timeout=1800)
    if result.returncode:raise CONTRACT.ContractError("origin HTTP contract failed")
    HTTP.public(info,release,identity["shared"],profile,direct=direct)
    print("PRODUCTION MACHINE ACCEPTANCE: PASS (current exact origin/public Site contract)")

def main():
    parser=argparse.ArgumentParser();parser.add_argument("release");parser.add_argument("--direct-origin",action="store_true")
    args=parser.parse_args()
    try:verify(pathlib.Path(args.release),args.direct_origin);return 0
    except (RuntimeError,OSError,ValueError,StopIteration) as exc:
        print("[verify-production] FAILED: "+str(exc),file=sys.stderr);return 1
if __name__=="__main__":raise SystemExit(main())
