#!/usr/bin/env python3

import base64
import ast
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

    def test_edgeone_purge_exact_payload_and_processing_success(self):
        client = FakeTencent([
            ("CreatePurgeTask", {"JobId": "job-1", "FailedList": [], "RequestId": "create"}),
            ("DescribePurgeTasks", {"Tasks": [self.task("processing")]}),
            ("DescribePurgeTasks", {"Tasks": [self.task("success")], "RequestId": "done"}),
        ])
        self.assertEqual(CDN.purge_edgeone(client, "zone-1", "www.example.com"), "done")
        action, payload, mutation = client.seen[0]
        self.assertEqual((action, mutation), ("CreatePurgeTask", True))
        self.assertEqual(payload, {"ZoneId": "zone-1", "Type": "purge_host", "Method": "delete", "Targets": ["www.example.com"]})
        rendered = json.dumps(payload)
        self.assertNotIn("purge_all", rendered)
        self.assertNotIn("*", rendered)
        self.assertNotIn("invalidate", rendered)

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
