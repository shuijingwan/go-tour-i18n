#!/usr/bin/env python3
"""Existing single-locale release execution with exact per-locale evidence."""
from __future__ import annotations
import datetime as dt
import importlib.util
import json
import os
import pathlib
import re
import subprocess
import sys
import time

ROOT = pathlib.Path(__file__).resolve().parent.parent
def module(name, filename):
    spec = importlib.util.spec_from_file_location(name, ROOT / "scripts" / filename)
    value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value); return value
IDENTITY = module("production_identity_maintenance", "production-identity.py")
CONTRACT = module("release_contract_maintenance", "release_contract.py")
RECEIPT_SCHEMA = "go-learning/maintenance-execution-state/v2"
STAGE_LABELS = {"deploy": "deployment", "purge": "CDN purge", "machine": "machine acceptance", "browser": "browser acceptance"}
LEGACY_STAGE = "visual"

class MaintenanceProductionError(RuntimeError):
    def __init__(self, stage, expected, actual, next_step):
        super().__init__(actual); self.stage=stage; self.expected=expected; self.actual=actual; self.next_step=next_step

def utc_now():
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")

def release_locale(release_dir):
    data=json.loads((release_dir/"release.json").read_text())
    if not isinstance(data.get("locale"),str) or not data["locale"]:
        raise MaintenanceProductionError("preflight","release locale",repr(data),"检查sealed bundle")
    return data["locale"]

def safe_release_name(release_dir):
    name=release_dir.name
    if not name.startswith("go-tour-release-") or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*",name[len("go-tour-release-"):]):
        raise MaintenanceProductionError("preflight","safe release name",name,"使用正式publish目录")
    return name[len("go-tour-release-"):]

