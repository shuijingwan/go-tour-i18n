#!/usr/bin/env python3
"""Identity-bound Production CDN authority checks and hostname purge.

The local coordinator accepts only a formal locale.  The same file is sent to
the aliyun root account for secret handling and API calls; credentials never
return to the caller.  Keep this file parseable by aliyun's Python 3.6.
"""

import argparse
import base64
import datetime
import hashlib
import hmac
import importlib.util
import json
import os
import pathlib
import shutil
import socket
import stat
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parent.parent
TENCENT_ENDPOINT = "https://teo.tencentcloudapi.com"
TENCENT_HOST = "teo.tencentcloudapi.com"
TENCENT_SERVICE = "teo"
TENCENT_VERSION = "2022-09-01"
READ_ATTEMPTS = 3
POLL_ATTEMPTS = 10
RECONCILE_ATTEMPTS = 3
RECONCILE_WINDOW_SECONDS = 180


class CDNError(RuntimeError):
    pass


class TransportError(CDNError):
    pass


class MutationUncertain(CDNError):
    pass


def _json_bytes(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True,
                      separators=(",", ":")).encode("utf-8")


def _sha256(value):
    return hashlib.sha256(value).hexdigest()


def tc3_headers(secret_id, secret_key, action, payload, timestamp):
    """Return deterministic Tencent Cloud API 3.0 headers."""
    body = _json_bytes(payload)
    date = datetime.datetime.fromtimestamp(timestamp, datetime.timezone.utc).strftime("%Y-%m-%d")
    content_type = "application/json; charset=utf-8"
    canonical_headers = (
        "content-type:" + content_type + "\n" +
        "host:" + TENCENT_HOST + "\n" +
        "x-tc-action:" + action.lower() + "\n"
    )
    signed_headers = "content-type;host;x-tc-action"
    canonical_request = "POST\n/\n\n%s\n%s\n%s" % (
        canonical_headers, signed_headers, _sha256(body))
    scope = "%s/%s/tc3_request" % (date, TENCENT_SERVICE)
    string_to_sign = "TC3-HMAC-SHA256\n%d\n%s\n%s" % (
        timestamp, scope, _sha256(canonical_request.encode("utf-8")))

    def sign(key, message):
        return hmac.new(key, message.encode("utf-8"), hashlib.sha256).digest()

    secret_date = sign(("TC3" + secret_key).encode("utf-8"), date)
    secret_service = sign(secret_date, TENCENT_SERVICE)
    secret_signing = sign(secret_service, "tc3_request")
    signature = hmac.new(secret_signing, string_to_sign.encode("utf-8"), hashlib.sha256).hexdigest()
    authorization = (
        "TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s" %
        (secret_id, scope, signed_headers, signature)
    )
    return {
        "Authorization": authorization,
        "Content-Type": content_type,
        "Host": TENCENT_HOST,
        "X-TC-Action": action,
        "X-TC-Timestamp": str(timestamp),
        "X-TC-Version": TENCENT_VERSION,
    }, body


