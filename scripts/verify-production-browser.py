#!/usr/bin/env python3
"""Production Chrome adapter using the carried Preview plan and checks."""
import argparse
import importlib.util
import pathlib
import sys
ROOT=pathlib.Path(__file__).resolve().parent.parent
def load(name,path):
    spec=importlib.util.spec_from_file_location(name,path)
    value=importlib.util.module_from_spec(spec); spec.loader.exec_module(value); return value
CORE=load("browser_acceptance_production",ROOT/"scripts/browser_acceptance.py")
SITE=load("production_carried_site_checks",ROOT/"scripts/verify-preview-browser.py")
IDENTITY=load("production_identity_browser",ROOT/"scripts/production-identity.py")
CONTRACT=load("production_browser_release_contract",ROOT/"scripts/release_contract.py")
BrowserFailure=CORE.BrowserFailure
browser_ad_gate=CORE.browser_ad_gate

def parse_proxy_server(value):
    parsed=CORE.urllib.parse.urlsplit(value)
    if parsed.scheme!="socks5" or not parsed.hostname or parsed.port is None or not 1<=parsed.port<=65535 or parsed.username or parsed.password or parsed.path or parsed.query or parsed.fragment:
        raise ValueError("proxy must be socks5://<host>:<port>")
    return value

def assert_asset_policy(chrome,info):
    paths=CORE.json.dumps(list(info["content"]["shared_assets"]))
    origin=CORE.json.dumps(info["content"]["asset_origin"])
    result=chrome.evaluate("""(() => {const paths=new Set(%s),origin=%s;
      return performance.getEntriesByType('resource').filter(e=>paths.has(new URL(e.name).pathname.slice(1)))
        .every(e=>new URL(e.name).origin===origin)})()"""%(paths,origin),check="exact shared asset hostname policy")
    CORE.assert_true(result,"shared asset requested from wrong origin")

def acceptance(base,locale,profile,shared,proxy_server=None,info=None):
    if info is None: raise BrowserFailure("sealed --release-dir contract required; no Tour-only alternate verifier")
    plan=info["browser_plan"]
    if plan["locale"]!=locale or plan["canonical_routes"]!=info["content"]["canonical_routes"]:
        raise BrowserFailure("Production/Preview plan route drift")
    profile=dict(profile); profile["shared_assets_policy"]=info["content"]["asset_policy"]
    class CheckedChrome(CORE.Chrome):
        def __init__(self,*args,**kwargs):
            super().__init__(*args,**kwargs); self.call("Log.enable")
            owner=self
            class Events(list):
                def clear(self):
                    SITE.assert_browser_events(owner,base); super().clear()
            self.events=Events(self.events)
        def navigate(self,url,width,height):
            if self.current_route!="about:blank":
                self.evaluate("true",check="flush production browser errors")
                SITE.assert_browser_events(self,base)
                assert_asset_policy(self,info)
            return super().navigate(url,width,height)
        def close(self):
            try:
                if sys.exc_info()[0] is None:
                    self.evaluate("true",check="flush final production errors")
                    SITE.assert_browser_events(self,base); assert_asset_policy(self,info)
            finally: super().close()
    original=CORE.Chrome; CORE.Chrome=CheckedChrome
    try: CORE.acceptance(base,locale,profile,shared,proxy_server=proxy_server,site_plan=plan)
    finally: CORE.Chrome=original
    chrome=CheckedChrome(proxy_server=proxy_server)
    try:
        SITE.site_browser_acceptance(chrome,base,profile,plan)
        assert_asset_policy(chrome,info)
    finally: chrome.close()

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("base"); parser.add_argument("locale")
    parser.add_argument("--release-dir",required=True); parser.add_argument("--proxy-server",type=parse_proxy_server)
    args=parser.parse_args()
    try:
        identity=IDENTITY.load_identity(ROOT/"production/identity.json")
        profile=next(p for p in identity["locales"] if p["locale"]==args.locale)
        if args.base!=profile["production_public_url"]: raise BrowserFailure("public hostname/HTTPS identity mismatch")
        info=CONTRACT.load(args.release_dir,current=True)
        if info["locale"]!=args.locale: raise BrowserFailure("release locale mismatch")
        CONTRACT.shared_freshness(info,identity["shared"])
        acceptance(args.base,args.locale,profile,identity["shared"],args.proxy_server,info)
    except (RuntimeError,OSError,ValueError,KeyError,TypeError,StopIteration) as exc:
        print("[production-browser] FAILED: "+str(exc),file=sys.stderr); return 1
    print("PRODUCTION BROWSER ACCEPTANCE: PASS (sealed Site plan + Tour + asset policy)")
    return 0
if __name__=="__main__": raise SystemExit(main())
