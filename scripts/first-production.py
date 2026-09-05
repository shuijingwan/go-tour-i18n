#!/usr/bin/env python3
"""Fail-closed first-production bootstrap orchestrator.

The sole positional input is a validated release bundle. Every production
target comes from production/identity.json. Secrets are opened only by root on
the aliyun host and are never returned to this process.
"""

from __future__ import annotations

import datetime as dt
import base64
import hashlib
import importlib.util
import json
import os
import pathlib
import re
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time


ROOT = pathlib.Path(__file__).resolve().parent.parent
IDENTITY_SPEC = importlib.util.spec_from_file_location(
    "production_identity", ROOT / "scripts" / "production-identity.py"
)
IDENTITY = importlib.util.module_from_spec(IDENTITY_SPEC)
IDENTITY_SPEC.loader.exec_module(IDENTITY)
RECEIPT_SCHEMA = "go-tour-i18n/first-production-receipt/v1"
ALIYUN_ONEINSTACK_NGINX = "/usr/local/nginx/sbin/nginx"
ZGOCLOUD_ONEINSTACK_NGINX = "/usr/local/nginx/sbin/nginx"
STAGE_ORDER = (
    "preflight", "infrastructure", "playground-origin", "deploy",
    "direct-origin", "cloudflare-dns", "public-machine", "browser",
)
STAGE_LABELS = {
    "preflight": "预检",
    "infrastructure": "基础设施",
    "playground-origin": "Playground Origin",
    "deploy": "首次部署",
    "direct-origin": "源站验收",
    "cloudflare-dns": "Cloudflare DNS",
    "public-machine": "公网验收",
    "browser": "浏览器验收",
}


class FirstProductionError(RuntimeError):
    def __init__(self, stage: str, expected: str, actual: str, next_step: str):
        super().__init__(actual)
        self.stage = stage
        self.expected = expected
        self.actual = actual
        self.next_step = next_step


def cloudflare_dns_state(records, hostname, origin_ip):
    """Return absent/exact or reject every unknown/conflicting record."""
    if type(records) is not list or len(records) > 1:
        raise ValueError("DNS identity is not unique")
    if not records:
        return "absent"
    record = records[0]
    expected = {"type": "A", "name": hostname, "content": origin_ip, "proxied": True}
    if any(record.get(key) != value for key, value in expected.items()):
        raise ValueError("DNS identity conflicts")
    return "exact"


def update_playground_config(text, hostname):
    """Apply the exact two-location allowlist mutation used on ZgoCloud."""
    if text.count("location = /compile {") != 1 or text.count("location = /fmt {") != 1:
        raise ValueError("Playground compile/fmt locations are not unique")
    pattern = r'if \(\$http_origin !~ "\^https://\(([^"()]+)\)\\\.shuijingwanwq\\\.com\$"\) \{'
    matches = re.findall(pattern, text)
    if len(matches) != 2 or matches[0] != matches[1]:
        raise ValueError("Playground allowlist structure is not uniquely identifiable")
    labels = matches[0].split("|")
    if len(labels) != len(set(labels)) or not all(re.fullmatch(r"[a-z0-9-]+", label) for label in labels):
        raise ValueError("Playground allowlist labels are invalid")
    suffix = ".shuijingwanwq.com"
    if not hostname.endswith(suffix) or not re.fullmatch(r"[a-z0-9-]+", hostname[:-len(suffix)]):
        raise ValueError("production hostname cannot be represented by the formal allowlist")
    label = hostname[:-len(suffix)]
    if label in labels:
        return text
    labels.append(label)
    replacement = 'if ($http_origin !~ "^https://(' + "|".join(labels) + r')\.shuijingwanwq\.com$") {'
    updated = re.sub(pattern, lambda _: replacement, text)
    if updated == text or len(re.findall(pattern, updated)) != 2:
        raise ValueError("Playground allowlist mutation was not exact")
    return updated


def systemd_unit_text(profile):
    return f'''[Unit]
Description=A Tour of Go {profile["locale"]}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={profile["service_user"]}
Group={profile["service_user"]}
ExecStart={profile["current"]}/bin/tour -http 127.0.0.1:{profile["loopback_port"]}
EnvironmentFile={profile["environment_file"]}
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=strict

[Install]
WantedBy=multi-user.target
'''


