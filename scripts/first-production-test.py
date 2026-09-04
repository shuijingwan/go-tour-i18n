#!/usr/bin/env python3

import ast
import importlib.util
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import unittest
from unittest import mock


ROOT = pathlib.Path(__file__).resolve().parent.parent


def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


FIRST = load("first_production", ROOT / "scripts" / "first-production.py")
BROWSER = load("production_browser", ROOT / "scripts" / "verify-production-browser.py")


PLAYGROUND = '''location = /compile {
    if ($http_origin !~ "^https://(go-dev|ja-go-dev)\\.shuijingwanwq\\.com$") {
      return 403;
    }
}
location = /fmt {
    if ($http_origin !~ "^https://(go-dev|ja-go-dev)\\.shuijingwanwq\\.com$") {
      return 403;
    }
}
'''


class FirstProductionTest(unittest.TestCase):
    def public_machine_request_helper(self):
        identity = FIRST.IDENTITY.load_identity(ROOT / "production" / "identity.json")
        instance = FIRST.Orchestrator.__new__(FIRST.Orchestrator)
        instance.profile = next(profile for profile in identity["locales"] if profile["locale"] == "es-ES")
        instance.shared = identity["shared"]
        instance.release_dir = pathlib.Path("/tmp/go-tour-release-es-ES-test")
        captured = []
        instance.ssh = lambda host, script, args=(), **kwargs: captured.append(script)
        instance.run = lambda *args, **kwargs: None
        instance.record = lambda stage: None
        instance.public_machine()
        script = captured[0]
        start = script.index("readonly CURL_CONNECT_TIMEOUT=5")
        end = script.index("for attempt in $(seq 1 30); do", start)
        return script[start:end]

    def run_public_machine_request(self, failures, expect_success):
        helper = self.public_machine_request_helper()
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            fake_bin = root / "bin"
            fake_bin.mkdir()
            counter = root / "curl-count"
            (fake_bin / "curl").write_text("""#!/usr/bin/env bash
set -Eeuo pipefail
count=0
[[ -f $FAKE_CURL_COUNT ]] && count=$(<"$FAKE_CURL_COUNT")
count=$((count + 1))
printf '%s' "$count" >"$FAKE_CURL_COUNT"
headers=''
body=''
while (($#)); do
  case $1 in
    -D|-o) value=$1; shift; [[ $# -gt 0 ]] || exit 2; [[ $value == -D ]] && headers=$1 || body=$1 ;;
  esac
  shift
done
if (( count <= FAKE_CURL_FAILURES )); then exit 7; fi
[[ -z $headers ]] || printf 'HTTP/2 200\\r\\nCF-Cache-Status: HIT\\r\\n' >"$headers"
[[ -z $body || $body == /dev/null ]] || printf '<html lang="es-ES"></html>' >"$body"
printf 200
""", encoding="utf-8")
            (fake_bin / "sleep").write_text("#!/usr/bin/env bash\nexit 0\n", encoding="utf-8")
            for executable in (fake_bin / "curl", fake_bin / "sleep"):
                executable.chmod(0o755)
            runner = root / "run-request.sh"
            outcome = "request 200 \"$temporary/body\" \"$temporary/headers\" https://example.invalid/" if expect_success else "if request 200 \"$temporary/body\" \"$temporary/headers\" https://example.invalid/; then exit 99; fi"
            runner.write_text("#!/usr/bin/env bash\nset -Eeuo pipefail\ntemporary=$(mktemp -d)\ntrap 'rm -rf \"$temporary\"' EXIT\n" + helper + outcome + "\n", encoding="utf-8")
            runner.chmod(0o755)
            env = dict(os.environ, PATH=str(fake_bin) + os.pathsep + os.environ["PATH"], FAKE_CURL_COUNT=str(counter), FAKE_CURL_FAILURES=str(failures))
            result = subprocess.run([str(runner)], capture_output=True, text=True, env=env)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(counter.read_text(encoding="utf-8"), str(failures + 1 if expect_success else 3))

    def test_public_machine_retries_transient_curl_failure(self):
        self.run_public_machine_request(failures=1, expect_success=True)

    def test_public_machine_fails_closed_after_retry_exhaustion(self):
        self.run_public_machine_request(failures=3, expect_success=False)

    def test_stage_order_puts_origin_before_dns(self):
        self.assertLess(FIRST.STAGE_ORDER.index("direct-origin"), FIRST.STAGE_ORDER.index("cloudflare-dns"))
        self.assertLess(FIRST.STAGE_ORDER.index("cloudflare-dns"), FIRST.STAGE_ORDER.index("public-machine"))

    def test_dns_absent_exact_and_conflict(self):
        self.assertEqual(FIRST.cloudflare_dns_state([], "fr.example", "192.0.2.1"), "absent")
        exact = [{"type": "A", "name": "fr.example", "content": "192.0.2.1", "proxied": True}]
        self.assertEqual(FIRST.cloudflare_dns_state(exact, "fr.example", "192.0.2.1"), "exact")
        for changed in ({"content": "192.0.2.2"}, {"proxied": False}, {"type": "CNAME"}):
            record = dict(exact[0]); record.update(changed)
            with self.assertRaises(ValueError):
                FIRST.cloudflare_dns_state([record], "fr.example", "192.0.2.1")
        with self.assertRaises(ValueError):
            FIRST.cloudflare_dns_state(exact * 2, "fr.example", "192.0.2.1")
        for record_type in ("AAAA", "CNAME"):
            with self.assertRaises(ValueError):
                FIRST.cloudflare_dns_state([dict(exact[0], type=record_type)], "fr.example", "192.0.2.1")

    def test_playground_add_is_idempotent_and_preserves_existing(self):
        updated = FIRST.update_playground_config(PLAYGROUND, "fr-go-dev.shuijingwanwq.com")
        self.assertIn("go-dev|ja-go-dev|fr-go-dev", updated)
        self.assertEqual(FIRST.update_playground_config(updated, "fr-go-dev.shuijingwanwq.com"), updated)
        self.assertEqual(updated.count("go-dev|ja-go-dev|fr-go-dev"), 2)

    def test_playground_malformed_or_different_locations_fail(self):
        with self.assertRaises(ValueError):
            FIRST.update_playground_config(PLAYGROUND.replace("location = /fmt", "location /fmt"), "fr-go-dev.shuijingwanwq.com")
        malformed = PLAYGROUND.replace("go-dev|ja-go-dev", "go-dev|de-go-dev", 1)
        with self.assertRaises(ValueError):
            FIRST.update_playground_config(malformed, "fr-go-dev.shuijingwanwq.com")

    def test_browser_accepts_filled_and_unfilled_ads(self):
        self.assertTrue(BROWSER.browser_ad_gate({"mount": 1, "loader": True, "ad": 1, "filled": True}))
        self.assertTrue(BROWSER.browser_ad_gate({"mount": 1, "loader": True, "ad": 1, "filled": False}))
        self.assertFalse(BROWSER.browser_ad_gate({"mount": 1, "loader": True, "ad": 0, "filled": False}))

    def test_browser_entrypoint_binds_urls_to_formal_identity(self):
        captured = []
        original = sys.argv
        try:
            sys.argv = ["verify-production-browser.py", "https://fr-go-dev.shuijingwanwq.com/", "fr-FR"]
            with mock.patch.object(BROWSER, "acceptance", side_effect=lambda *args: captured.append(args)):
                self.assertEqual(BROWSER.main(), 0)
        finally:
            sys.argv = original
        self.assertEqual(captured[0][2]["production_state"], "live")
        self.assertEqual(captured[0][3]["playground_public_origin"], "https://play.go-dev.shuijingwanwq.com:8443")

    def test_browser_acceptance_checks_observable_behavior(self):
        source = (ROOT / "scripts" / "browser_acceptance.py").read_text(encoding="utf-8")
        for evidence in (
            "formatted !=", "Reset did not restore Angular model", "Reset did not restore CodeMirror view",
            "Run produced no browser-visible expected result",
            "Playground POST Origin mismatch", "SPA next-page transition did not change route",
            "unexpected page-level horizontal overflow", 'identity["canonical"] == expected_canonical',
            'identity["origin"] == base.rstrip("/")',
        ):
            self.assertIn(evidence, source)

    def test_secret_values_are_not_receipt_fields(self):
        allowed = {"schema", "run_id", "locale", "hostname", "release", "started_at", "completed_at", "result", "stages"}
        receipt = {key: None for key in allowed}
        serialized = __import__("json").dumps(receipt)
        self.assertNotIn("CF_Token", serialized)
        self.assertNotIn("TOUR_AD_HTML", serialized)

    def test_origin_failure_never_reaches_dns_or_receipt_pass(self):
        calls = []
        class Fake:
            receipt = {"stages": {}}
            def write_receipt(self, result=None): calls.append(("receipt", result))
            def stage_passed(self, stage): return False
            def preflight(self): calls.append("preflight")
            def bootstrap_infrastructure(self): calls.append("infrastructure")
            def configure_playground(self): calls.append("playground")
            def deploy(self): calls.append("deploy")
            def direct_origin(self): calls.append("direct-origin"); raise FIRST.FirstProductionError("direct-origin", "PASS", "failed", "inspect")
            def cloudflare_dns(self): calls.append("dns")
            def public_machine(self): calls.append("public")
            def browser(self): calls.append("browser")
        with self.assertRaises(FIRST.FirstProductionError):
            FIRST.Orchestrator.execute(Fake())
        self.assertNotIn("dns", calls)
        self.assertNotIn(("receipt", "passed"), calls)

    def test_preflight_failure_reaches_no_production_mutation(self):
        calls = []
        class Fake:
            receipt = {"stages": {}}
            def write_receipt(self, result=None): calls.append(("receipt", result))
            def stage_passed(self, stage): return False
            def preflight(self):
                calls.append("preflight")
                raise FIRST.FirstProductionError("preflight", "Cloudflare secret", "missing", "provision")
            def bootstrap_infrastructure(self): calls.append("infrastructure")
            def configure_playground(self): calls.append("playground")
            def deploy(self): calls.append("deploy")
            def direct_origin(self): calls.append("direct-origin")
            def cloudflare_dns(self): calls.append("dns")
            def public_machine(self): calls.append("public")
            def browser(self): calls.append("browser")
        with self.assertRaises(FIRST.FirstProductionError):
            FIRST.Orchestrator.execute(Fake())
        self.assertEqual(calls, [("receipt", None), "preflight"])

    def test_controlmaster_cleanup_covers_both_hosts(self):
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            paths = {"aliyun": root / "a.sock", "zgocloud": root / "z.sock"}
            instance = FIRST.Orchestrator.__new__(FIRST.Orchestrator)
            instance.control = paths
            instance.temp = root
            instance.ssh_options = lambda host: ["-o", f"ControlPath={paths[host]}"]
            with mock.patch.object(pathlib.Path, "is_socket", return_value=True), mock.patch.object(FIRST.subprocess, "run") as run:
                instance.cleanup()
            self.assertEqual(run.call_count, 2)

    def test_secret_and_nginx_fail_closed_guards_are_present(self):
        source = (ROOT / "scripts" / "first-production.py").read_text(encoding="utf-8")
        self.assertIn("root:root 600", source)
        self.assertNotIn("set -x", source)
        self.assertIn("nginx -t failed; invocation-created vhost was removed", source)
        self.assertIn("Nginx reload failed; invocation-created vhost was removed", source)
        self.assertIn("cp -a \"$backup\" \"$vhost\"", source)
        self.assertIn("existing systemd unit content is not the exact formal baseline", source)
        self.assertIn("existing vhost content is not the exact formal baseline", source)
        self.assertIn("resumed service is not active", source)
        self.assertIn("resumed service is not listening on its identity port", source)
        self.assertIn("resumed source health is not HTTP 200", source)
        self.assertIn('systemctl enable "$service"', source)

    def test_aliyun_uses_formal_oneinstack_nginx_without_path_lookup(self):
        identity = FIRST.IDENTITY.load_identity(ROOT / "production" / "identity.json")
        instance = FIRST.Orchestrator.__new__(FIRST.Orchestrator)
        instance.profile = next(profile for profile in identity["locales"] if profile["locale"] == "ko-KR")
        instance.shared = identity["shared"]
        instance.release_name = "20260902-ko-KR-test"
        instance.stage_passed = lambda stage: False
        instance.record = lambda stage: None
        calls = []

        def fake_ssh(host, script, args=(), **kwargs):
            calls.append((host, script, args, kwargs))
            return "zone_id=test"

        instance.ssh = fake_ssh
        self.assertEqual(instance.aliyun_preflight(), "zone_id=test")
        host, preflight, args, _ = calls.pop()
        self.assertEqual(host, identity["shared"]["aliyun_ssh_alias"])
        self.assertEqual(args[-1], FIRST.ALIYUN_ONEINSTACK_NGINX)
        self.assertIn("nginx=${21}", preflight)
        self.assertIn('[[ -x $nginx ]]', preflight)
        self.assertNotIn(" mv nginx openssl ", preflight)

        instance.bootstrap_infrastructure()
        host, infrastructure, args, _ = calls.pop()
        self.assertEqual(host, identity["shared"]["aliyun_ssh_alias"])
        self.assertEqual(args[-1], FIRST.ALIYUN_ONEINSTACK_NGINX)
        self.assertIn('if ! "$nginx" -t; then', infrastructure)
        self.assertIn('"$nginx" -t && service nginx reload', infrastructure)
        self.assertNotIn("if ! nginx -t", infrastructure)

    def test_zgocloud_uses_formal_oneinstack_nginx_for_preflight_mutation_and_recovery(self):
        identity = FIRST.IDENTITY.load_identity(ROOT / "production" / "identity.json")
        instance = FIRST.Orchestrator.__new__(FIRST.Orchestrator)
        instance.profile = next(profile for profile in identity["locales"] if profile["locale"] == "ko-KR")
        instance.shared = identity["shared"]
        instance.run_id = "test"
        instance.record = lambda stage: None
        calls = []
        instance.ssh = lambda host, script, args=(), **kwargs: calls.append((host, script, args, kwargs)) or "origins=go-dev"

        self.assertEqual(instance.zgocloud_preflight(), "origins=go-dev")
        host, preflight, args, _ = calls.pop()
        self.assertEqual(host, identity["shared"]["zgocloud_ssh_alias"])
        self.assertEqual(args[-1], FIRST.ZGOCLOUD_ONEINSTACK_NGINX)
        self.assertIn('[[ -x $nginx ]] || fail "missing formal ZgoCloud Nginx executable: $nginx"', preflight)
        self.assertIn('"$nginx" -t >/dev/null', preflight)
        self.assertNotIn("nginx -t", preflight.replace('"$nginx" -t', ""))

        instance.configure_playground()
        host, mutation, args, _ = calls.pop()
        self.assertEqual(host, identity["shared"]["zgocloud_ssh_alias"])
        self.assertEqual(args[-1], FIRST.ZGOCLOUD_ONEINSTACK_NGINX)
        self.assertIn('[[ -x $nginx ]] ||', mutation)  # Missing absolute executable fails before mutation.
        self.assertIn('if ! "$nginx" -t; then', mutation)
        self.assertIn('cp -a "$backup" "$vhost"; "$nginx" -t || true', mutation)
        self.assertIn('cp -a "$backup" "$vhost"; "$nginx" -t && service nginx reload || true', mutation)
        self.assertNotIn("nginx -t", mutation.replace('"$nginx" -t', ""))

    def test_public_machine_cache_eligibility_uses_real_status_without_rule_expression_parser(self):
        source = (ROOT / "scripts" / "first-production.py").read_text(encoding="utf-8")
        start = source.index("    def public_machine(self):")
        end = source.index("\n    def browser(self):", start)
        public_machine = source[start:end]
        self.assertIn("MISS|HIT|EXPIRED|REVALIDATED|UPDATING|STALE", public_machine)
        self.assertIn("readonly CURL_RETRY_ATTEMPTS=3", public_machine)
        self.assertIn("6|7|16|28|35", public_machine)
        self.assertIn("request 200 /dev/null \"$temporary/cache\"", public_machine)
        self.assertNotIn("DYNAMIC", public_machine)
        self.assertNotIn("BYPASS", public_machine)
        self.assertNotIn("curl -4", public_machine)
        self.assertNotIn("rulesets/phases/http_request_cache_settings", source)
        self.assertNotIn("cache_rule=", source)

    def test_aliyun_vhost_scanner_is_python36_compatible_and_separates_failures(self):
        identity = FIRST.IDENTITY.load_identity(ROOT / "production" / "identity.json")
        instance = FIRST.Orchestrator.__new__(FIRST.Orchestrator)
        instance.profile = next(profile for profile in identity["locales"] if profile["locale"] == "ko-KR")
        instance.shared = identity["shared"]
        instance.release_name = "20260902-ko-KR-test"
        instance.stage_passed = lambda stage: False
        calls = []
        instance.ssh = lambda host, script, args=(), **kwargs: calls.append(script) or "zone_id=test"
        instance.aliyun_preflight()
        remote = calls[0]
        marker = 'python3 - "$vhost" "$hostname" <<\'PY\' 2>&1\n'
        self.assertIn(marker, remote)
        scanner = remote.split(marker, 1)[1].split('\nPY\n)', 1)[0]
        ast.parse(scanner, feature_version=(3, 6))
        self.assertNotIn("removeprefix", scanner)
        self.assertNotIn("removesuffix", scanner)
        self.assertIn("vhost_scan_status", remote)
        self.assertIn("Nginx vhost hostname checker failed", remote)

        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            target = root / "ko.conf"
            target.write_text("server_name ko-go-dev.shuijingwanwq.com;\n", encoding="utf-8")
            clean = subprocess.run([sys.executable, "-c", scanner, str(target), "ko-go-dev.shuijingwanwq.com"], capture_output=True, text=True)
            self.assertEqual(clean.returncode, 0)
            (root / "conflict.conf").write_text("server_name ko-go-dev.shuijingwanwq.com;\n", encoding="utf-8")
            conflict = subprocess.run([sys.executable, "-c", scanner, str(target), "ko-go-dev.shuijingwanwq.com"], capture_output=True, text=True)
            self.assertEqual(conflict.returncode, 10)
            self.assertIn("production hostname is declared by:", conflict.stderr)
        broken = subprocess.run([sys.executable, "-c", scanner, "/missing/vhost.conf", "ko-go-dev.shuijingwanwq.com"], capture_output=True, text=True)
        self.assertEqual(broken.returncode, 2)
        self.assertIn("vhost hostname checker exception:", broken.stderr)

    def test_shared_assets_freshness_reuses_public_core_after_current_export(self):
        source = (ROOT / "scripts" / "first-production.py").read_text(encoding="utf-8")
        start = source.index("    def shared_assets_freshness(self):")
        end = source.index("\n    def preflight(self):", start)
        freshness = source[start:end]
        self.assertIn('"assets", "export", "--output", export', freshness)
        self.assertIn('"assets", "validate", "--input", export', freshness)
        self.assertIn('ROOT / "scripts" / "verify-shared-assets-public.sh", export', freshness)
        self.assertIn('shared-assets origin matches current export', freshness)
        self.assertNotIn("public_script", freshness)
        self.assertNotIn("curl -f", freshness)
        self.assertNotIn('self.ssh(self.shared["zgocloud_ssh_alias"]', freshness)

    def test_templates_are_derived_from_profile(self):
        profile = dict(FIRST.IDENTITY.load_identity(ROOT / "production" / "identity.json")["locales"][1])
        profile.update({
            "locale": "es-ES", "systemd_service": "go-tour-es-ES.service",
            "current": "/data/go-tour-es-ES/current", "loopback_port": 4998,
            "production_hostname": "es-go-dev.shuijingwanwq.com",
            "tls_certificate_path": "/tmp/es.crt", "tls_key_path": "/tmp/es.key",
        })
        rendered = FIRST.systemd_unit_text(profile) + FIRST.nginx_vhost_text(profile)
        for expected in ("es-ES", "/data/go-tour-es-ES/current", "127.0.0.1:4998", "es-go-dev.shuijingwanwq.com", "/tmp/es.crt", "/tmp/es.key"):
            self.assertIn(expected, rendered)
        for forbidden in ("fr-FR", "fr-go-dev", "4002"):
            self.assertNotIn(forbidden, rendered)

    def test_completed_receipt_cannot_repeat_bootstrap(self):
        with tempfile.TemporaryDirectory() as directory:
            release = pathlib.Path(directory) / "go-tour-release-fr-FR-audit"
            release.mkdir()
            receipt = release.parent / f"{release.name}.first-production-receipt.json"
            receipt.write_text(json.dumps({
                "schema": FIRST.RECEIPT_SCHEMA, "locale": "fr-FR",
                "hostname": "fr-go-dev.shuijingwanwq.com", "release": "fr-FR-audit",
                "result": "passed", "stages": {},
            }), encoding="utf-8")
            instance = FIRST.Orchestrator.__new__(FIRST.Orchestrator)
            instance.run_id = "audit"
            instance.locale = "fr-FR"
            instance.profile = {"production_hostname": "fr-go-dev.shuijingwanwq.com"}
            instance.release_name = "fr-FR-audit"
            instance.receipt_path = receipt
            with self.assertRaisesRegex(FIRST.FirstProductionError, "first-production already passed"):
                instance._load_or_new_receipt()

    def test_live_locale_cannot_enter_first_production(self):
        with tempfile.TemporaryDirectory() as directory:
            release = pathlib.Path(directory) / "go-tour-release-fr-FR-audit"
            release.mkdir()
            (release / "release.json").write_text(json.dumps({"locale": "fr-FR"}), encoding="utf-8")
            with self.assertRaisesRegex(FIRST.FirstProductionError, "production_state=live"):
                FIRST.Orchestrator(release)


if __name__ == "__main__":
    unittest.main()