class Orchestrator:
    def __init__(self, release_dir):
        if not release_dir.is_dir() or release_dir.is_symlink():
            raise MaintenanceProductionError("preflight","real release directory",str(release_dir),"重新检查bundle")
        self.release_dir=release_dir.resolve(); self.release_name=safe_release_name(self.release_dir)
        self.info=CONTRACT.load(self.release_dir, current=True)
        self.locale=self.info["locale"]; self.identity=IDENTITY.load_identity(ROOT/"production/identity.json")
        matches=[p for p in self.identity["locales"] if p["locale"]==self.locale]
        if len(matches)!=1 or matches[0]["production_state"]!="live":
            raise MaintenanceProductionError("preflight","unique live profile",self.locale,"first-production locale使用首次入口")
        self.profile=matches[0]
        if self.info["origin"]!=self.profile["production_public_url"].rstrip("/"):
            raise MaintenanceProductionError("preflight","exact production origin",self.info["origin"],"不要复用不同target的bundle")
        self.receipt_path=pathlib.Path(str(self.release_dir)+".maintenance-production-state.json")
        self.result_path=pathlib.Path(str(self.release_dir)+".maintenance-production-result-"+self.info["input_identity"]+".json")
        self.progress_context=None; self.historical_pass=False; self.preflight_complete=False
        self.receipt=self._load_or_new_receipt()

    def set_progress_context(self,phase,ordinal,total): self.progress_context=(phase,ordinal,total)
    def progress(self,stage,result,elapsed=None,detail=None):
        text="[maintenance-production] locale=%s stage=%s %s"%(self.locale,stage,result)
        if elapsed is not None: text+=" elapsed=%.1fs"%elapsed
        if detail: text+=" "+detail
        print(text,flush=True)

    def _new_receipt(self):
        return {"schema":RECEIPT_SCHEMA,"locale":self.locale,"hostname":self.profile["production_hostname"],
                "cdn":self.profile["cdn"],"public_url":self.profile["production_public_url"],"release":self.release_name,
                "started_at":utc_now(),"completed_at":None,"result":"running","stages":{},
                "PREFLIGHT":"PENDING","INPUT_IDENTITY":self.info["input_identity"],"CONTENT_IDENTITY":self.info["content"]["identity"],
                "SHARED_ASSETS_IDENTITY":self.info["content"]["shared_assets_identity"],
                "PRIOR_STATE":None,"ACTION":"PENDING","RESULT":"PENDING","VERIFIER":{},
                "MUTATION_STATUS":"NONE"}

    def _load_or_new_receipt(self):
        base=self._new_receipt()
        if not self.receipt_path.exists(): return base
        if self.receipt_path.is_symlink(): raise CONTRACT.ContractError("state cannot be a symlink")
        previous=json.loads(self.receipt_path.read_text())
        for key in ("schema","locale","hostname","cdn","public_url","release","INPUT_IDENTITY","CONTENT_IDENTITY","SHARED_ASSETS_IDENTITY"):
            if previous.get(key)!=base[key]: raise CONTRACT.ContractError("stale execution state: "+key)
        stages=previous.get("stages")
        if not isinstance(stages,dict) or set(stages)-set(STAGE_LABELS): raise CONTRACT.ContractError("invalid execution stages")
        if previous.get("MUTATION_STATUS") in ("ATTEMPTED","UNKNOWN"): raise CONTRACT.ContractError("mutation unknown: preserve evidence; never replay")
        if previous.get("cdn_attempt_started") and stages.get("purge",{}).get("result")!="PASS":
            raise CONTRACT.ContractError("CDN attempt has no durable successful result; never replay")
        if pathlib.Path(str(self.receipt_path)+".cdn-attempt").exists() and stages.get("purge",{}).get("result")!="PASS":
            raise CONTRACT.ContractError("exclusive CDN attempt has no successful result; never replay")
        for stage,value in stages.items():
            if value.get("result")!="PASS" or value.get("input_identity")!=base["INPUT_IDENTITY"]:
                raise CONTRACT.ContractError("stage not bound to current exact input")
        prefix=[s for s in STAGE_LABELS if s in stages]
        if prefix!=list(STAGE_LABELS)[:len(prefix)]: raise CONTRACT.ContractError("stage order mismatch")
        if previous.get("result")=="passed":
            if set(stages)!=set(STAGE_LABELS) or not self.result_path.is_file() or self.result_path.is_symlink():
                raise CONTRACT.ContractError("missing immutable result proof")
            result=json.loads(self.result_path.read_text())
            if result!=previous: raise CONTRACT.ContractError("immutable result/state mismatch")
        elif previous.get("result") not in ("running","failed"): raise CONTRACT.ContractError("unknown result")
        return previous

    def write_receipt(self,result=None):
        if result is not None:
            self.receipt["result"]=result
            self.receipt["RESULT"]={"passed":"PASS","failed":"FAILED","running":"PENDING"}[result]
            if result in ("passed","failed"): self.receipt["completed_at"]=utc_now()
        data=json.dumps(self.receipt,ensure_ascii=False,sort_keys=True,separators=(",",":"))+"\n"
        temporary=self.receipt_path.with_name(self.receipt_path.name+".tmp-%d"%os.getpid())
        temporary.write_text(data); os.chmod(temporary,0o644); os.replace(temporary,self.receipt_path)
        if result=="passed":
            try:
                with self.result_path.open("x") as output: output.write(data)
            except FileExistsError:
                if self.result_path.read_text()!=data: raise CONTRACT.ContractError("refusing to overwrite immutable result")

    def stage_passed(self,stage): return self.receipt["stages"].get(stage,{}).get("result")=="PASS"
    def record(self,stage):
        # The CDN child persists its attempt before POST. Keep that durable
        # boundary when the parent records success instead of replacing it
        # with an older in-memory state.
        if stage=="purge":
            current=json.loads(self.receipt_path.read_text())
            if current.get("INPUT_IDENTITY")!=self.info["input_identity"]:
                raise CONTRACT.ContractError("execution state changed during purge")
            self.receipt["cdn_attempt_started"]=current.get("cdn_attempt_started",False)
        self.receipt["stages"][stage]={"result":"PASS","completed_at":utc_now(),"input_identity":self.info["input_identity"]}
        self.receipt["MUTATION_STATUS"]="KNOWN"
        if stage in ("machine","browser"): self.receipt["VERIFIER"][stage]="PASS"
        self.write_receipt()

    def run_command(self,stage,command,timeout):
        self.progress(stage,"START"); started=time.monotonic()
        mutating=stage in ("deploy","purge")
        if mutating:
            self.receipt["MUTATION_STATUS"]="ATTEMPTED"; self.receipt["ACTION"]=stage.upper(); self.write_receipt()
        try:
            result=subprocess.run([str(x) for x in command],cwd=ROOT,stdin=subprocess.DEVNULL,check=False,
                                  timeout=timeout,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True)
        except OSError as exc:
            if mutating: self.receipt["MUTATION_STATUS"]="KNOWN" if self.stage_passed("deploy") else "NONE"; self.write_receipt()
            raise MaintenanceProductionError(stage,"command started",str(exc),"failed-before-mutation可重试") from exc
        except subprocess.TimeoutExpired as exc:
            if mutating: self.receipt["MUTATION_STATUS"]="UNKNOWN"; self.write_receipt()
            raise MaintenanceProductionError(stage,"known mutation outcome",str(exc),"停止，保留现场，先只读recovery") from exc
        output=result.stdout or ""; print(output,end="",flush=True)
        if result.returncode:
            if mutating:
                self.receipt["MUTATION_STATUS"]=("KNOWN" if self.stage_passed("deploy") else "NONE") if "MUTATION_STATUS=NONE" in output else "UNKNOWN"
                self.write_receipt()
            raise MaintenanceProductionError(stage,"command exit 0","exit %d"%result.returncode,"检查exact stage evidence后恢复")
        self.progress(stage,"PASS",time.monotonic()-started)

    def preflight(self,shared_proof=None):
        current=CONTRACT.load(self.release_dir,current=True)
        if current["input_identity"]!=self.info["input_identity"]: raise CONTRACT.ContractError("input changed after selection")
        proof=shared_proof or CONTRACT.shared_freshness(self.info,self.identity["shared"])
        if proof.get("shared_assets_identity")!=self.info["content"]["shared_assets_identity"]: raise CONTRACT.ContractError("shared dependency mismatch")
        prior=CONTRACT.prior_state(self.profile,self.info)
        if self.stage_passed("deploy") and prior.get("input_identity")!=self.info["input_identity"]:
            raise CONTRACT.ContractError("passed deployment is no longer the exact current runtime")
        self.run_command("preflight",[ROOT/"scripts/production-cdn.py","preflight","--locale",self.locale],300)
        self.current_prior=prior; self.shared_proof=proof; self.preflight_complete=True
        if self.receipt["result"]!="passed":
            self.receipt["PREFLIGHT"]="PASS"
            if self.receipt["PRIOR_STATE"] is None: self.receipt["PRIOR_STATE"]=prior
            self.receipt["CURRENT_STATE"]=prior; self.receipt["SHARED_ASSETS_PROOF"]=proof
            self.write_receipt()
        return proof

    def begin(self):
        if self.receipt["result"]=="passed": return False
        if not self.preflight_complete: raise CONTRACT.ContractError("current preflight required before execution")
        self.receipt.pop("failure",None); self.receipt["result"]="running"; self.receipt["RESULT"]="PENDING"; self.write_receipt(); return True

    def fail(self,exc):
        if self.receipt["result"]=="passed": return
        self.receipt["failure"]={"stage":getattr(exc,"stage","contract"),"result":"FAILED","evidence":str(exc)}
        self.write_receipt("failed")

    def execute_mutation(self,revalidate_passed_deploy=False):
        if not self.preflight_complete: raise CONTRACT.ContractError("no current preflight")
        if self.receipt["result"]=="passed": return
        if not self.stage_passed("deploy"):
            self.run_command("deploy",[ROOT/"scripts/deploy-production.sh",self.release_dir],1800); self.record("deploy")
        if not self.stage_passed("purge"):
            self.run_command("purge",[ROOT/"scripts/production-cdn.py","purge","--locale",self.locale,"--execution-state",self.receipt_path],300); self.record("purge")

    def verify_current(self):
        self.run_command("machine",[ROOT/"scripts/verify-production.sh",self.release_dir],1800)
        self.run_command("browser",[ROOT/"scripts/verify-production-browser.py",self.profile["production_public_url"],self.locale,"--release-dir",self.release_dir],1800)

    def execute_acceptance(self):
        if not all(self.stage_passed(s) for s in ("deploy","purge")): raise CONTRACT.ContractError("mutation stages incomplete")
        for stage,command,timeout in (
            ("machine",[ROOT/"scripts/verify-production.sh",self.release_dir],1800),
            ("browser",[ROOT/"scripts/verify-production-browser.py",self.profile["production_public_url"],self.locale,"--release-dir",self.release_dir],1800)):
            if not self.stage_passed(stage): self.run_command(stage,command,timeout); self.record(stage)
        self.receipt["ACTION"]="COMPLETED"; self.receipt["MUTATION_STATUS"]="KNOWN"; self.write_receipt("passed")

    def execute(self):
        self.preflight()
        if self.receipt["result"]=="passed":
            self.verify_current(); self.print_summary("CARRIED"); return
        self.begin(); self.execute_mutation(); self.execute_acceptance(); self.print_summary("PASS")

    def print_summary(self,result="PASS"):
        print("MAINTENANCE PRODUCTION: "+result)
        print("locale=%s input_identity=%s result=%s mutation_status=%s evidence=%s"%(self.locale,self.info["input_identity"],result,self.receipt["MUTATION_STATUS"],self.result_path))

def main():
    if len(sys.argv)!=2: print("usage: maintenance-production.sh <release-dir>",file=sys.stderr); return 2
    workflow=None
    try:
        workflow=Orchestrator(pathlib.Path(sys.argv[1])); workflow.execute(); return 0
    except (MaintenanceProductionError,IDENTITY.IdentityError,CONTRACT.ContractError,OSError,ValueError,subprocess.SubprocessError) as exc:
        if workflow: workflow.fail(exc)
        print("[maintenance-production] FAILED: "+str(exc),file=sys.stderr); return 1
if __name__=="__main__": raise SystemExit(main())