def nginx_vhost_text(profile):
    host = profile["production_hostname"]
    port = profile["loopback_port"]
    cert = profile["tls_certificate_path"]
    key = profile["tls_key_path"]
    return rf'''server {{
  listen 80;
  listen [::]:80;
  server_name {host};

  return 301 https://$host$request_uri;
}}

server {{
  listen 443 ssl;
  listen [::]:443 ssl;
  http2 on;

  server_name {host};

  ssl_certificate {cert};
  ssl_certificate_key {key};
  ssl_protocols TLSv1.2 TLSv1.3;
  ssl_ecdh_curve X25519:prime256v1:secp384r1:secp521r1;
  ssl_ciphers ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256;
  ssl_conf_command Ciphersuites TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256:TLS_AES_128_GCM_SHA256;
  ssl_conf_command Options PrioritizeChaCha;
  ssl_prefer_server_ciphers on;
  ssl_session_timeout 10m;
  ssl_session_cache shared:SSL:10m;
  ssl_buffer_size 2k;

  add_header Strict-Transport-Security max-age=15768000;

  location / {{
    proxy_pass http://127.0.0.1:{port};
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header Host $http_host;
    proxy_set_header X-NginX-Proxy true;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_max_temp_file_size 0;
  }}

  location ~ /(\.user\.ini|\.ht|\.git|\.svn|\.project|LICENSE|README\.md) {{
    deny all;
  }}

  location /.well-known {{
    allow all;
  }}
}}
'''


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def release_locale(release_dir: pathlib.Path) -> str:
    try:
        release = json.loads((release_dir / "release.json").read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise FirstProductionError("preflight", "valid release.json", str(exc), "重新执行 production publish") from exc
    locale = release.get("locale") if type(release) is dict else None
    if type(locale) is not str or not locale:
        raise FirstProductionError("preflight", "release.json locale", repr(locale), "重新执行 production publish")
    return locale


def safe_release_name(release_dir: pathlib.Path) -> str:
    name = release_dir.name
    if not name.startswith("go-tour-release-"):
        raise FirstProductionError("preflight", "go-tour-release-<safe-name>", name, "使用 publish 生成的正式目录")
    remote = name.removeprefix("go-tour-release-")
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*", remote):
        raise FirstProductionError("preflight", "safe release name", remote, "使用 publish 生成的正式目录")
    return remote


class Orchestrator:
    def __init__(self, release_dir: pathlib.Path):
        if not release_dir.is_dir() or release_dir.is_symlink():
            raise FirstProductionError("preflight", "real release directory", str(release_dir), "重新执行 production publish")
        self.release_dir = release_dir.resolve()
        self.release_name = safe_release_name(self.release_dir)
        self.locale = release_locale(self.release_dir)
        self.identity = IDENTITY.load_identity(ROOT / "production" / "identity.json")
        profiles = [p for p in self.identity["locales"] if p["locale"] == self.locale]
        if len(profiles) != 1:
            raise FirstProductionError("preflight", "one formal production identity", self.locale, "先在 production/identity.json 建立并审核 identity")
        self.profile = profiles[0]
        self.shared = self.identity["shared"]
        if self.profile["production_state"] != "first-production":
            raise FirstProductionError(
                "preflight", "production_state=first-production",
                f"production_state={self.profile['production_state']}",
                "已有 production locale 只能使用日常 deploy/verify，不得重复 bootstrap",
            )
        if self.profile["cdn"] != "cloudflare":
            raise FirstProductionError("preflight", "Cloudflare first-production profile", self.profile["cdn"], "现有非 Cloudflare 站点继续使用日常发布流程")
        self.run_id = f"{dt.datetime.now(dt.timezone.utc):%Y%m%dT%H%M%SZ}-{os.getpid()}"
        self.receipt_path = self.release_dir.parent / f"{self.release_dir.name}.first-production-receipt.json"
        self.receipt = self._load_or_new_receipt()
        self.temp = pathlib.Path(tempfile.mkdtemp(prefix="go-tour-first-production-"))
        self.control = {
            self.shared["aliyun_ssh_alias"]: self.temp / "aliyun.control",
            self.shared["zgocloud_ssh_alias"]: self.temp / "zgocloud.control",
        }
        self.cf_socks_local_port = None
        self.cf_socks_aliyun_port = None
        self._signal_handlers = {}
        self._install_signal_cleanup()

    @staticmethod
    def _free_loopback_port():
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
            listener.bind(("127.0.0.1", 0))
            return listener.getsockname()[1]

    def _install_signal_cleanup(self):
        def stop(signum, _frame):
            raise KeyboardInterrupt(f"received signal {signum}")
        for signum in (signal.SIGINT, signal.SIGTERM, signal.SIGHUP):
            self._signal_handlers[signum] = signal.getsignal(signum)
            signal.signal(signum, stop)

    def setup_cloudflare_network_tunnel(self):
        """Bind an aliyun-only SOCKS endpoint whose TCP egress is zgocloud."""
        if self.cf_socks_aliyun_port is not None:
            return
        self.cf_socks_local_port = self._free_loopback_port()
        self.cf_socks_aliyun_port = self._free_loopback_port()
        zgocloud = self.shared["zgocloud_ssh_alias"]
        aliyun = self.shared["aliyun_ssh_alias"]
        self.run(["ssh", *self.ssh_options(zgocloud), "-o", "ControlMaster=yes",
                  "-o", "ExitOnForwardFailure=yes", "-f", "-N",
                  "-D", f"127.0.0.1:{self.cf_socks_local_port}", zgocloud], stage="preflight", timeout=30)
        self.run(["ssh", *self.ssh_options(aliyun), "-o", "ControlMaster=yes",
                  "-o", "ExitOnForwardFailure=yes", "-o", "GatewayPorts=no", "-f", "-N",
                  "-R", f"127.0.0.1:{self.cf_socks_aliyun_port}:127.0.0.1:{self.cf_socks_local_port}", aliyun], stage="preflight", timeout=30)

    def _load_or_new_receipt(self):
        base = {
            "schema": RECEIPT_SCHEMA,
            "run_id": self.run_id,
            "locale": self.locale,
            "hostname": self.profile["production_hostname"],
            "release": self.release_name,
            "started_at": utc_now(),
            "completed_at": None,
            "result": "running",
            "stages": {},
        }
        if not self.receipt_path.exists():
            return base
        try:
            previous = json.loads(self.receipt_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            raise FirstProductionError("preflight", "valid existing receipt", str(self.receipt_path), "移走损坏 receipt 后重试")
        identity = (previous.get("schema"), previous.get("locale"), previous.get("hostname"), previous.get("release"))
        expected = (RECEIPT_SCHEMA, self.locale, self.profile["production_hostname"], self.release_name)
        if identity != expected:
            raise FirstProductionError("preflight", repr(expected), repr(identity), "不要复用其他 release 或 hostname 的 receipt")
        previous_result = previous.get("result")
        if previous_result == "passed":
            raise FirstProductionError(
                "preflight", "locale without completed first-production receipt",
                "first-production already passed", "使用 verify-production/browser acceptance，不得重复 bootstrap",
            )
        if previous_result not in ("running", "failed"):
            raise FirstProductionError("preflight", "running or failed resumable receipt", repr(previous_result), "移走无效 receipt 并检查 production identity")
        print(f"[首次生产] resume receipt 已识别，将重新验证所有关键 identity：{self.receipt_path}")
        base["started_at"] = previous.get("started_at") or base["started_at"]
        base["stages"] = previous.get("stages") if type(previous.get("stages")) is dict else {}
        return base

    def write_receipt(self, result=None):
        if result:
            self.receipt["result"] = result
            if result in ("passed", "failed"):
                self.receipt["completed_at"] = utc_now()
        temporary = self.receipt_path.with_name(self.receipt_path.name + f".tmp-{os.getpid()}")
        temporary.write_text(json.dumps(self.receipt, ensure_ascii=False, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
        os.chmod(temporary, 0o644)
        os.replace(temporary, self.receipt_path)

    def record(self, stage, result="PASS"):
        self.receipt["stages"][stage] = {"completed_at": utc_now(), "result": result}
        self.write_receipt()
        print(f"[首次生产] {STAGE_LABELS.get(stage, stage)}：{result}")

    def stage_passed(self, stage):
        value = self.receipt.get("stages", {}).get(stage, {})
        return type(value) is dict and value.get("result") == "PASS"

    def ssh_options(self, host):
        return [
            "-o", "BatchMode=yes", "-o", "ConnectTimeout=10",
            "-o", "ServerAliveInterval=5", "-o", "ServerAliveCountMax=3",
            "-o", "ConnectionAttempts=3", "-o", "ControlMaster=auto",
            "-o", "ControlPersist=60", "-o", f"ControlPath={self.control[host]}",
        ]

    def run(self, command, *, input_text=None, capture=False, stage="preflight", timeout=300):
        try:
            completed = subprocess.run(
                [str(part) for part in command], input=input_text, text=True,
                stdout=subprocess.PIPE if capture else None,
                stderr=subprocess.PIPE if capture else None,
                timeout=timeout, check=False,
            )
        except (OSError, subprocess.TimeoutExpired) as exc:
            raise FirstProductionError(stage, "command completed", str(exc), "检查本机工具和网络后重试") from exc
        if completed.returncode:
            detail = (completed.stderr or completed.stdout or f"exit {completed.returncode}").strip()
            # Secret values are never command arguments and remote scripts never print them.
            raise FirstProductionError(stage, "command exit 0", detail[-2000:], "按 stage/evidence 检查后重试")
        return (completed.stdout or "").strip()

    def ssh(self, host, script, args=(), *, capture=False, stage="preflight", timeout=300):
        command = ["ssh", *self.ssh_options(host), host, "bash", "-s", "--", *args]
        return self.run(command, input_text=script, capture=capture, stage=stage, timeout=timeout)

    def cleanup(self):
        for host, path in self.control.items():
            if path.exists() or path.is_socket():
                subprocess.run(["ssh", *self.ssh_options(host), "-O", "exit", host], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)
        shutil.rmtree(self.temp, ignore_errors=True)
        for signum, previous in getattr(self, "_signal_handlers", {}).items():
            signal.signal(signum, previous)

    def local_bundle_preflight(self):
        shell = f'''set -Eeuo pipefail
source {str(ROOT / "scripts" / "deploy-production.sh")!r}
validate_local_tools
release_name_from_path "$1" >/dev/null
validate_local_release "$1" >/dev/null
'''
        self.run(["bash", "-c", shell, "first-production-local-preflight", str(self.release_dir)], stage="preflight")

    def aliyun_preflight(self):
        p, s = self.profile, self.shared
        resume_deployed = "1" if self.stage_passed("deploy") else "0"
        expected_remote = f'{p["releases_root"]}/{self.release_name}'
        expected_unit_sha = hashlib.sha256(systemd_unit_text(p).encode()).hexdigest()
        expected_vhost_sha = hashlib.sha256(nginx_vhost_text(p).encode()).hexdigest()
        script = r'''set -Eeuo pipefail
data_root=$1; releases=$2; current=$3; lock=$4; service=$5; user=$6; port=$7
health=$8; env_file=$9; vhost=${10}; cert=${11}; key=${12}; hostname=${13}
secret=${14}; zone=${15}; origin_ip=${16}; expected_remote=${17}; resume_deployed=${18}
expected_unit_sha=${19}; expected_vhost_sha=${20}
nginx=${21}; cf_socks=${22}
fail() { printf '[first-production:aliyun] ERROR: %s\n' "$*" >&2; exit 1; }
[[ $(id -u) == 0 ]] || fail 'SSH account must be root'
for command_name in base64 chown chmod curl dirname grep id install mktemp mv openssl python3 readlink service sha256sum ss stat systemctl; do command -v "$command_name" >/dev/null || fail "missing tool: $command_name"; done
[[ -x $nginx ]] || fail "missing formal OneinStack Nginx executable: $nginx"
[[ -f $secret && ! -L $secret ]] || fail "Cloudflare secret source missing: $secret; provision root:root mode 0600 with CF_Token=<token>"
[[ $(stat -c '%U:%G %a' "$secret") == 'root:root 600' ]] || fail "Cloudflare secret source must be root:root mode 0600: $secret"
set -a; . "$secret"; set +a
[[ -n ${CF_Token:-} ]] || fail "Cloudflare secret source does not define non-empty CF_Token: $secret"
[[ -f $env_file && ! -L $env_file && $(stat -c '%U:%G %a' "$env_file") == 'root:root 600' ]] || fail "EnvironmentFile must be root:root mode 0600: $env_file"
set -a; . "$env_file"; set +a
[[ -n ${TOUR_AD_HTML:-} ]] || fail 'TOUR_AD_HTML is missing or empty (value is not printed)'
id "$user" >/dev/null 2>&1 || fail "service user missing: $user"
if [[ -e $data_root || -L $data_root ]]; then
  [[ -d $data_root && ! -L $data_root && $(readlink -f "$data_root") == "$data_root" ]] || fail "invalid data root: $data_root"
  if [[ -e $current || -L $current ]]; then
    [[ $resume_deployed == 1 && -L $current && $(readlink -f "$current") == "$expected_remote" ]] || fail "current already exists with an unrecognized identity: $current"
  fi
  [[ ! -e $lock && ! -L $lock ]] || fail "deployment lock exists: $lock"
  [[ ! -e $releases || ( -d $releases && ! -L $releases ) ]] || fail "invalid releases root: $releases"
fi
if [[ $resume_deployed == 1 ]]; then
  [[ $(systemctl is-active "$service" 2>/dev/null || true) == active ]] || fail 'resumed service is not active'
  ss -H -ltn "sport = :$port" | grep -q . || fail 'resumed service is not listening on its identity port'
  [[ $(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time 5 "$health" || true) == 200 ]] || fail 'resumed source health is not HTTP 200'
elif ss -H -ltn "sport = :$port" | grep -q .; then
  fail "target port is already listening: $port"
fi
unit_path=/etc/systemd/system/$service
for parent in "$(dirname "$unit_path")" "$(dirname "$vhost")" "$(dirname "$cert")" "$(dirname "$key")"; do
  [[ -d $parent && ! -L $parent ]] || fail "configuration parent is missing, non-directory, or symlinked: $parent"
done
if [[ -e $unit_path || -L $unit_path ]]; then
  [[ -f $unit_path && ! -L $unit_path ]] || fail 'existing systemd unit is not a regular file'
  actual_unit_sha=$(sha256sum "$unit_path"); actual_unit_sha=${actual_unit_sha%% *}
  [[ $actual_unit_sha == "$expected_unit_sha" ]] || fail 'existing systemd unit content is not the exact formal baseline'
elif systemctl cat "$service" >/dev/null 2>&1; then
  fail 'service identity exists outside its formal unit path'
fi
for dropin in "/etc/systemd/system/$service.d" "/run/systemd/system/$service.d" "/usr/local/lib/systemd/system/$service.d" "/usr/lib/systemd/system/$service.d" "/lib/systemd/system/$service.d"; do
  [[ ! -e $dropin && ! -L $dropin ]] || fail "systemd drop-in conflicts with formal baseline: $dropin"
done
if [[ -e $vhost || -L $vhost ]]; then
  [[ -f $vhost && ! -L $vhost ]] || fail 'existing vhost is not a regular file'
  actual_vhost_sha=$(sha256sum "$vhost"); actual_vhost_sha=${actual_vhost_sha%% *}
  [[ $actual_vhost_sha == "$expected_vhost_sha" ]] || fail 'existing vhost content is not the exact formal baseline'
fi
set +e
vhost_scan_output=$(python3 - "$vhost" "$hostname" <<'PY' 2>&1
import pathlib,sys
target=pathlib.Path(sys.argv[1]); hostname=sys.argv[2]
try:
    hits=set()
    for path in target.parent.iterdir():
        if not path.is_file():
            continue
        try: text=path.read_text(encoding='utf-8')
        except (OSError,UnicodeError): continue
        for raw in text.splitlines():
            line=raw.split('#',1)[0].strip()
            if line.startswith('server_name '):
                names=line[len('server_name '):]
                if names.endswith(';'):
                    names=names[:-1]
                if hostname in names.split():
                    hits.add(path)
except Exception as exc:
    sys.stderr.write('vhost hostname checker exception: {}\n'.format(exc))
    raise SystemExit(2)
if hits and hits != {target}:
    sys.stderr.write('production hostname is declared by: {}\n'.format(', '.join(sorted(str(path) for path in hits))) )
    raise SystemExit(10)
PY
)
vhost_scan_status=$?
set -e
case $vhost_scan_status in
  0) ;;
  10) fail 'production hostname is already declared by another Nginx vhost';;
  *) fail "Nginx vhost hostname checker failed: ${vhost_scan_output:-exit $vhost_scan_status}";;
esac
if [[ -e $cert || -L $cert || -e $key || -L $key ]]; then
  [[ -f $cert && ! -L $cert && -f $key && ! -L $key ]] || fail 'TLS certificate/key identity is partial or invalid'
  openssl x509 -in "$cert" -noout -checkhost "$hostname" >/dev/null || fail 'existing certificate does not cover hostname'
  cert_pub=$(openssl x509 -in "$cert" -pubkey -noout | sha256sum); cert_pub=${cert_pub%% *}
  key_pub=$(openssl pkey -in "$key" -pubout | sha256sum); key_pub=${key_pub%% *}
  [[ $cert_pub == "$key_pub" ]] || fail 'existing certificate and private key do not match'
else
  acme=''
  for candidate in /root/.acme.sh/acme.sh /root/oneinstack/acme.sh/acme.sh; do [[ -x $candidate ]] && { acme=$candidate; break; }; done
  [[ -n $acme ]] || fail 'cannot locate the verified acme.sh installation'
  grep -Eq '(^|[^[:alnum:]_])curl([^[:alnum:]_]|$)' "$acme" || fail 'installed acme.sh has no verifiable curl invocation'
  ! grep -Eq '/(usr/)?bin/curl([^[:alnum:]_]|$)' "$acme" || fail 'installed acme.sh bypasses PATH-resolved curl'
fi
readonly CF_CONNECT_TIMEOUT=5 CF_MAX_TIME=20 CF_RETRY_ATTEMPTS=3
cf_get() {
  output=$1; shift
  for attempt in $(seq 1 "$CF_RETRY_ATTEMPTS"); do
    set +e
    code=$(curl --socks5-hostname "$cf_socks" -sS --connect-timeout "$CF_CONNECT_TIMEOUT" --max-time "$CF_MAX_TIME" -o "$output" -w '%{http_code}' "$@")
    curl_exit=$?
    set -e
    if [[ $curl_exit == 0 && $code == 200 ]]; then return 0; fi
    if [[ $curl_exit != 0 ]]; then
      case $curl_exit in
        6|7|16|28|35) [[ $attempt != "$CF_RETRY_ATTEMPTS" ]] && { sleep "$attempt"; continue; };;
      esac
      fail "Cloudflare API transport failed (curl exit $curl_exit)"
    fi
    fail "Cloudflare API returned HTTP ${code:-000}"
  done
  fail 'Cloudflare API read retry exhausted'
}
cf_tmp=$(mktemp -d); trap 'rm -rf "$cf_tmp"' EXIT
cf_get "$cf_tmp/zone" -H "Authorization: Bearer $CF_Token" -H 'Content-Type: application/json' --get --data-urlencode "name=$zone" --data-urlencode 'status=active' https://api.cloudflare.com/client/v4/zones
zone_json=$(<"$cf_tmp/zone")
zone_id=$(python3 -c 'import json,sys; d=json.load(sys.stdin); r=d.get("result",[]); assert d.get("success") is True and len(r)==1; print(r[0]["id"])' <<<"$zone_json") || fail 'Cloudflare zone name did not resolve uniquely'
cf_get "$cf_tmp/dns" -H "Authorization: Bearer $CF_Token" -H 'Content-Type: application/json' --get --data-urlencode "name=$hostname" "https://api.cloudflare.com/client/v4/zones/$zone_id/dns_records"
dns_json=$(<"$cf_tmp/dns")
DNS_JSON="$dns_json" python3 - "$hostname" "$origin_ip" <<'PY' || fail 'existing Cloudflare DNS identity conflicts'
import json,os,sys
host,ip=sys.argv[1:]
d=json.loads(os.environ['DNS_JSON']); r=d.get('result',[])
assert d.get('success') is True
assert len(r) <= 1
if r:
    x=r[0]; assert x.get('type') == 'A' and x.get('name') == host and x.get('content') == ip and x.get('proxied') is True
PY
printf 'zone_id=%s\n' "$zone_id"
'''
        return self.ssh(s["aliyun_ssh_alias"], script, (
            p["data_root"], p["releases_root"], p["current"], p["deployment_lock"],
            p["systemd_service"], p["service_user"], str(p["loopback_port"]),
            p["localhost_health_url"], p["environment_file"], p["nginx_vhost_path"],
            p["tls_certificate_path"], p["tls_key_path"], p["production_hostname"],
            s["cloudflare_secret_file"], s["cloudflare_zone_name"], p["origin_ip"],
            expected_remote, resume_deployed, expected_unit_sha, expected_vhost_sha,
            ALIYUN_ONEINSTACK_NGINX, f"127.0.0.1:{self.cf_socks_aliyun_port}",
        ), capture=True, stage="preflight")

    def zgocloud_preflight(self):
        s = self.shared
        script = r'''set -Eeuo pipefail
vhost=$1; nginx=$2
fail() { printf '[first-production:zgocloud] ERROR: %s\n' "$*" >&2; exit 1; }
[[ $(id -u) == 0 ]] || fail 'SSH account must be root'
[[ -f $vhost && ! -L $vhost ]] || fail "Playground vhost is missing or invalid: $vhost"
[[ -x $nginx ]] || fail "missing formal ZgoCloud Nginx executable: $nginx"
"$nginx" -t >/dev/null || fail 'formal ZgoCloud Nginx config test failed'
python3 - "$vhost" <<'PY'
import pathlib,re,sys
text=pathlib.Path(sys.argv[1]).read_text(encoding='utf-8')
pattern=r'if \(\$http_origin !~ "\^https://\(([^"()]+)\)\\\.shuijingwanwq\\\.com\$"\) \{'
matches=re.findall(pattern,text)
assert len(matches)==2 and matches[0]==matches[1]
labels=matches[0].split('|')
assert labels and len(labels)==len(set(labels)) and all(re.fullmatch(r'[a-z0-9-]+', x) for x in labels)
assert text.count('location = /compile {')==1 and text.count('location = /fmt {')==1
print('origins=' + ','.join(labels))
PY
'''
        return self.ssh(s["zgocloud_ssh_alias"], script, (
            s["playground_vhost_path"], ZGOCLOUD_ONEINSTACK_NGINX,
        ), capture=True, stage="preflight")

    def shared_assets_freshness(self):
        if self.profile["shared_assets_policy"] != "shared-cloudflare":
            return
        export = self.temp / "shared-assets"
        self.run(["go", "run", "-mod=readonly", "./cmd/tour-i18n", "assets", "export", "--output", export], stage="preflight", timeout=300)
        self.run(["go", "run", "-mod=readonly", "./cmd/tour-i18n", "assets", "validate", "--input", export], stage="preflight", timeout=300)
        origin = self.shared["shared_assets_origin_root"]
        script = r'''set -Eeuo pipefail
root=$1
[[ -d $root && ! -L $root ]]
cd "$root"
sha256sum -c --strict SHA256SUMS >/dev/null
sha256sum SHA256SUMS
'''
        remote = self.ssh(self.shared["aliyun_ssh_alias"], script, (origin,), capture=True, stage="preflight")
        local_manifest_sha = hashlib.sha256((export / "SHA256SUMS").read_bytes()).hexdigest()
        if remote.split()[0] != local_manifest_sha:
            raise FirstProductionError("preflight", "shared-assets origin matches current export", remote.split()[0], "先完成 deploy-shared-assets 与正式公网验证")
        self.run([ROOT / "scripts" / "verify-shared-assets-public.sh", export], stage="preflight", timeout=300)

    def preflight(self):
        self.local_bundle_preflight()
        self.shared_assets_freshness()
        self.setup_cloudflare_network_tunnel()
        aliyun = self.aliyun_preflight()
        zgocloud = self.zgocloud_preflight()
        self.receipt["cloudflare_zone_id"] = aliyun.removeprefix("zone_id=")
        self.receipt["preflight_playground_identity"] = zgocloud
        self.record("preflight")

    def bootstrap_infrastructure(self):
        p, s = self.profile, self.shared
        unit_text = systemd_unit_text(p)
        vhost_text = nginx_vhost_text(p)
        unit_b64 = base64.b64encode(unit_text.encode()).decode()
        vhost_b64 = base64.b64encode(vhost_text.encode()).decode()
        unit_sha = hashlib.sha256(unit_text.encode()).hexdigest()
        vhost_sha = hashlib.sha256(vhost_text.encode()).hexdigest()
        script = r"""set -Eeuo pipefail
data_root=$1; releases=$2; current=$3; service=$4; user=$5; port=$6; env_file=$7
vhost=$8; cert=$9; key=${10}; hostname=${11}; secret=${12}
unit_b64=${13}; vhost_b64=${14}; expected_unit_sha=${15}; expected_vhost_sha=${16}
nginx=${17}; cf_socks=${18}
fail() { printf '[first-production:infrastructure] ERROR: %s\n' "$*" >&2; exit 1; }
install -d -o root -g root -m 0755 "$data_root" "$releases"
unit=/etc/systemd/system/$service
if [[ -e $unit || -L $unit ]]; then
  [[ -f $unit && ! -L $unit ]] || fail "systemd unit conflicts: $unit"
  actual=$(sha256sum "$unit"); actual=${actual%% *}
  [[ $actual == "$expected_unit_sha" ]] || fail "systemd unit conflicts: $unit"
else
  temporary=$unit.tmp.$$
  printf '%s' "$unit_b64" | base64 -d >"$temporary"
  chown root:root "$temporary"; chmod 0644 "$temporary"; mv -T "$temporary" "$unit"
fi
systemctl daemon-reload
if [[ ! -f $cert || ! -f $key ]]; then
  acme=''
  for candidate in /root/.acme.sh/acme.sh /root/oneinstack/acme.sh/acme.sh; do [[ -x $candidate ]] && { acme=$candidate; break; }; done
  [[ -n $acme ]] || fail 'cannot locate the verified acme.sh installation'
  set -a; . "$secret"; set +a
  [[ -n ${CF_Token:-} ]] || fail 'CF_Token missing from formal secret source'
  curl_bin=$(command -v curl)
  proxy_bin=$(mktemp -d); trap 'rm -rf "$proxy_bin"' EXIT
  # Do not assume acme.sh proxy-variable behavior: require its installed
  # source to use PATH-resolved curl, then wrap that executable per invocation.
  grep -Eq '(^|[^[:alnum:]_])curl([^[:alnum:]_]|$)' "$acme" || fail 'installed acme.sh has no verifiable curl invocation'
  ! grep -Eq '/(usr/)?bin/curl([^[:alnum:]_]|$)' "$acme" || fail 'installed acme.sh bypasses PATH-resolved curl'
  printf '#!/usr/bin/env bash\nexec %q --socks5-hostname %q "$@"\n' "$curl_bin" "$cf_socks" >"$proxy_bin/curl"
  chmod 0700 "$proxy_bin/curl"
  PATH="$proxy_bin:$PATH" "$acme" --issue --dns dns_cf -d "$hostname" --keylength ec-256 --server letsencrypt
  "$acme" --install-cert -d "$hostname" --ecc --fullchain-file "$cert" --key-file "$key"
  chown root:root "$cert" "$key"; chmod 0644 "$cert"; chmod 0600 "$key"
fi
openssl x509 -in "$cert" -noout -checkhost "$hostname" >/dev/null || fail 'installed certificate does not cover hostname'
vhost_created=0
if [[ -e $vhost || -L $vhost ]]; then
  [[ -f $vhost && ! -L $vhost ]] || fail "Nginx vhost conflicts: $vhost"
  actual=$(sha256sum "$vhost"); actual=${actual%% *}
  [[ $actual == "$expected_vhost_sha" ]] || fail "Nginx vhost conflicts: $vhost"
else
  temporary=$vhost.tmp.$$
  printf '%s' "$vhost_b64" | base64 -d >"$temporary"
  chown root:root "$temporary"; chmod 0644 "$temporary"; mv -T "$temporary" "$vhost"
  vhost_created=1
fi
if ! "$nginx" -t; then
  (( vhost_created )) && rm -f -- "$vhost"
  "$nginx" -t || true
  fail 'nginx -t failed; invocation-created vhost was removed'
fi
if ! service nginx reload; then
  if (( vhost_created )); then rm -f -- "$vhost"; "$nginx" -t && service nginx reload || true; fi
  fail 'Nginx reload failed; invocation-created vhost was removed'
fi
nginx_ready=0
for attempt in 1 2 3 4 5; do
  if "$nginx" -t >/dev/null; then nginx_ready=1; break; fi
  sleep 1
done
(( nginx_ready )) || fail 'Nginx did not become ready after reload'
systemctl enable "$service"
[[ $(systemctl is-enabled "$service" 2>/dev/null || true) == enabled ]] || fail 'systemd service is not enabled'
"""
        self.ssh(s["aliyun_ssh_alias"], script, (
            p["data_root"], p["releases_root"], p["current"], p["systemd_service"],
            p["service_user"], str(p["loopback_port"]), p["environment_file"],
            p["nginx_vhost_path"], p["tls_certificate_path"], p["tls_key_path"],
            p["production_hostname"], s["cloudflare_secret_file"], unit_b64,
            vhost_b64, unit_sha, vhost_sha, ALIYUN_ONEINSTACK_NGINX,
            f"127.0.0.1:{self.cf_socks_aliyun_port}",
        ), stage="infrastructure", timeout=900)
        self.record("infrastructure")

    def configure_playground(self):
        p, s = self.profile, self.shared
        script = r'''set -Eeuo pipefail
vhost=$1; hostname=$2; origin=$3; public=$4; run_id=$5; nginx=$6
backup=$vhost.before-$run_id
[[ -x $nginx ]] || { printf '[first-production:zgocloud] ERROR: missing formal ZgoCloud Nginx executable: %s\n' "$nginx" >&2; exit 1; }
python3 - "$vhost" "$hostname" "$backup" <<'PY'
import os,pathlib,re,shutil,sys
path=pathlib.Path(sys.argv[1]); hostname=sys.argv[2]; backup=pathlib.Path(sys.argv[3])
text=path.read_text(encoding='utf-8')
pattern=r'if \(\$http_origin !~ "\^https://\(([^"()]+)\)\\\.shuijingwanwq\\\.com\$"\) \{'
matches=re.findall(pattern,text)
assert len(matches)==2 and matches[0]==matches[1]
labels=matches[0].split('|'); assert len(labels)==len(set(labels))
suffix='.shuijingwanwq.com'; assert hostname.endswith(suffix)
label=hostname[:-len(suffix)]; assert re.fullmatch(r'[a-z0-9-]+',label)
if label in labels: raise SystemExit(0)
labels.append(label)
replacement='if ($http_origin !~ "^https://('+'|'.join(labels)+r')\.shuijingwanwq\.com$") {'
updated=re.sub(pattern,lambda _: replacement,text)
assert updated != text and len(re.findall(pattern,updated))==2
shutil.copy2(path,backup)
temporary=path.with_name(path.name+f'.tmp-{os.getpid()}')
with open(temporary,'x',encoding='utf-8') as target: target.write(updated)
os.chmod(temporary,0o644); os.replace(temporary,path)
PY
if ! "$nginx" -t; then [[ -f $backup ]] && cp -a "$backup" "$vhost"; "$nginx" -t || true; exit 1; fi
if ! service nginx reload; then [[ -f $backup ]] && cp -a "$backup" "$vhost"; "$nginx" -t && service nginx reload || true; exit 1; fi
check() { expected=$1; shift; for attempt in 1 2 3 4 5; do code=$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 20 "$@" || true); [[ $code == "$expected" ]] && return 0; sleep 1; done; printf 'expected HTTP %s, got %s\n' "$expected" "${code:-000}" >&2; return 1; }
for endpoint in compile fmt; do
  check 204 -X OPTIONS -H "Origin: $origin" "$public/$endpoint"
  check 403 -X OPTIONS -H 'Origin: https://wrong.invalid' "$public/$endpoint"
  check 405 -X GET -H "Origin: $origin" "$public/$endpoint"
done
check 200 -X POST -H "Origin: $origin" --data-urlencode 'version=2' --data-urlencode $'body=package main\nfunc main() {}' "$public/compile"
check 200 -X POST -H "Origin: $origin" --data-urlencode $'body=package main\nfunc main(){println("ok")}' "$public/fmt"
'''
        self.ssh(s["zgocloud_ssh_alias"], script, (
            s["playground_vhost_path"], p["production_hostname"],
            p["playground_allowed_origin"], s["playground_public_origin"], self.run_id,
            ZGOCLOUD_ONEINSTACK_NGINX,
        ), stage="playground-origin", timeout=300)
        self.record("playground-origin")

    def deploy(self):
        self.run([ROOT / "scripts" / "deploy-production.sh", self.release_dir], stage="deploy", timeout=1200)
        self.record("deploy")

    def direct_origin(self):
        p, s = self.profile, self.shared
        script = r'''set -Eeuo pipefail
host=$1; ip=$2; locale=$3; shared_policy=$4; shared_assets=$5
temporary=$(mktemp -d); trap 'rm -rf "$temporary"' EXIT
request() { path=$1; expected=$2; code=$(curl -sS --resolve "$host:443:$ip" --connect-timeout 5 --max-time 20 -o "$temporary/body" -D "$temporary/headers" -w '%{http_code}' "https://$host$path" || true); [[ $code == "$expected" ]] || { printf '%s expected %s got %s\n' "$path" "$expected" "${code:-000}" >&2; exit 1; }; }
for path in / /tour/ /tour/list /tour/welcome/1 /tour/static/js/app.js; do request "$path" 200; done
request /socket 404
request / 200
python3 - "$temporary/body" "$locale" "https://$host/" <<'PY'
from html.parser import HTMLParser
import pathlib,sys
class P(HTMLParser):
  def __init__(self): super().__init__(); self.lang=[]; self.canonical=[]
  def handle_starttag(self,tag,attrs):
    a=dict(attrs)
    if tag=='html': self.lang.append(a.get('lang'))
    if tag=='link' and 'canonical' in a.get('rel','').split(): self.canonical.append(a.get('href'))
p=P(); p.feed(pathlib.Path(sys.argv[1]).read_text(encoding='utf-8'))
assert p.lang == [sys.argv[2]] and p.canonical == [sys.argv[3]]
PY
if [[ $shared_policy == shared-cloudflare ]]; then grep -F "$shared_assets/" "$temporary/body" >/dev/null; fi
code=$(curl -sS --resolve "$host:80:$ip" --connect-timeout 5 --max-time 20 -o /dev/null -D "$temporary/http" -w '%{http_code}' "http://$host/tour/" || true)
[[ $code == 301 || $code == 308 ]]; grep -Eiq "^location: https://$host/tour/" "$temporary/http"
'''
        self.ssh(s["zgocloud_ssh_alias"], script, (
            p["production_hostname"], p["origin_ip"], p["locale"], p["shared_assets_policy"],
            s["shared_assets_public_origin"],
        ), stage="direct-origin", timeout=300)
        self.record("direct-origin")

    def cloudflare_dns(self):
        p, s = self.profile, self.shared
        script = r'''set -Eeuo pipefail
secret=$1; zone=$2; host=$3; ip=$4; cf_socks=$5
set -a; . "$secret"; set +a
[[ -n ${CF_Token:-} ]]
fail() { printf '[first-production:cloudflare-dns] ERROR: %s\n' "$*" >&2; exit 1; }
api=https://api.cloudflare.com/client/v4
readonly CF_CONNECT_TIMEOUT=5 CF_MAX_TIME=20 CF_RETRY_ATTEMPTS=3
cf_tmp=$(mktemp -d); trap 'rm -rf "$cf_tmp"' EXIT
cf_get() {
  output=$1; shift
  for attempt in $(seq 1 "$CF_RETRY_ATTEMPTS"); do
    set +e
    code=$(curl --socks5-hostname "$cf_socks" -sS --connect-timeout "$CF_CONNECT_TIMEOUT" --max-time "$CF_MAX_TIME" -o "$output" -w '%{http_code}' "$@")
    curl_exit=$?
    set -e
    if [[ $curl_exit == 0 && $code == 200 ]]; then return 0; fi
    if [[ $curl_exit != 0 ]]; then
      case $curl_exit in
        6|7|16|28|35) [[ $attempt != "$CF_RETRY_ATTEMPTS" ]] && { sleep "$attempt"; continue; };;
      esac
      fail "Cloudflare API transport failed (curl exit $curl_exit)"
    fi
    fail "Cloudflare API returned HTTP ${code:-000}"
  done
  fail 'Cloudflare API read retry exhausted'
}
dns_state() {
  cf_get "$cf_tmp/records" -H "Authorization: Bearer $CF_Token" -H 'Content-Type: application/json' --get --data-urlencode "name=$host" "$api/zones/$zone_id/dns_records"
  RECORDS_JSON=$(<"$cf_tmp/records") python3 - "$host" "$ip" <<'PY'
import json,os,sys
h,ip=sys.argv[1:]; d=json.loads(os.environ['RECORDS_JSON']); r=d.get('result',[]); assert d.get('success') is True and len(r)<=1
if not r: print('absent')
else:
 x=r[0]; assert x.get('type')=='A' and x.get('name')==h and x.get('content')==ip and x.get('proxied') is True; print('exact')
PY
}
cf_get "$cf_tmp/zone" -H "Authorization: Bearer $CF_Token" -H 'Content-Type: application/json' --get --data-urlencode "name=$zone" --data-urlencode 'status=active' "$api/zones"
zone_json=$(<"$cf_tmp/zone")
zone_id=$(python3 -c 'import json,sys; d=json.load(sys.stdin); r=d.get("result",[]); assert d.get("success") is True and len(r)==1; print(r[0]["id"])' <<<"$zone_json")
payload=$(python3 - "$host" "$ip" <<'PY'
import json,sys
print(json.dumps({'type':'A','name':sys.argv[1],'content':sys.argv[2],'proxied':True},separators=(',',':')))
PY
)
for mutation_attempt in $(seq 1 "$CF_RETRY_ATTEMPTS"); do
  state=$(dns_state) || fail 'Cloudflare DNS identity query failed'
  [[ $state == exact ]] && exit 0
  [[ $state == absent ]] || fail 'Cloudflare DNS identity conflicts'
  set +e
  post_code=$(curl --socks5-hostname "$cf_socks" -sS --connect-timeout "$CF_CONNECT_TIMEOUT" --max-time "$CF_MAX_TIME" -o "$cf_tmp/post" -w '%{http_code}' -X POST -H "Authorization: Bearer $CF_Token" -H 'Content-Type: application/json' --data "$payload" "$api/zones/$zone_id/dns_records")
  post_exit=$?
  set -e
  if [[ $post_exit == 0 && $post_code == 200 ]]; then
    python3 -c 'import json,sys; d=json.load(sys.stdin); assert d.get("success") is True and d.get("result",{}).get("proxied") is True' <"$cf_tmp/post" || fail 'Cloudflare DNS POST semantic response failed'
  elif [[ $post_exit == 0 ]]; then
    fail "Cloudflare DNS POST returned HTTP ${post_code:-000}"
  else
    case $post_exit in
      6|7|16|28|35) : ;; # Result unknown: re-query before any possible next POST.
      *) fail "Cloudflare DNS POST transport failed (curl exit $post_exit)";;
    esac
  fi
  state=$(dns_state) || fail 'Cloudflare DNS identity re-query failed'
  [[ $state == exact ]] && exit 0
  [[ $state == absent ]] || fail 'Cloudflare DNS identity conflicts'
done
fail 'Cloudflare DNS mutation attempts exhausted without an exact identity'
'''
        self.ssh(s["aliyun_ssh_alias"], script, (
            s["cloudflare_secret_file"], s["cloudflare_zone_name"],
            p["production_hostname"], p["origin_ip"], f"127.0.0.1:{self.cf_socks_aliyun_port}",
        ), stage="cloudflare-dns", timeout=300)
        self.record("cloudflare-dns")

    def public_machine(self):
        p, s = self.profile, self.shared
        script = r'''set -Eeuo pipefail
host=$1; locale=$2
temporary=$(mktemp -d); trap 'rm -rf "$temporary"' EXIT
readonly CURL_CONNECT_TIMEOUT=5
readonly CURL_MAX_TIME=20
readonly CURL_RETRY_ATTEMPTS=3
request() {
  expected=$1; body=$2; headers=$3; shift 3
  for request_attempt in $(seq 1 "$CURL_RETRY_ATTEMPTS"); do
    set +e
    code=$(curl -sS --connect-timeout "$CURL_CONNECT_TIMEOUT" --max-time "$CURL_MAX_TIME" -o "$body" -D "$headers" -w '%{http_code}' "$@")
    curl_exit=$?
    set -e
    if [[ $curl_exit != 0 ]]; then
      case $curl_exit in
        6|7|16|28|35)
          if [[ $request_attempt != "$CURL_RETRY_ATTEMPTS" ]]; then sleep "$request_attempt"; continue; fi
          ;;
      esac
      printf 'curl exit %s for %s after %s attempt(s)\n' "$curl_exit" "${!#}" "$request_attempt" >&2
      return 1
    fi
    case $code in
      522|525)
        if [[ $request_attempt != "$CURL_RETRY_ATTEMPTS" ]]; then sleep "$request_attempt"; continue; fi
        ;;
    esac
    [[ $code == "$expected" ]] && return 0
    printf '%s returned HTTP %s (expected %s)\n' "${!#}" "${code:-000}" "$expected" >&2
    return 1
  done
  return 1
}
for readiness_attempt in $(seq 1 30); do
  if request 200 "$temporary/home" "$temporary/home.headers" "https://$host/" && grep -Eq "<html[^>]+lang=[\"']$locale[\"']" "$temporary/home"; then break; fi
  [[ $readiness_attempt != 30 ]] || exit 1
  sleep 2
done
'''
        self.ssh(s["zgocloud_ssh_alias"], script, (
            p["production_hostname"], p["locale"],
        ), stage="public-machine", timeout=1200)
        self.run([ROOT / "scripts" / "verify-production.sh", self.release_dir], stage="public-machine", timeout=1200)
        self.record("public-machine")

    def browser(self):
        self.run([ROOT / "scripts" / "verify-production-browser.py", self.profile["production_public_url"], self.locale], stage="browser", timeout=600)
        self.record("browser")

    def execute(self):
        self.write_receipt()
        self.preflight()
        if not self.stage_passed("infrastructure"):
            self.bootstrap_infrastructure()
        else:
            print("[首次生产] 基础设施：RESUME（预检已重新验证）")
        self.configure_playground()
        if not self.stage_passed("deploy"):
            self.deploy()
        else:
            print("[首次生产] 首次部署：RESUME（current/source health 已重新验证）")
        self.direct_origin()
        self.cloudflare_dns()
        self.public_machine()
        self.browser()
        self.write_receipt("passed")
        print("\n[首次生产] READY FOR HUMAN VISUAL GATE")
        print(f"receipt: {self.receipt_path}")


def usage():
    print("usage: first-production.sh <release-dir>", file=sys.stderr)


def main():
    if len(sys.argv) != 2:
        usage()
        return 2
    orchestrator = None
    try:
        orchestrator = Orchestrator(pathlib.Path(sys.argv[1]))
        orchestrator.execute()
        return 0
    except (FirstProductionError, IDENTITY.IdentityError) as exc:
        if orchestrator is not None:
            orchestrator.receipt["failure"] = {
                "stage": getattr(exc, "stage", "identity"),
                "completed_at": utc_now(),
                "result": "failed",
            }
            orchestrator.write_receipt("failed")
        print("\n[首次生产] FAILED", file=sys.stderr)
        print(f"stage: {getattr(exc, 'stage', 'identity')}", file=sys.stderr)
        print(f"expected: {getattr(exc, 'expected', 'valid formal identity')}", file=sys.stderr)
        print(f"actual: {getattr(exc, 'actual', str(exc))}", file=sys.stderr)
        print(f"下一步：{getattr(exc, 'next_step', '修复 production identity 后重试')}", file=sys.stderr)
        return 1
    finally:
        if orchestrator is not None:
            orchestrator.cleanup()


if __name__ == "__main__":
    raise SystemExit(main())
