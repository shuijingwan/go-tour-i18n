#!/usr/bin/env python3
"""Fail-closed per-locale orchestration using the single release contract."""
from __future__ import annotations
import importlib.util
import json
import pathlib
import subprocess
import sys

ROOT=pathlib.Path(__file__).resolve().parent.parent
SPEC=importlib.util.spec_from_file_location("maintenance_production_batch_core",ROOT/"scripts/maintenance-production.py")
CORE=importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(CORE)

class BatchError(RuntimeError): pass

class Batch:
    def __init__(self,release_dirs):
        if not release_dirs: raise BatchError("at least one release directory required")
        self.items=[CORE.Orchestrator(pathlib.Path(p)) for p in release_dirs]
        if len({i.locale for i in self.items})!=len(self.items): raise BatchError("duplicate locale")
        self.complete_at_start=[i.receipt["result"]=="passed" for i in self.items]
        self.retry_at_start=[i.receipt["result"]=="failed" for i in self.items]
        self.rows=[{"locale":i.locale,"PREFLIGHT":"PENDING","INPUT_IDENTITY":i.info["input_identity"],
                    "PRIOR_STATE":None,"ACTION":"PENDING","RESULT":"PENDING","VERIFIER":{},
                    "MUTATION_STATUS":"NONE","evidence":str(i.result_path)} for i in self.items]

    @staticmethod
    def run(command,timeout,progress=None):
        result=subprocess.run([str(x) for x in command],stdin=subprocess.DEVNULL,check=False,timeout=timeout)
        if result.returncode: raise BatchError("command exit %d"%result.returncode)

    def preflight(self):
        shared={}
        for index,item in enumerate(self.items):
            key=item.info["content"]["shared_assets_identity"]
            proof=item.preflight(shared.get(key)); shared[key]=proof
            self.rows[index]["PREFLIGHT"]="PASS"; self.rows[index]["PRIOR_STATE"]=item.current_prior

    def refresh_row(self,index):
        item=self.items[index]; r=item.receipt
        for key in ("INPUT_IDENTITY","ACTION","VERIFIER","MUTATION_STATUS"):
            self.rows[index][key]=r.get(key)
        self.rows[index]["PRIOR_STATE"]=getattr(item,"current_prior",r.get("PRIOR_STATE"))
        self.rows[index]["PREFLIGHT"]="PASS" if item.preflight_complete else "FAILED"
        return r

    def block_remaining(self,start,result):
        for index in range(start,len(self.items)):
            self.rows[index]["RESULT"]=result

    def fail_item(self,index,phase,exc):
        self.items[index].fail(exc); self.refresh_row(index)
        self.rows[index]["RESULT"]="UNKNOWN" if self.items[index].receipt["MUTATION_STATUS"]=="UNKNOWN" else "FAILED"
        return BatchError("locale=%s phase=%s evidence=%s: %s"%(self.items[index].locale,phase,self.items[index].receipt_path,exc))

    def execute(self):
        try: self.preflight()
        except (CORE.CONTRACT.ContractError,CORE.MaintenanceProductionError,OSError,ValueError) as exc:
            for index,item in enumerate(self.items):
                self.refresh_row(index)
                self.rows[index]["RESULT"]="BLOCKED" if item.preflight_complete else "PREFLIGHT_FAILED"
            self.print_summary(); raise BatchError(str(exc)) from exc
        for index,item in enumerate(self.items):
            item.set_progress_context("locale",index+1,len(self.items))
            try:
                if self.complete_at_start[index]:
                    item.verify_current(); self.rows[index]["RESULT"]="CARRIED"
                else:
                    item.begin(); item.execute_mutation(); item.execute_acceptance()
                    if item.receipt["result"]!="passed": raise BatchError("no complete immutable per-locale result")
                    self.rows[index]["RESULT"]="RETRIED" if self.retry_at_start[index] else "SUCCEEDED"
                self.refresh_row(index)
            except (CORE.MaintenanceProductionError,CORE.CONTRACT.ContractError,OSError,ValueError,BatchError) as exc:
                failed=self.fail_item(index,"locale",exc); self.block_remaining(index+1,"BLOCKED")
                self.print_summary(); raise failed
        if any(row["RESULT"] not in ("CARRIED","SUCCEEDED","RETRIED") for row in self.rows):
            raise BatchError("partial batch cannot pass")
        self.print_summary()

    def print_summary(self):
        print(json.dumps({"locales":self.rows},ensure_ascii=False,sort_keys=True))
        for result in ("CARRIED","SUCCEEDED","RETRIED","FAILED","UNKNOWN","BLOCKED","PREFLIGHT_FAILED"):
            print("%s=%d"%(result,sum(row["RESULT"]==result for row in self.rows)))
        complete=all(row["RESULT"] in ("CARRIED","SUCCEEDED","RETRIED") for row in self.rows)
        print("MAINTENANCE BATCH: "+("PASS" if complete else "FAILED"))

def main():
    try: Batch(sys.argv[1:]).execute(); return 0
    except (BatchError,CORE.MaintenanceProductionError,CORE.IDENTITY.IdentityError,CORE.CONTRACT.ContractError,OSError,ValueError) as exc:
        print("[maintenance-production-batch] FAILED: "+str(exc),file=sys.stderr); return 1
if __name__=="__main__": raise SystemExit(main())
