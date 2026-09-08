#!/usr/bin/env python3
"""Minimal server provisioning for the IndexNow closeout protocol.

This is not a production deployment orchestrator: it makes one Aliyun mutation,
then leaves public verification and submission to tour-i18n indexnow bootstrap.
"""
import argparse
import importlib.util
import os
import pathlib
import re
import secrets
import shlex
import stat
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location("production_identity", ROOT / "scripts" / "production-identity.py")
IDENTITY = importlib.util.module_from_spec(spec)
spec.loader.exec_module(IDENTITY)

class CloseoutError(RuntimeError): pass

KEY_MIN_LENGTH = 8
KEY_MAX_LENGTH = 128
KEY_PATTERN = re.compile(r"^[A-Za-z0-9-]+$")
LOCALE_PATTERN = re.compile(r"^[a-z]{2,3}-[A-Z]{2}$")

def read_key(path):
    path = pathlib.Path(path)
    try:
        info, raw = path.lstat(), path.read_bytes()
    except OSError as exc:
        raise CloseoutError(f"read IndexNow key file: {exc}") from exc
    if not stat.S_ISREG(info.st_mode) or stat.S_ISLNK(info.st_mode):
        raise CloseoutError("IndexNow key file must be a regular non-symlink file")
    value = raw[:-2] if raw.endswith(b"\r\n") else raw[:-1] if raw.endswith(b"\n") else raw
    if not value or b"\r" in value or b"\n" in value:
        raise CloseoutError("IndexNow key file must contain one non-empty line")
    try: key = value.decode("utf-8")
    except UnicodeDecodeError as exc: raise CloseoutError("IndexNow key file must contain UTF-8 text") from exc
    if not KEY_MIN_LENGTH <= len(key) <= KEY_MAX_LENGTH or not KEY_PATTERN.fullmatch(key):
        raise CloseoutError(f"IndexNow key must be {KEY_MIN_LENGTH}..{KEY_MAX_LENGTH} characters of [A-Za-z0-9-]")
    if path.name != key + ".txt": raise CloseoutError("IndexNow key file name must be <key>.txt")
    return key, raw

def default_key_store_root():
    data_home = pathlib.Path(os.environ["XDG_DATA_HOME"]) if os.environ.get("XDG_DATA_HOME") else pathlib.Path.home() / ".local" / "share"
    if not data_home.is_absolute():
        raise CloseoutError("XDG_DATA_HOME must be an absolute path")
    root = data_home / "go-tour-indexnow"
    try:
        root.resolve().relative_to(ROOT.resolve())
    except ValueError:
        return root
    raise CloseoutError("IndexNow local key store must be outside the repository")

def secure_directory(path):
    if path.exists() or path.is_symlink():
        if path.is_symlink() or not path.is_dir():
            raise CloseoutError("IndexNow local key directory must be a real directory")
    else:
        path.mkdir(mode=0o700, parents=True)
    try:
        path.chmod(0o700)
    except OSError as exc:
        raise CloseoutError("cannot secure IndexNow local key directory") from exc

def default_key_file(locale, store_root=None):
    if not LOCALE_PATTERN.fullmatch(locale):
        raise CloseoutError("invalid locale for IndexNow local key store")
    root = pathlib.Path(store_root) if store_root is not None else default_key_store_root()
    if not root.is_absolute():
        raise CloseoutError("IndexNow local key store must be an absolute path")
    try:
        root.resolve().relative_to(ROOT.resolve())
    except ValueError:
        pass
    else:
        raise CloseoutError("IndexNow local key store must be outside the repository")
    secure_directory(root)
    directory = root / locale
    if not directory.exists() and not directory.is_symlink():
        secure_directory(directory)
        key = secrets.token_hex(32)
        path = directory / (key + ".txt")
        try:
            descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            with os.fdopen(descriptor, "wb") as output:
                output.write((key + "\n").encode("ascii")); output.flush(); os.fsync(output.fileno())
        except OSError as exc:
            raise CloseoutError("cannot create IndexNow local key") from exc
        return path, "generated"
    secure_directory(directory)
    entries = list(directory.iterdir())
    if len(entries) != 1:
        raise CloseoutError("IndexNow local key directory must contain exactly one key file")
    path = entries[0]
    if path.is_symlink() or not path.is_file():
        raise CloseoutError("IndexNow local key entry must be a regular non-symlink file")
    try:
        path.chmod(0o600)
    except OSError as exc:
        raise CloseoutError("cannot secure IndexNow local key file") from exc
    read_key(path)
    return path, "reused"

def profile_for_locale(identity, locale):
    matches = [p for p in identity["locales"] if p["locale"] == locale]
    if len(matches) != 1: raise CloseoutError(f"production identity must contain exactly one locale {locale}")
    if matches[0]["production_state"] != "live":
        raise CloseoutError(f"indexnow closeout requires production_state=live, got {matches[0]['production_state']}")
    return matches[0]

