#!/usr/bin/env python3

import base64
import ast
import hashlib
import importlib.util
import io
import json
import os
import pathlib
import stat
import tempfile
import unittest
from contextlib import redirect_stdout
from unittest import mock


ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location("production_cdn", ROOT / "scripts" / "production-cdn.py")
CDN = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CDN)


class QueueTransport(object):
    def __init__(self, responses):
        self.responses = list(responses)
        self.calls = []

    def request(self, method, url, headers, body, timeout):
        self.calls.append((method, url, headers, json.loads(body.decode()) if body else None))
        response = self.responses.pop(0)
        if isinstance(response, Exception):
            raise response
        return response


def response(**values):
    return {"Response": values}


class FakeTencent(object):
    def __init__(self, calls, times=None):
        self.calls = list(calls)
        self.seen = []
        self.times = list(times or [1700000000] * 20)
        self.sleeps = []

    def call(self, action, payload, mutation=False):
        self.seen.append((action, payload, mutation))
        expected_action, value = self.calls.pop(0)
        if action != expected_action:
            raise AssertionError("got %s want %s" % (action, expected_action))
        if isinstance(value, Exception):
            raise value
        if type(value) is dict and action.startswith("Describe") and "TotalCount" not in value:
            value = dict(value)
            field = {"DescribeZones": "Zones", "DescribeAccelerationDomains": "AccelerationDomains",
                     "DescribePurgeTasks": "Tasks"}[action]
            value["TotalCount"] = len(value.get(field, []))
        return value

    def now(self):
        return self.times.pop(0) if self.times else 1700000000

    def sleep(self, seconds):
        self.sleeps.append(seconds)