class UrllibTransport(object):
    def request(self, method, url, headers, body, timeout):
        request = urllib.request.Request(url, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                raw = response.read()
        except urllib.error.HTTPError as exc:
            raw = exc.read()
            try:
                data = json.loads(raw.decode("utf-8"))
            except (UnicodeError, ValueError):
                raise CDNError("Tencent API returned HTTP %d" % exc.code)
            return data
        except (urllib.error.URLError, OSError) as exc:
            raise TransportError("transport failure: %s" % exc)
        try:
            return json.loads(raw.decode("utf-8"))
        except (UnicodeError, ValueError) as exc:
            raise TransportError("incomplete or invalid JSON response: %s" % exc)


class TencentClient(object):
    def __init__(self, secret_id, secret_key, transport=None, now=None, sleep=None):
        self.secret_id = secret_id
        self.secret_key = secret_key
        self.transport = transport or UrllibTransport()
        self.now = now or time.time
        self.sleep = sleep or time.sleep

    def call(self, action, payload, mutation=False):
        attempts = 1 if mutation else READ_ATTEMPTS
        last = None
        for attempt in range(attempts):
            timestamp = int(self.now())
            headers, body = tc3_headers(self.secret_id, self.secret_key, action, payload, timestamp)
            try:
                data = self.transport.request("POST", TENCENT_ENDPOINT, headers, body, 20)
            except TransportError as exc:
                last = exc
                if mutation:
                    raise MutationUncertain(str(exc))
                if attempt + 1 < attempts:
                    self.sleep(attempt + 1)
                    continue
                raise
            if type(data) is not dict or type(data.get("Response")) is not dict:
                exc = TransportError("missing Tencent Response object")
                if mutation:
                    raise MutationUncertain(str(exc))
                last = exc
                if attempt + 1 < attempts:
                    self.sleep(attempt + 1)
                    continue
                raise exc
            response = data["Response"]
            if "Error" in response:
                error = response.get("Error") or {}
                raise CDNError("Tencent API error %s: %s" %
                               (error.get("Code", "unknown"), error.get("Message", "unknown")))
            return response
        raise last or TransportError("read retry exhausted")


def _one_exact(items, label):
    if type(items) is not list or len(items) != 1 or type(items[0]) is not dict:
        raise CDNError("%s must resolve to exactly one result" % label)
    return items[0]


def _complete_page(response, field, label):
    items = response.get(field)
    total = response.get("TotalCount")
    if type(items) is not list or type(total) is not int or total != len(items):
        raise CDNError("%s response is incomplete or inconsistent" % label)
    return items


def _edgeone_zone_id(zone, zone_name):
    if zone.get("ZoneName") != zone_name:
        raise CDNError("EdgeOne ZoneName does not match the formal zone name")
    zone_id = zone.get("ZoneId")
    if type(zone_id) is not str or not zone_id:
        raise CDNError("EdgeOne ZoneId is missing or empty")
    if (zone.get("ActiveStatus") != "active" or zone.get("Paused") is not False or
            zone.get("LockStatus") != "enable"):
        raise CDNError("EdgeOne zone is inactive, paused, or locked")
    zone_type = zone.get("Type")
    if zone_type == "partial":
        if zone.get("Status") not in ("active", "pending") or zone.get("CnameStatus") != "finished":
            raise CDNError("EdgeOne partial zone has invalid NS or CNAME status")
    elif zone_type == "full":
        if zone.get("Status") != "active":
            raise CDNError("EdgeOne full zone NS status is not active")
    else:
        raise CDNError("unsupported EdgeOne zone Type: %r" % zone_type)
    return zone_id


def resolve_edgeone_identity(client, zone_name, hostname):
    zone_response = client.call("DescribeZones", {
        "Filters": [{"Name": "zone-name", "Values": [zone_name], "Fuzzy": False}],
        "Limit": 100,
        "Offset": 0,
    })
    zone = _one_exact(_complete_page(zone_response, "Zones", "EdgeOne zone"), "EdgeOne zone")
    zone_id = _edgeone_zone_id(zone, zone_name)
    domain_response = client.call("DescribeAccelerationDomains", {
        "ZoneId": zone_id,
        "Filters": [{"Name": "domain-name", "Values": [hostname], "Fuzzy": False}],
        "Limit": 100,
        "Offset": 0,
    })
    domains = _complete_page(domain_response, "AccelerationDomains", "EdgeOne acceleration domain")
    domain = _one_exact(domains, "EdgeOne acceleration domain")
    if (domain.get("ZoneId") != zone_id or domain.get("DomainName") != hostname or
            domain.get("DomainStatus") != "online"):
        raise CDNError("EdgeOne acceleration domain identity/status is not exact and online")
    return zone_id, zone_response.get("RequestId"), domain_response.get("RequestId")


def _parse_time(value):
    if type(value) is not str:
        return None
    try:
        normalized = value[:-1] + "+00:00" if value.endswith("Z") else value
        parsed = datetime.datetime.strptime(normalized[:19], "%Y-%m-%dT%H:%M:%S")
        return int((parsed - datetime.datetime(1970, 1, 1)).total_seconds())
    except (ValueError, TypeError):
        return None


def _task_state(task, job_id, hostname):
    if (type(job_id) is not str or not job_id or type(task) is not dict or task.get("JobId") != job_id or
            task.get("Type") != "purge_host" or task.get("Target") != hostname):
        raise CDNError("EdgeOne purge task identity mismatch")
    status = task.get("Status")
    if status not in ("processing", "success", "failed", "timeout", "canceled"):
        raise CDNError("unknown EdgeOne purge task status: %r" % status)
    return status


def poll_edgeone_task(client, zone_id, job_id, hostname):
    payload = {
        "ZoneId": zone_id,
        "Filters": [{"Name": "job-id", "Values": [job_id], "Fuzzy": False}],
        "Limit": 2,
        "Offset": 0,
    }
    for attempt in range(POLL_ATTEMPTS):
        response = client.call("DescribePurgeTasks", payload)
        task = _one_exact(_complete_page(response, "Tasks", "EdgeOne purge JobId"), "EdgeOne purge JobId")
        status = _task_state(task, job_id, hostname)
        if status == "success":
            return response.get("RequestId")
        if status != "processing":
            raise CDNError("EdgeOne purge task ended with status %s" % status)
        if attempt + 1 < POLL_ATTEMPTS:
            client.sleep(2)
    raise CDNError("EdgeOne purge task polling exhausted")


def reconcile_edgeone_task(client, zone_id, hostname, started_at):
    start = started_at - RECONCILE_WINDOW_SECONDS
    end = int(client.now()) + RECONCILE_WINDOW_SECONDS
    payload = {
        "ZoneId": zone_id,
        "StartTime": datetime.datetime.fromtimestamp(start, datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "EndTime": datetime.datetime.fromtimestamp(end, datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "Filters": [
            {"Name": "domains", "Values": [hostname], "Fuzzy": False},
            {"Name": "type", "Values": ["purge_host"], "Fuzzy": False},
        ],
        "Limit": 100,
        "Offset": 0,
    }
    for attempt in range(RECONCILE_ATTEMPTS):
        response = client.call("DescribePurgeTasks", payload)
        tasks = _complete_page(response, "Tasks", "EdgeOne reconciliation")
        matches = []
        for task in tasks:
            created = _parse_time(task.get("CreateTime") if type(task) is dict else None)
            if (type(task) is dict and task.get("Type") == "purge_host" and
                    task.get("Target") == hostname and created is not None and start <= created <= end):
                matches.append(task)
        if len(matches) == 1:
            task = matches[0]
            _task_state(task, task.get("JobId"), hostname)
            return task
        if len(matches) > 1:
            raise CDNError("EdgeOne reconciliation found ambiguous matching purge tasks")
        if attempt + 1 < RECONCILE_ATTEMPTS:
            client.sleep(2)
    return None


def purge_edgeone(client, zone_id, hostname):
    payload = {"ZoneId": zone_id, "Type": "purge_host", "Method": "delete", "Targets": [hostname]}
    for mutation_attempt in range(2):
        started_at = int(client.now())
        try:
            response = client.call("CreatePurgeTask", payload, mutation=True)
        except MutationUncertain:
            task = reconcile_edgeone_task(client, zone_id, hostname, started_at)
            if task is not None:
                status = _task_state(task, task.get("JobId"), hostname)
                if status == "success":
                    return None
                if status != "processing":
                    raise CDNError("EdgeOne reconciled purge task ended with status %s" % status)
                return poll_edgeone_task(client, zone_id, task.get("JobId"), hostname)
            if mutation_attempt == 0:
                continue
            raise CDNError("EdgeOne mutation result remains unresolved after reconciliation")
        failed = response.get("FailedList")
        job_id = response.get("JobId")
        if type(failed) is not list or failed:
            raise CDNError("EdgeOne CreatePurgeTask FailedList is not empty")
        if type(job_id) is not str or not job_id:
            raise CDNError("EdgeOne CreatePurgeTask returned no JobId")
        return poll_edgeone_task(client, zone_id, job_id, hostname)
    raise CDNError("EdgeOne purge did not reach a terminal result")


class CloudflareClient(object):
    def __init__(self, token, transport, sleep=None):
        self.token = token
        self.transport = transport
        self.sleep = sleep or time.sleep
        self.base = "https://api.cloudflare.com/client/v4"

    def _request(self, method, path, payload=None, mutation=False):
        attempts = 1 if mutation else READ_ATTEMPTS
        body = None if payload is None else _json_bytes(payload)
        headers = {"Authorization": "Bearer " + self.token, "Content-Type": "application/json"}
        for attempt in range(attempts):
            try:
                data = self.transport.request(method, self.base + path, headers, body, 20)
            except TransportError as exc:
                if mutation:
                    raise MutationUncertain(str(exc))
                if attempt + 1 < attempts:
                    self.sleep(attempt + 1)
                    continue
                raise
            if type(data) is not dict:
                if mutation:
                    raise MutationUncertain("Cloudflare mutation response is incomplete")
                raise CDNError("Cloudflare response is not an object")
            if data.get("success") is not True:
                raise CDNError("Cloudflare API reported a definite failure")
            return data
        raise CDNError("Cloudflare read retry exhausted")

    def resolve_zone(self, zone_name):
        query = urllib.parse.urlencode({"name": zone_name, "status": "active"})
        result = self._request("GET", "/zones?" + query).get("result")
        zone = _one_exact(result, "Cloudflare zone")
        if not zone.get("id") or zone.get("name") not in (None, zone_name):
            raise CDNError("Cloudflare zone identity mismatch")
        return zone["id"]

    def purge_hostname(self, zone_id, hostname):
        try:
            response = self._request("POST", "/zones/%s/purge_cache" % zone_id,
                                     {"hosts": [hostname]}, mutation=True)
        except MutationUncertain:
            # Cloudflare exposes no formal purge-task query for this operation.
            raise CDNError("Cloudflare purge result is uncertain; no duplicate mutation was attempted")
        if type(response.get("result")) is not dict:
            raise CDNError("Cloudflare purge response has no result object")


class CurlTransport(object):
    def __init__(self, socks):
        self.socks = socks

    def request(self, method, url, headers, body, timeout):
        header_file = tempfile.NamedTemporaryFile(mode="w", delete=False)
        try:
            os.chmod(header_file.name, 0o600)
            for name, value in headers.items():
                header_file.write("%s: %s\n" % (name, value))
            header_file.close()
            command = ["curl", "--socks5-hostname", self.socks, "-sS", "--connect-timeout", "5",
                       "--max-time", str(timeout), "-X", method, "-H", "@" + header_file.name,
                       "-w", "\n%{http_code}", url]
            if body is not None:
                command[-1:-1] = ["--data-binary", "@-"]
            completed = subprocess.run(command, input=body, stdout=subprocess.PIPE,
                                       stderr=subprocess.PIPE, check=False)
            if completed.returncode != 0:
                raise TransportError("curl exit %d" % completed.returncode)
            raw, separator, status = completed.stdout.rpartition(b"\n")
            if not separator or status != b"200":
                raise CDNError("Cloudflare API returned HTTP %s" % status.decode("ascii", "replace"))
            try:
                return json.loads(raw.decode("utf-8"))
            except (UnicodeError, ValueError) as exc:
                raise TransportError("incomplete or invalid JSON response: %s" % exc)
        finally:
            try:
                os.unlink(header_file.name)
            except OSError:
                pass


def read_secret(path, required, exact=False):
    info = os.lstat(path)
    if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_gid != 0 or stat.S_IMODE(info.st_mode) != 0o600:
        raise CDNError("secret file must be a root:root 0600 regular file")
    values = {}
    with open(path, encoding="utf-8") as source:
        lines = source.read().splitlines()
    for line in lines:
        if not line or line.lstrip().startswith("#"):
            continue
        if "=" not in line:
            raise CDNError("secret file contains an invalid assignment")
        name, value = line.split("=", 1)
        if name in values or not name:
            raise CDNError("secret file contains a duplicate or invalid name")
        values[name] = value.strip().strip("'\"")
    if exact and set(values) != set(required):
        raise CDNError("secret file keys do not match the formal contract")
    for name in required:
        if not values.get(name):
            raise CDNError("secret file is missing required non-empty variables")
    return [values[name] for name in required]


def remote_main(encoded):
    config = json.loads(base64.b64decode(encoded).decode("utf-8"))
    provider = config["provider"]
    action = config["action"]
    hostname = config["hostname"]
    if provider == "edgeone":
        secret_id, secret_key = read_secret(config["secret_file"],
                                            ("TENCENTCLOUD_SECRET_ID", "TENCENTCLOUD_SECRET_KEY"), exact=True)
        client = TencentClient(secret_id, secret_key)
        zone_id, zone_request, domain_request = resolve_edgeone_identity(client, config["zone_name"], hostname)
        if action == "preflight":
            print("EDGEONE AUTHORITY PREFLIGHT: PASS")
            print("zone_name: %s" % config["zone_name"])
            print("zone_id: %s" % zone_id)
            print("hostname: %s" % hostname)
            if zone_request or domain_request:
                print("request_id: %s" % (domain_request or zone_request))
            return 0
        purge_edgeone(client, zone_id, hostname)
        print("EDGEONE HOSTNAME PURGE: PASS")
        return 0
    token, = read_secret(config["secret_file"], ("CF_Token",))
    client = CloudflareClient(token, CurlTransport(config["socks"]))
    zone_id = client.resolve_zone(config["zone_name"])
    if action == "preflight":
        print("CLOUDFLARE AUTHORITY PREFLIGHT: PASS")
        print("zone_name: %s" % config["zone_name"])
        print("hostname: %s" % hostname)
        return 0
    client.purge_hostname(zone_id, hostname)
    print("CLOUDFLARE HOSTNAME PURGE: PASS")
    return 0


def _load_identity():
    spec = importlib.util.spec_from_file_location("production_identity_cdn", ROOT / "scripts" / "production-identity.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.load_identity(ROOT / "production" / "identity.json")


def _free_port():
    listener = socket.socket()
    try:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]
    finally:
        listener.close()


def coordinator(action, locale):
    identity = _load_identity()
    profiles = [profile for profile in identity["locales"] if profile["locale"] == locale]
    if len(profiles) != 1:
        raise CDNError("locale has no unique formal production identity")
    profile = profiles[0]
    if profile["production_state"] != "live":
        raise CDNError("CDN maintenance requires production_state=live")
    shared = identity["shared"]
    config = {
        "action": action,
        "provider": profile["cdn"],
        "hostname": profile["production_hostname"],
    }
    if profile["cdn"] == "edgeone":
        config.update(zone_name=shared["edgeone_zone_name"], secret_file=shared["edgeone_secret_file"])
    elif profile["cdn"] == "cloudflare":
        config.update(zone_name=shared["cloudflare_zone_name"], secret_file=shared["cloudflare_secret_file"])
    else:
        raise CDNError("unsupported formal CDN")
    temp = pathlib.Path(tempfile.mkdtemp(prefix="go-tour-production-cdn-"))
    controls = []
    base = ["-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "ServerAliveInterval=5",
            "-o", "ServerAliveCountMax=3", "-o", "ConnectionAttempts=3"]
    aliyun = shared["aliyun_ssh_alias"]
    try:
        aliyun_options = list(base)
        if profile["cdn"] == "cloudflare":
            local_port, aliyun_port = _free_port(), _free_port()
            zcontrol = temp / "zgocloud.control"
            acontrol = temp / "aliyun.control"
            zoptions = base + ["-o", "ControlMaster=yes", "-o", "ControlPersist=yes", "-o", "ExitOnForwardFailure=yes", "-o", "ControlPath=" + str(zcontrol)]
            aliyun_options = base + ["-o", "ControlMaster=yes", "-o", "ControlPersist=yes", "-o", "ExitOnForwardFailure=yes", "-o", "GatewayPorts=no", "-o", "ControlPath=" + str(acontrol)]
            subprocess.run(["ssh"] + zoptions + ["-f", "-N", "-D", "127.0.0.1:%d" % local_port, shared["zgocloud_ssh_alias"]], check=True, timeout=30)
            controls.append((shared["zgocloud_ssh_alias"], zoptions))
            subprocess.run(["ssh"] + aliyun_options + ["-f", "-N", "-R", "127.0.0.1:%d:127.0.0.1:%d" % (aliyun_port, local_port), aliyun], check=True, timeout=30)
            controls.append((aliyun, aliyun_options))
            config["socks"] = "127.0.0.1:%d" % aliyun_port
        encoded = base64.b64encode(_json_bytes(config)).decode("ascii")
        source = pathlib.Path(__file__).read_text(encoding="utf-8")
        result = subprocess.run(["ssh"] + aliyun_options + [aliyun, "python3", "-", "--remote", encoded],
                                input=source, text=True, check=False, timeout=300)
        if result.returncode != 0:
            raise CDNError("remote CDN %s failed (exit %d)" % (action, result.returncode))
    finally:
        for host, options in reversed(controls):
            subprocess.run(["ssh"] + options + ["-O", "exit", host], stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL, check=False)
        shutil.rmtree(str(temp), ignore_errors=True)


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    if len(argv) == 2 and argv[0] == "--remote":
        try:
            return remote_main(argv[1])
        except (CDNError, OSError, ValueError, KeyError) as exc:
            print("[production-cdn] FAILED: %s" % exc, file=sys.stderr)
            return 1
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=("preflight", "purge"))
    parser.add_argument("--locale", required=True)
    args = parser.parse_args(argv)
    try:
        coordinator(args.action, args.locale)
        return 0
    except (CDNError, OSError, subprocess.SubprocessError, ValueError, KeyError) as exc:
        print("[production-cdn] FAILED: %s" % exc, file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