# Sent through stdin to the formal origin. Commands originate exclusively from
# production identity, are split to argv, and never pass through a shell.
REMOTE_PROVISION = r'''import os, pathlib, re, shlex, stat, subprocess, sys
data_root,vhost_name,hostname,filename,test_command,reload_command=sys.argv[1:]
root=pathlib.Path(data_root); vhost=pathlib.Path(vhost_name); verification=root/"verification"; final=verification/filename
def fail(message): raise RuntimeError(message)
def regular(path):
 try: info=path.lstat()
 except FileNotFoundError: return False
 return stat.S_ISREG(info.st_mode) and not stat.S_ISLNK(info.st_mode)
def blocks(text,marker,label):
 result=[]
 for match in re.finditer(marker,text):
  depth=0
  for pos in range(match.end()-1,len(text)):
   if text[pos]=="{": depth+=1
   elif text[pos]=="}":
    depth-=1
    if depth==0: result.append((match.start(),pos+1,text[match.start():pos+1])); break
  else: fail("unterminated "+label)
 return result
def run(command):
 argv=shlex.split(command)
 if not argv: fail("empty formal Nginx command")
 subprocess.run(argv,check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
if not root.is_dir() or root.is_symlink() or not regular(vhost): fail("invalid formal data root or vhost")
incoming=sys.stdin.buffer.read()
if not incoming: fail("empty IndexNow key")
before_vhost=vhost.read_bytes(); before_key=final.read_bytes() if regular(final) else None; created_dir=False
if final.exists() and not regular(final): fail("existing IndexNow key is not a regular file")
try:
 if verification.exists() or verification.is_symlink():
  if not verification.is_dir() or verification.is_symlink(): fail("verification must be a real directory")
 else: verification.mkdir(mode=0o755); created_dir=True
 if before_key is None:
  staged=verification/("."+filename+".staging"); fd=os.open(staged,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o640)
  with os.fdopen(fd,"wb") as output: output.write(incoming); output.flush(); os.fsync(output.fileno())
  os.replace(staged,final); os.chmod(final,0o644)
 elif before_key != incoming: fail("existing IndexNow key content conflicts")
 text=before_vhost.decode("utf-8"); expected=str(final)
 locations=blocks(text,r"location\s*=\s*/"+re.escape(filename)+r"\s*\{","exact IndexNow location")
 exact=lambda block: re.search(r"\balias\s+"+re.escape(expected)+r";",block) and re.search(r"\bdefault_type\s+text/plain;",block)
 if len(locations)>1 or (locations and not exact(locations[0][2])): fail("conflicting existing exact IndexNow location")
 if not locations:
  servers=blocks(text,r"\bserver\s*\{","server block")
  candidates=[b for b in servers if re.search(r"\blisten\s+443(?:\s+ssl)?\s*;",b[2]) and re.search(r"\bserver_name\s+"+re.escape(hostname)+r"(?:\s|;)",b[2])]
  if len(candidates)!=1: fail("expected exactly one HTTPS server insertion point")
  start,end,_=candidates[0]; location="\n    location = /%s {\n        alias %s;\n        default_type text/plain;\n    }\n"%(filename,expected)
  vhost.write_text(text[:end-1]+location+text[end-1:],encoding="utf-8")
 run(test_command); run(reload_command)
except Exception:
 vhost.write_bytes(before_vhost)
 if before_key is None: final.unlink(missing_ok=True)
 else: final.write_bytes(before_key)
 if created_dir:
  try: verification.rmdir()
  except OSError: pass
 try: run(test_command); run(reload_command)
 except Exception: pass
 raise
print("PROVISIONING PASS")
'''

class Closeout:
    def __init__(self, locale, key_file=None, key_store_root=None):
        self.locale = locale
        identity = IDENTITY.load_identity(ROOT / "production" / "identity.json")
        self.profile, self.shared = profile_for_locale(identity, locale), identity["shared"]
        if key_file is None:
            self.key_file, self.key_source = default_key_file(locale, key_store_root)
        else:
            self.key_file, self.key_source = pathlib.Path(key_file), "explicit"
        self.key, self.key_bytes = read_key(self.key_file)
    def provision(self):
        values=(self.profile["data_root"],self.profile["nginx_vhost_path"],self.profile["production_hostname"],self.key+".txt",self.shared["nginx_test_command"],self.shared["nginx_reload_command"])
        command="python3 -c %s %s"%(shlex.quote(REMOTE_PROVISION)," ".join(shlex.quote(v) for v in values))
        result=subprocess.run(["ssh","-o","BatchMode=yes","-o","ConnectTimeout=10",self.profile["origin_ssh_alias"],command],input=self.key_bytes,timeout=300)
        if result.returncode: raise CloseoutError("remote IndexNow provisioning failed")
    def run(self):
        if self.key_source in ("generated", "reused"):
            print(f"IndexNow local key: {self.key_source}")
        self.provision()
        result=subprocess.run(["go","run","-mod=readonly","./cmd/tour-i18n","indexnow","bootstrap","--locale",self.locale,"--key-file",str(self.key_file)])
        if result.returncode: raise CloseoutError("formal IndexNow bootstrap failed after provisioning")
        print(f"IndexNow closeout: PASS (locale={self.locale})")

def main(argv=None):
    parser=argparse.ArgumentParser(description="provision and submit IndexNow for one live locale")
    parser.add_argument("--locale",required=True); parser.add_argument("--key-file"); args=parser.parse_args(argv)
    try: Closeout(args.locale,args.key_file).run()
    except (CloseoutError,IDENTITY.IdentityError,OSError,subprocess.TimeoutExpired) as exc:
        print(f"indexnow closeout: FAILED: {exc}",file=sys.stderr); return 1
    return 0
if __name__ == "__main__": raise SystemExit(main())