class ProductionCDNTest(unittest.TestCase):
    def shared_receipt(self, directory, changed=None, base="https://assets-go-dev.shuijingwanwq.com",
                       schema=None, purge="PENDING", verification="PENDING"):
        export = pathlib.Path(directory) / "shared-export"
        export.mkdir()
        (export / "SHA256SUMS").write_text("formal manifest\n", encoding="utf-8")
        digest = hashlib.sha256((export / "SHA256SUMS").read_bytes()).hexdigest()
        receipt = pathlib.Path(str(export) + ".verification-receipt.json")
        value = {
            "schema": schema or CDN.SHARED_ASSETS_RECEIPT_V1,
            "export_dir": str(export),
            "manifest_sha256": digest,
            "deployment_result": "DEPLOYED" if changed else "NO_CHANGES",
            "production_base_url": base,
            "changed_paths": changed or [],
            "boundary_paths": CDN.SHARED_ASSETS_BOUNDARY_PATHS,
        }
        if schema == CDN.SHARED_ASSETS_RECEIPT_V2:
            value.update(purge_result=purge, verification_result=verification)
        receipt.write_text(json.dumps(value), encoding="utf-8")
        return receipt

    def test_remote_helper_is_python_36_parseable(self):
        source = (ROOT / "scripts" / "production-cdn.py").read_text(encoding="utf-8")
        ast.parse(source, feature_version=(3, 6))

    def test_tc3_signing_is_deterministic_and_secret_is_not_logged(self):
        headers, body = CDN.tc3_headers("AKIDEXAMPLE", "SECRET", "DescribeZones", {"Limit": 1}, 1700000000)
        self.assertEqual(body, b'{"Limit":1}')
        self.assertEqual(headers["Authorization"], "TC3-HMAC-SHA256 Credential=AKIDEXAMPLE/2023-11-14/teo/tc3_request, SignedHeaders=content-type;host;x-tc-action, Signature=27f223eb9ec4cc1d68a46bb65c9b98be9312627508f39cf40c437e2d134c12d6")
        transport = QueueTransport([CDN.TransportError("timeout")] * CDN.READ_ATTEMPTS)
        client = CDN.TencentClient("DO-NOT-LOG-ID", "DO-NOT-LOG-KEY", transport=transport, now=lambda: 1700000000, sleep=lambda _: None)
        with self.assertRaises(CDN.TransportError) as raised:
            client.call("DescribeZones", {})
        self.assertNotIn("DO-NOT-LOG", str(raised.exception))

    def identity_client(self, zones, domains):
        return FakeTencent([
            ("DescribeZones", {"Zones": zones, "RequestId": "zone-request"}),
            ("DescribeAccelerationDomains", {"AccelerationDomains": domains, "RequestId": "domain-request"}),
        ])

    def zone(self, **changes):
        value = {
            "ZoneId": "zone-1",
            "ZoneName": "example.com",
            "Type": "partial",
            "Status": "pending",
            "CnameStatus": "finished",
            "ActiveStatus": "active",
            "LockStatus": "enable",
            "Paused": False,
        }
        value.update(changes)
        return value

    def test_edgeone_partial_pending_exact_identity_success(self):
        client = self.identity_client(
            [self.zone()],
            [{"ZoneId": "zone-1", "DomainName": "www.example.com", "DomainStatus": "online"}],
        )
        self.assertEqual(CDN.resolve_edgeone_identity(client, "example.com", "www.example.com"),
                         ("zone-1", "zone-request", "domain-request"))
        self.assertEqual(client.seen[0][1]["Filters"], [{"Name": "zone-name", "Values": ["example.com"], "Fuzzy": False}])
        self.assertEqual(client.seen[1][1]["Filters"], [{"Name": "domain-name", "Values": ["www.example.com"], "Fuzzy": False}])

    def test_edgeone_zone_identity_failures(self):
        cases = (
            [],
            [self.zone(ZoneId="a"), self.zone(ZoneId="b")],
            [self.zone(ZoneName="wrong.example")],
            [self.zone(ZoneId="")],
            [self.zone(ZoneId=None)],
            [self.zone(CnameStatus="pending")],
            [self.zone(ActiveStatus="inactive")],
            [self.zone(ActiveStatus="paused")],
            [self.zone(Paused=True)],
            [self.zone(LockStatus="disable")],
            [self.zone(Type="unknown")],
            [self.zone(Status="unknown")],
        )
        for zones in cases:
            with self.subTest(zones=zones):
                with self.assertRaises(CDN.CDNError):
                    CDN.resolve_edgeone_identity(self.identity_client(zones, []), "example.com", "www.example.com")

    def test_edgeone_full_zone_requires_active_ns_and_ignores_cname_status(self):
        domain = {"ZoneId": "zone-1", "DomainName": "www.example.com", "DomainStatus": "online"}
        client = self.identity_client([self.zone(Type="full", Status="active", CnameStatus="pending")], [domain])
        self.assertEqual(CDN.resolve_edgeone_identity(client, "example.com", "www.example.com")[0], "zone-1")
        with self.assertRaisesRegex(CDN.CDNError, "NS status is not active"):
            CDN.resolve_edgeone_identity(
                self.identity_client([self.zone(Type="full", Status="pending", CnameStatus="finished")], []),
                "example.com", "www.example.com")

    def test_edgeone_domain_identity_failures(self):
        zone = [self.zone()]
        valid = {"ZoneId": "zone-1", "DomainName": "www.example.com", "DomainStatus": "online"}
        cases = ([], [valid, valid], [dict(valid, DomainName="wrong.example")],
                 [dict(valid, ZoneId="zone-2")], [dict(valid, DomainStatus="offline")])
        for domains in cases:
            with self.subTest(domains=domains):
                with self.assertRaises(CDN.CDNError):
                    CDN.resolve_edgeone_identity(self.identity_client(zone, domains), "example.com", "www.example.com")

    def test_edgeone_incomplete_or_inconsistent_pages_fail_closed(self):
        client = FakeTencent([
            ("DescribeZones", {"Zones": [self.zone()],
                               "TotalCount": 2}),
        ])
        with self.assertRaisesRegex(CDN.CDNError, "incomplete or inconsistent"):
            CDN.resolve_edgeone_identity(client, "example.com", "www.example.com")

    def task(self, status="success", job="job-1", created="2023-11-14T22:13:20Z"):
        return {"JobId": job, "Status": status, "Target": "www.example.com", "Type": "purge_host", "CreateTime": created}

    def test_edgeone_purge_exact_payload_and_eventual_visibility_success(self):
        client = FakeTencent([
            ("CreatePurgeTask", {"JobId": "job-1", "FailedList": [], "RequestId": "create"}),
            ("DescribePurgeTasks", {"Tasks": []}),
            ("DescribePurgeTasks", {"Tasks": [self.task("processing")]}),
            ("DescribePurgeTasks", {"Tasks": [self.task("success")], "RequestId": "done"}),
        ])
        self.assertEqual(CDN.purge_edgeone(client, "zone-1", "www.example.com"), "done")
        self.assertEqual([action for action, _, _ in client.seen].count("CreatePurgeTask"), 1)
        self.assertEqual([action for action, _, _ in client.seen].count("DescribePurgeTasks"), 3)
        self.assertEqual(client.sleeps, [2, 2])
        action, payload, mutation = client.seen[0]
        self.assertEqual((action, mutation), ("CreatePurgeTask", True))
        self.assertEqual(payload, {"ZoneId": "zone-1", "Type": "purge_host", "Method": "delete", "Targets": ["www.example.com"]})
        rendered = json.dumps(payload)
        self.assertNotIn("purge_all", rendered)
        self.assertNotIn("*", rendered)
        self.assertNotIn("invalidate", rendered)

    def test_edgeone_purge_job_visibility_polling_exhausts_without_duplicate_create(self):
        calls = [("CreatePurgeTask", {"JobId": "job-1", "FailedList": []})]
        calls += [("DescribePurgeTasks", {"Tasks": []})] * CDN.POLL_ATTEMPTS
        client = FakeTencent(calls)
        with self.assertRaisesRegex(CDN.CDNError, "did not become visible before polling exhausted"):
            CDN.purge_edgeone(client, "zone-1", "www.example.com")
        self.assertEqual([action for action, _, _ in client.seen].count("CreatePurgeTask"), 1)
        self.assertEqual([action for action, _, _ in client.seen].count("DescribePurgeTasks"), CDN.POLL_ATTEMPTS)
        self.assertEqual(client.sleeps, [2] * (CDN.POLL_ATTEMPTS - 1))

    def test_edgeone_purge_job_ambiguity_fails_immediately(self):
        client = FakeTencent([
            ("CreatePurgeTask", {"JobId": "job-1", "FailedList": []}),
            ("DescribePurgeTasks", {"Tasks": [self.task(), self.task()]}),
        ])
        with self.assertRaisesRegex(CDN.CDNError, "must resolve to exactly one result"):
            CDN.purge_edgeone(client, "zone-1", "www.example.com")
        self.assertEqual([action for action, _, _ in client.seen].count("DescribePurgeTasks"), 1)
        self.assertEqual(client.sleeps, [])

    def test_edgeone_purge_job_inconsistent_page_fails_immediately(self):
        client = FakeTencent([
            ("CreatePurgeTask", {"JobId": "job-1", "FailedList": []}),
            ("DescribePurgeTasks", {"Tasks": [], "TotalCount": 1}),
        ])
        with self.assertRaisesRegex(CDN.CDNError, "incomplete or inconsistent"):
            CDN.purge_edgeone(client, "zone-1", "www.example.com")
        self.assertEqual([action for action, _, _ in client.seen].count("DescribePurgeTasks"), 1)
        self.assertEqual(client.sleeps, [])

    def test_edgeone_create_response_failures(self):
        for result in ({"JobId": "", "FailedList": []}, {"JobId": "job", "FailedList": ["bad"]}, {"JobId": "job"}):
            with self.subTest(result=result), self.assertRaises(CDN.CDNError):
                CDN.purge_edgeone(FakeTencent([("CreatePurgeTask", result)]), "zone-1", "www.example.com")

    def test_edgeone_terminal_and_unknown_statuses_fail_closed(self):
        for status in ("failed", "timeout", "canceled", "unknown"):
            calls = [("CreatePurgeTask", {"JobId": "job-1", "FailedList": []}),
                     ("DescribePurgeTasks", {"Tasks": [self.task(status)]})]
            with self.subTest(status=status), self.assertRaises(CDN.CDNError):
                CDN.purge_edgeone(FakeTencent(calls), "zone-1", "www.example.com")

    def test_edgeone_uncertain_create_reconciles_unique_without_duplicate(self):
        client = FakeTencent([
            ("CreatePurgeTask", CDN.MutationUncertain("timeout")),
            ("DescribePurgeTasks", {"Tasks": [self.task("processing")]}),
            ("DescribePurgeTasks", {"Tasks": [self.task("success")]}),
        ])
        CDN.purge_edgeone(client, "zone-1", "www.example.com")
        self.assertEqual([action for action, _, _ in client.seen].count("CreatePurgeTask"), 1)
        filters = client.seen[1][1]["Filters"]
        self.assertIn({"Name": "domains", "Values": ["www.example.com"], "Fuzzy": False}, filters)
        self.assertIn({"Name": "type", "Values": ["purge_host"], "Fuzzy": False}, filters)

    def test_edgeone_uncertain_create_reconciles_terminal_task_without_duplicate(self):
        for status in ("success", "failed", "timeout", "canceled"):
            with self.subTest(status=status):
                client = FakeTencent([
                    ("CreatePurgeTask", CDN.MutationUncertain("timeout")),
                    ("DescribePurgeTasks", {"Tasks": [self.task(status)]}),
                ])
                if status == "success":
                    CDN.purge_edgeone(client, "zone-1", "www.example.com")
                else:
                    with self.assertRaisesRegex(CDN.CDNError, status):
                        CDN.purge_edgeone(client, "zone-1", "www.example.com")
                self.assertEqual([action for action, _, _ in client.seen].count("CreatePurgeTask"), 1)

    def test_edgeone_uncertain_create_ambiguous_or_unresolved_fails_closed(self):
        ambiguous = FakeTencent([
            ("CreatePurgeTask", CDN.MutationUncertain("timeout")),
            ("DescribePurgeTasks", {"Tasks": [self.task(job="one"), self.task(job="two")]}),
        ])
        with self.assertRaisesRegex(CDN.CDNError, "ambiguous"):
            CDN.purge_edgeone(ambiguous, "zone-1", "www.example.com")
        self.assertEqual([x[0] for x in ambiguous.seen].count("CreatePurgeTask"), 1)

        calls = [("CreatePurgeTask", CDN.MutationUncertain("timeout"))]
        calls += [("DescribePurgeTasks", {"Tasks": []})] * CDN.RECONCILE_ATTEMPTS
        calls += [("CreatePurgeTask", CDN.MutationUncertain("timeout"))]
        calls += [("DescribePurgeTasks", {"Tasks": []})] * CDN.RECONCILE_ATTEMPTS
        unresolved = FakeTencent(calls)
        with self.assertRaisesRegex(CDN.CDNError, "unresolved"):
            CDN.purge_edgeone(unresolved, "zone-1", "www.example.com")
        self.assertEqual([x[0] for x in unresolved.seen].count("CreatePurgeTask"), 2)

    def test_cloudflare_exact_hostname_payload_and_uncertain_no_retry(self):
        transport = QueueTransport([
            {"success": True, "result": [{"id": "zone-1", "name": "example.com"}]},
            {"success": True, "result": {}},
        ])
        client = CDN.CloudflareClient("TOKEN", transport, sleep=lambda _: None)
        zone = client.resolve_zone("example.com")
        client.purge_hostname(zone, "www.example.com")
        method, url, _, payload = transport.calls[-1]
        self.assertEqual((method, payload), ("POST", {"hosts": ["www.example.com"]}))
        self.assertNotIn("purge_everything", json.dumps(payload).lower())

        uncertain = QueueTransport([CDN.TransportError("timeout")])
        with self.assertRaisesRegex(CDN.CDNError, "no duplicate"):
            CDN.CloudflareClient("TOKEN", uncertain).purge_hostname("zone", "www.example.com")
        self.assertEqual(len(uncertain.calls), 1)

    def test_cloudflare_exact_url_payload_uses_official_response_shape(self):
        official_success = {"success": True, "errors": [], "messages": [],
                            "result": {"id": "023e105f4ecef8ad9ca31a8372d0c353"}}
        transport = QueueTransport([official_success])
        urls = ["https://assets-go-dev.shuijingwanwq.com/SHA256SUMS",
                "https://assets-go-dev.shuijingwanwq.com/tour/static/css/app.css"]
        CDN.CloudflareClient("TOKEN", transport).purge_files("zone-1", urls)
        method, path, _, payload = transport.calls[0]
        self.assertEqual(method, "POST")
        self.assertTrue(path.endswith("/zones/zone-1/purge_cache"))
        self.assertEqual(payload, {"files": urls})
        for forbidden in ("purge_everything", "hosts", "prefixes", "tags"):
            self.assertNotIn(forbidden, payload)

    def test_cloudflare_exact_url_definite_and_uncertain_failures_do_not_retry(self):
        definite = QueueTransport([
            {"success": False, "errors": [{"code": 1000, "message": "denied"}], "messages": [], "result": None},
        ])
        with self.assertRaises(CDN.CDNError) as raised:
            CDN.CloudflareClient("DO-NOT-LOG", definite).purge_files(
                "zone-1", ["https://assets-go-dev.shuijingwanwq.com/SHA256SUMS"])
        self.assertNotIsInstance(raised.exception, CDN.MutationUncertain)
        self.assertEqual(len(definite.calls), 1)

        for response_value in (CDN.TransportError("timeout"),
                               {"success": True, "errors": [], "messages": [], "result": {}}):
            with self.subTest(response=response_value):
                transport = QueueTransport([response_value])
                with self.assertRaises(CDN.MutationUncertain):
                    CDN.CloudflareClient("DO-NOT-LOG", transport).purge_files(
                        "zone-1", ["https://assets-go-dev.shuijingwanwq.com/SHA256SUMS"])
                self.assertEqual(len(transport.calls), 1)

    def test_shared_assets_receipt_constructs_only_formal_exact_urls(self):
        shared = CDN._load_identity()["shared"]
        with tempfile.TemporaryDirectory() as directory:
            receipt = self.shared_receipt(directory, ["SHA256SUMS", "tour/static/css/app.css"])
            _, urls = CDN.load_shared_assets_receipt(receipt, shared)
            self.assertEqual(urls, [
                "https://assets-go-dev.shuijingwanwq.com/SHA256SUMS",
                "https://assets-go-dev.shuijingwanwq.com/tour/static/css/app.css",
            ])

    def test_shared_assets_receipt_rejects_arbitrary_origin_and_unsafe_paths(self):
        shared = CDN._load_identity()["shared"]
        with tempfile.TemporaryDirectory() as directory:
            receipt = self.shared_receipt(directory, ["tour/static/css/app.css"], "https://evil.example")
            with self.assertRaises(CDN.CDNError):
                CDN.load_shared_assets_receipt(receipt, shared)
        for unsafe in ("../outside", "/absolute", "safe.css?x=1", "https://evil.example/a.css", "a//b"):
            with self.subTest(path=unsafe), tempfile.TemporaryDirectory() as directory:
                receipt = self.shared_receipt(directory, [unsafe])
                with self.assertRaises(CDN.CDNError):
                    CDN.load_shared_assets_receipt(receipt, shared)

    def test_shared_assets_definite_failure_resumes_but_uncertain_never_repeats(self):
        with tempfile.TemporaryDirectory() as directory:
            receipt = self.shared_receipt(directory, ["SHA256SUMS"], schema=CDN.SHARED_ASSETS_RECEIPT_V2)
            with mock.patch.object(CDN, "_run_remote_config", side_effect=CDN.CDNError("denied")) as remote:
                with self.assertRaisesRegex(CDN.CDNError, "denied"):
                    CDN.shared_assets_coordinator("purge-shared-assets", receipt)
            self.assertEqual(json.loads(receipt.read_text(encoding="utf-8"))["purge_result"], "PENDING")
            with mock.patch.object(CDN, "_run_remote_config") as remote:
                CDN.shared_assets_coordinator("purge-shared-assets", receipt)
            self.assertEqual(remote.call_count, 1)
            self.assertEqual(json.loads(receipt.read_text(encoding="utf-8"))["purge_result"], "PASS")

        with tempfile.TemporaryDirectory() as directory:
            receipt = self.shared_receipt(directory, ["SHA256SUMS"], schema=CDN.SHARED_ASSETS_RECEIPT_V2)
            with mock.patch.object(CDN, "_run_remote_config", side_effect=CDN.MutationUncertain("timeout")) as remote:
                with self.assertRaises(CDN.MutationUncertain):
                    CDN.shared_assets_coordinator("purge-shared-assets", receipt)
            self.assertEqual(remote.call_count, 1)
            self.assertEqual(json.loads(receipt.read_text(encoding="utf-8"))["purge_result"], "ATTEMPTED")
            with mock.patch.object(CDN, "_run_remote_config") as repeated:
                with self.assertRaisesRegex(CDN.CDNError, "v2 PENDING"):
                    CDN.shared_assets_coordinator("purge-shared-assets", receipt)
            repeated.assert_not_called()

    def test_shared_assets_read_only_preflight_never_purges_or_logs_token(self):
        config = {"provider": "cloudflare", "action": "shared-assets-preflight",
                  "zone_name": "example.com", "origin": "https://assets.example.com", "secret_file": "/secret",
                  "socks": "127.0.0.1:1234"}
        encoded = base64.b64encode(json.dumps(config).encode()).decode()
        client = mock.Mock()
        client.resolve_zone.return_value = "zone-1"
        output = io.StringIO()
        with mock.patch.object(CDN, "read_secret", return_value=["DO-NOT-LOG"]), \
                mock.patch.object(CDN, "CloudflareClient", return_value=client), redirect_stdout(output):
            self.assertEqual(CDN.remote_main(encoded), 0)
        client.purge_files.assert_not_called()
        self.assertNotIn("DO-NOT-LOG", output.getvalue())

    def test_cloudflare_read_retry_is_bounded_and_secret_not_in_error(self):
        transport = QueueTransport([CDN.TransportError("timeout")] * CDN.READ_ATTEMPTS)
        client = CDN.CloudflareClient("DO-NOT-LOG", transport, sleep=lambda _: None)
        with self.assertRaises(CDN.TransportError) as raised:
            client.resolve_zone("example.com")
        self.assertEqual(len(transport.calls), CDN.READ_ATTEMPTS)
        self.assertNotIn("DO-NOT-LOG", str(raised.exception))

    def test_edgeone_read_only_preflight_never_creates_task_and_prints_no_secret(self):
        config = {"provider": "edgeone", "action": "preflight", "hostname": "www.example.com",
                  "zone_name": "example.com", "secret_file": "/secret"}
        encoded = base64.b64encode(json.dumps(config).encode()).decode()
        fake = self.identity_client(
            [self.zone()],
            [{"ZoneId": "zone-1", "DomainName": "www.example.com", "DomainStatus": "online"}],
        )
        output = io.StringIO()
        with mock.patch.object(CDN, "read_secret", return_value=["SECRET-ID", "SECRET-KEY"]), \
                mock.patch.object(CDN, "TencentClient", return_value=fake), redirect_stdout(output):
            self.assertEqual(CDN.remote_main(encoded), 0)
        self.assertNotIn("CreatePurgeTask", [x[0] for x in fake.seen])
        self.assertIn("EDGEONE AUTHORITY PREFLIGHT: PASS", output.getvalue())
        self.assertNotIn("SECRET", output.getvalue())

    def test_secret_file_contract_requires_root_owned_0600_regular(self):
        with tempfile.TemporaryDirectory() as directory:
            path = pathlib.Path(directory) / "edgeone.env"
            path.write_text("TENCENTCLOUD_SECRET_ID=id\nTENCENTCLOUD_SECRET_KEY=key\n", encoding="utf-8")
            path.chmod(0o600)
            real = os.lstat(path)
            valid = os.stat_result((stat.S_IFREG | 0o600, real.st_ino, real.st_dev, 1, 0, 0,
                                    real.st_size, real.st_atime, real.st_mtime, real.st_ctime))
            with mock.patch.object(CDN.os, "lstat", return_value=valid):
                self.assertEqual(CDN.read_secret(str(path), ("TENCENTCLOUD_SECRET_ID", "TENCENTCLOUD_SECRET_KEY"), exact=True), ["id", "key"])
            for mode, uid in ((stat.S_IFREG | 0o644, 0), (stat.S_IFREG | 0o600, 1000), (stat.S_IFLNK | 0o600, 0)):
                invalid = os.stat_result((mode, real.st_ino, real.st_dev, 1, uid, 0, real.st_size,
                                          real.st_atime, real.st_mtime, real.st_ctime))
                with self.subTest(mode=mode, uid=uid), mock.patch.object(CDN.os, "lstat", return_value=invalid):
                    with self.assertRaises(CDN.CDNError):
                        CDN.read_secret(str(path), ("TENCENTCLOUD_SECRET_ID", "TENCENTCLOUD_SECRET_KEY"), exact=True)


if __name__ == "__main__":
    unittest.main()
