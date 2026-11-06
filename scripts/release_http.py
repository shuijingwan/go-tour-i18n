"""HTTP transport adapter for the Go-derived sealed release verification model.

The same model and checks run on origin and the established zgocloud public
runner. No fixed route array or count, renderer or language authority is owned
here. Source of truth is release-contract check's exact current artifact.
"""
import hashlib
import importlib.util
import json
import pathlib
import shlex
import subprocess
import sys
import tempfile
import time
import urllib.parse
import xml.etree.ElementTree as ET
from html.parser import HTMLParser


class HTTPError(RuntimeError): pass


class Page(HTMLParser):
    def __init__(self):
        super().__init__(); self.lang=[]; self.canonical=[]; self.main=0; self.depth=0; self.text=[]; self.assets=[]; self.legacy=False
    def handle_starttag(self,tag,attrs):
        a=dict(attrs)
        if tag=="html": self.lang.append(a.get("lang"))
        if tag=="link" and "canonical" in a.get("rel","").split(): self.canonical.append(a.get("href"))
        if tag=="main": self.main+=1; self.depth+=1
        if tag in ("script","img") and a.get("src"): self.assets.append(a["src"])
        if tag=="link" and "stylesheet" in a.get("rel","").split(): self.assets.append(a.get("href",""))
        if "site-home" in a.get("class","").split() or a.get("ng-app")=="tour": self.legacy=True
    def handle_endtag(self,tag):
        if tag=="main" and self.depth: self.depth-=1
    def handle_data(self,text):
        if self.depth: self.text.append(text)


def page(body):
    p=Page(); p.feed(body.decode("utf-8")); return p


def main_digest(p):
    return hashlib.sha256(" ".join(" ".join(p.text).split()).encode()).hexdigest()


def model(info,release):
    a=info["content"]
    if info["browser_plan"]["canonical_routes"]!=a["canonical_routes"]:
        raise HTTPError("sealed browser plan and current route coverage differ")
    expected={}
    for route,path in a["pages"].items():
        expected[route]=main_digest(page((pathlib.Path(release)/"_content"/path).read_bytes()))
    return {"input_identity":info["input_identity"],"content_identity":a["identity"],"locale":a["locale"],
            "origin":a["origin"],"routes":a["canonical_routes"],"families":a["families"],
            "coverage_identity":a["coverage_identity"],"shared_assets_identity":a["shared_assets_identity"],
            "asset_origin":a["asset_origin"],"shared_paths":list(a["shared_assets"]),"page_main_sha256":expected}


def curl_request(url,headers=None,origin_ip=None):
    with tempfile.TemporaryDirectory(prefix="go-tour-http-") as directory:
        body=pathlib.Path(directory)/"body"; head=pathlib.Path(directory)/"headers"
        command=["curl","-sS","--connect-timeout","5","--max-time","15","-D",head,"-o",body,"-w","%{http_code}"]
        parsed=urllib.parse.urlsplit(url)
        if origin_ip: command += ["--resolve","%s:%s:%s"%(parsed.hostname,parsed.port or (443 if parsed.scheme=="https" else 80),origin_ip)]
        for key,value in (headers or {}).items(): command += ["-H",key+": "+value]
        for attempt in range(1,6):
            result=subprocess.run([str(x) for x in command]+[url],capture_output=True,text=True,check=False)
            status=int(result.stdout) if result.stdout.isdigit() else 0
            transient=result.returncode in (6,7,16,28,35,97) or result.returncode==0 and status in (522,525)
            if transient and attempt<5: time.sleep(min(attempt,4)); continue
            if result.returncode: raise HTTPError("url=%s curl_exit=%s HTTP=%s attempts=%s error=%s"%(url,result.returncode,status,attempt,result.stderr))
            values={}
            for line in head.read_text().splitlines():
                if ":" in line:
                    key,value=line.split(":",1); values[key.lower()]=value.strip()
            return status,body.read_bytes(),values


