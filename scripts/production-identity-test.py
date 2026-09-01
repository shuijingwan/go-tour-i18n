#!/usr/bin/env python3

import copy
import importlib.util
import json
import pathlib
import sys
import tempfile
import unittest


ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location(
    "production_identity", ROOT / "scripts" / "production-identity.py"
)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ProductionIdentityTest(unittest.TestCase):
    def setUp(self):
        self.identity_path = ROOT / "production" / "identity.json"
        self.identity = json.loads(self.identity_path.read_text(encoding="utf-8"))

    def validate(self, data):
        with tempfile.TemporaryDirectory() as directory:
            path = pathlib.Path(directory) / "identity.json"
            path.write_text(json.dumps(data), encoding="utf-8")
            return MODULE.load_identity(path)

    def test_repository_identity(self):
        parsed = MODULE.load_identity(self.identity_path)
        self.assertEqual([p["locale"] for p in parsed["locales"]], ["zh-CN", "ja-JP", "de-DE", "fr-FR"])

    def test_unknown_or_missing_field_fails_closed(self):
        data = copy.deepcopy(self.identity)
        del data["locales"][0]["systemd_service"]
        with self.assertRaisesRegex(MODULE.IdentityError, "missing=.*systemd_service"):
            self.validate(data)

    def test_duplicate_identity_fails_closed(self):
        for field in ("locale", "production_hostname", "loopback_port", "systemd_service", "data_root"):
            with self.subTest(field=field):
                data = copy.deepcopy(self.identity)
                data["locales"][1][field] = data["locales"][0][field]
                if field == "data_root":
                    root = data["locales"][1][field]
                    data["locales"][1]["releases_root"] = root + "/releases"
                    data["locales"][1]["current"] = root + "/current"
                    data["locales"][1]["deployment_lock"] = root + "/.deploy.lock"
                with self.assertRaises(MODULE.IdentityError):
                    self.validate(data)

    def test_url_port_and_path_must_agree(self):
        data = copy.deepcopy(self.identity)
        data["locales"][2]["localhost_health_url"] = "http://127.0.0.1:4999/"
        with self.assertRaisesRegex(MODULE.IdentityError, "does not match loopback_port"):
            self.validate(data)
        data = copy.deepcopy(self.identity)
        data["locales"][2]["current"] = "/data/go-tour-de-DE/other"
        with self.assertRaisesRegex(MODULE.IdentityError, "boundaries"):
            self.validate(data)

    def test_production_state_fails_closed(self):
        data = copy.deepcopy(self.identity)
        data["locales"][0]["production_state"] = "unknown"
        with self.assertRaisesRegex(MODULE.IdentityError, "production_state"):
            self.validate(data)

    def test_unknown_locale_cli_fails_closed(self):
        original = sys.argv
        try:
            sys.argv = ["production-identity.py", "--identity", str(self.identity_path), "locale", "es-ES"]
            self.assertEqual(MODULE.main(), 1)
        finally:
            sys.argv = original


if __name__ == "__main__":
    unittest.main()