def verify(m,base=None,cache_header=None,origin_ip=None,request=curl_request):
    base=(base or m["origin"]).rstrip("/")
    def get(route,status=200,headers=None):
        code,body,response_headers=request(base+route,headers=headers,origin_ip=origin_ip)
        if code!=status: raise HTTPError("%s: expected HTTP %s without redirect, got %s"%(route,status,code))
        return body,response_headers
    runtime,_=get("/site-release.json")
    runtime=json.loads(runtime)
    for key,expected in (("identity",m["content_identity"]),("input_identity",m["input_identity"]),
                         ("coverage_identity",m["coverage_identity"]),("shared_assets_identity",m["shared_assets_identity"])):
        if runtime.get(key)!=expected: raise HTTPError("deployed runtime identity mismatch: "+key)
    for route in m["routes"]:
        body,headers=get(route); p=page(body)
        if p.lang!=[m["locale"]] or p.canonical!=[m["origin"]+route]: raise HTTPError("canonical/lang mismatch: "+route)
        if route in m["page_main_sha256"]:
            if p.main!=1 or p.legacy or main_digest(p)!=m["page_main_sha256"][route]: raise HTTPError("wrong/stale Site content: "+route)
        for raw in p.assets:
            u=urllib.parse.urlsplit(urllib.parse.urljoin(base+route,raw))
            if u.path.lstrip("/") in m["shared_paths"] and "%s://%s"%(u.scheme,u.netloc)!=m["asset_origin"]:
                raise HTTPError("wrong shared asset hostname: "+raw)
        csp=headers.get("content-security-policy","")
        if csp:
            directives=dict((part.split()[0],part.split()[1:]) for part in csp.split(";") if part.split())
            asset_host=urllib.parse.urlsplit(m["asset_origin"]).hostname
            for directive in ("script-src","style-src","font-src","img-src"):
                allowed=directives.get(directive,directives.get("default-src"))
                if allowed is not None and not any(v in ("*","https:",m["asset_origin"],asset_host) for v in allowed):
                    raise HTTPError("CSP blocks shared asset policy: "+directive)
        if cache_header:
            cache=headers.get(cache_header.lower())
            if cache not in ("MISS","HIT","EXPIRED","REVALIDATED","UPDATING","STALE"): raise HTTPError("cache eligibility failure: "+str(cache))
    sitemap,_=get("/sitemap.xml"); root=ET.fromstring(sitemap)
    urls=[node.text for node in root.findall("{http://www.sitemaps.org/schemas/sitemap/0.9}url/{http://www.sitemaps.org/schemas/sitemap/0.9}loc")]
    if sorted(urls)!=[m["origin"]+r for r in m["routes"]] or len(urls)!=len(set(urls)): raise HTTPError("non-exact current sitemap")
    robots,_=get("/robots.txt")
    if "Sitemap: "+m["origin"]+"/sitemap.xml" not in robots.decode(): raise HTTPError("robots sitemap mismatch")
    negatives={"/__release_missing_route__","/socket","/socket/","/home.html","/doc/play/__release_missing__.go"}
    negatives.update((f[:-2] if f.endswith("**") else f).rstrip("/")+"/__release_missing_route__" for f in m["families"])
    for route in sorted(negatives): get(route,404)
    get("/socket",404,{"Connection":"Upgrade","Upgrade":"websocket"})
    return {"result":"PASS","routes":len(m["routes"]),"input_identity":m["input_identity"]}


def public(info,release,shared,profile,direct=False):
    # Payload travels through stdin, so page/route inventory cannot exceed an
    # argument-size boundary and silently truncate a multi-locale contract.
    payload={"model":model(info,release),"origin_ip":profile["origin_ip"] if direct else None,
             "cache_header":None if direct else profile["cache_header"]}
    code=pathlib.Path(__file__).read_text()
    command=["ssh","-o","BatchMode=yes","-o","ConnectTimeout=10",shared["zgocloud_ssh_alias"],
             "python3 -c "+shlex.quote(code)+" --remote"]
    result=subprocess.run(command,input=json.dumps(payload),text=True,check=False,timeout=1800)
    if result.returncode: raise HTTPError("public/direct-origin exact contract verifier failed")


def remote_main():
    value=json.load(sys.stdin)
    # Python 3.10 zgocloud; no deployment mutation or language input reads.
    print(json.dumps(verify(value["model"],base=value.get("base"),cache_header=value["cache_header"],origin_ip=value["origin_ip"])))


if __name__=="__main__" and sys.argv[1:]==["--remote"]: remote_main()
