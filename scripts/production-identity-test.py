#!/usr/bin/env python3

import copy
import importlib.util
import json
import pathlib
import sys
import tempfile
import unittest
from unittest import mock


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
        locales = [profile["locale"] for profile in parsed["locales"]]
        self.assertTrue(locales)
        self.assertEqual(len(locales), len(set(locales)))
        self.assertTrue(all(profile["production_state"] in ("first-production", "live") for profile in parsed["locales"]))

    def test_numeric_region_locale(self):
        data = copy.deepcopy(self.identity)
        data["locales"][-1]["locale"] = "zz-419"
        self.assertEqual(
            self.validate(data)["locales"][-1]["locale"], "zz-419"
        )
        data["locales"][-1]["locale"] = "zz-4a9"
        with self.assertRaisesRegex(MODULE.IdentityError, "not canonical"):
            self.validate(data)

    def test_dutch_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "nl-NL")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "nl-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["data_root"], "/data/go-tour-nl-NL")
        self.assertEqual(profile["systemd_service"], "go-tour-nl-NL.service")
        self.assertEqual(profile["loopback_port"], 4006)
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://nl-go-dev.shuijingwanwq.com/")

    def test_brazilian_portuguese_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "pt-BR")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "pt-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["data_root"], "/data/go-tour-pt-BR")
        self.assertEqual(profile["systemd_service"], "go-tour-pt-BR.service")
        self.assertEqual(profile["loopback_port"], 4007)
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://pt-go-dev.shuijingwanwq.com/")

    def test_turkish_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "tr-TR")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "tr-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["data_root"], "/data/go-tour-tr-TR")
        self.assertEqual(profile["releases_root"], "/data/go-tour-tr-TR/releases")
        self.assertEqual(profile["current"], "/data/go-tour-tr-TR/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-tr-TR/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-tr-TR.service")
        self.assertEqual(profile["loopback_port"], 4008)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4008/")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/tr-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/tr-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/tr-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://tr-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://tr-go-dev.shuijingwanwq.com/")

    def test_swedish_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "sv-SE")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "sv-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-sv-SE")
        self.assertEqual(profile["releases_root"], "/data/go-tour-sv-SE/releases")
        self.assertEqual(profile["current"], "/data/go-tour-sv-SE/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-sv-SE/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-sv-SE.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4009)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4009/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/sv-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/sv-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/sv-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://sv-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://sv-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_polish_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "pl-PL")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "pl-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-pl-PL")
        self.assertEqual(profile["releases_root"], "/data/go-tour-pl-PL/releases")
        self.assertEqual(profile["current"], "/data/go-tour-pl-PL/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-pl-PL/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-pl-PL.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4010)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4010/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/pl-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/pl-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/pl-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://pl-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://pl-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_traditional_chinese_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "zh-TW")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "zh-tw-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-zh-TW")
        self.assertEqual(profile["releases_root"], "/data/go-tour-zh-TW/releases")
        self.assertEqual(profile["current"], "/data/go-tour-zh-TW/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-zh-TW/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-zh-TW.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4011)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4011/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/zh-tw-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/zh-tw-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/zh-tw-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://zh-tw-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://zh-tw-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_indonesian_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "id-ID")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "id-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-id-ID")
        self.assertEqual(profile["releases_root"], "/data/go-tour-id-ID/releases")
        self.assertEqual(profile["current"], "/data/go-tour-id-ID/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-id-ID/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-id-ID.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4012)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4012/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/id-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/id-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/id-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://id-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://id-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_vietnamese_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "vi-VN")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "vi-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-vi-VN")
        self.assertEqual(profile["releases_root"], "/data/go-tour-vi-VN/releases")
        self.assertEqual(profile["current"], "/data/go-tour-vi-VN/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-vi-VN/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-vi-VN.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4013)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4013/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/vi-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/vi-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/vi-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://vi-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://vi-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_arabic_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "ar")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "ar-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["data_root"], "/data/go-tour-ar")
        self.assertEqual(profile["releases_root"], "/data/go-tour-ar/releases")
        self.assertEqual(profile["current"], "/data/go-tour-ar/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-ar/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-ar.service")
        self.assertEqual(profile["loopback_port"], 4014)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4014/")
        self.assertEqual(profile["playground_allowed_origin"], "https://ar-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://ar-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_thai_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "th-TH")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "th-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-th-TH")
        self.assertEqual(profile["releases_root"], "/data/go-tour-th-TH/releases")
        self.assertEqual(profile["current"], "/data/go-tour-th-TH/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-th-TH/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-th-TH.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4015)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4015/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/th-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/th-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/th-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://th-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://th-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_hindi_live_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "hi-IN")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "hi-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-hi-IN")
        self.assertEqual(profile["releases_root"], "/data/go-tour-hi-IN/releases")
        self.assertEqual(profile["current"], "/data/go-tour-hi-IN/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-hi-IN/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-hi-IN.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4016)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4016/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/hi-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/hi-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/hi-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://hi-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://hi-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_bengali_production_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "bn-BD")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "bn-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-bn-BD")
        self.assertEqual(profile["releases_root"], "/data/go-tour-bn-BD/releases")
        self.assertEqual(profile["current"], "/data/go-tour-bn-BD/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-bn-BD/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-bn-BD.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4017)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4017/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/bn-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/bn-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/bn-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://bn-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://bn-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_ukrainian_production_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "uk-UA")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "uk-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-uk-UA")
        self.assertEqual(profile["releases_root"], "/data/go-tour-uk-UA/releases")
        self.assertEqual(profile["current"], "/data/go-tour-uk-UA/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-uk-UA/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-uk-UA.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4019)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4019/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/uk-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/uk-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/uk-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://uk-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://uk-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_tamil_production_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "ta-IN")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "ta-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-ta-IN")
        self.assertEqual(profile["releases_root"], "/data/go-tour-ta-IN/releases")
        self.assertEqual(profile["current"], "/data/go-tour-ta-IN/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-ta-IN/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-ta-IN.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4022)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4022/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/ta-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/ta-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/ta-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://ta-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://ta-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")


    def test_telugu_production_profile_is_frozen(self):
        parsed = MODULE.load_identity(self.identity_path)
        profile = next(item for item in parsed["locales"] if item["locale"] == "te-IN")
        self.assertEqual(profile["production_state"], "live")
        self.assertEqual(profile["production_hostname"], "te-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["cdn"], "cloudflare")
        self.assertEqual(profile["origin_ssh_alias"], "aliyun")
        self.assertEqual(profile["origin_ip"], "121.40.248.29")
        self.assertEqual(profile["data_root"], "/data/go-tour-te-IN")
        self.assertEqual(profile["releases_root"], "/data/go-tour-te-IN/releases")
        self.assertEqual(profile["current"], "/data/go-tour-te-IN/current")
        self.assertEqual(profile["deployment_lock"], "/data/go-tour-te-IN/.deploy.lock")
        self.assertEqual(profile["systemd_service"], "go-tour-te-IN.service")
        self.assertEqual(profile["service_user"], "go-tour")
        self.assertEqual(profile["loopback_port"], 4023)
        self.assertEqual(profile["localhost_health_url"], "http://127.0.0.1:4023/")
        self.assertEqual(profile["environment_file"], "/etc/go-tour/go-tour.env")
        self.assertEqual(profile["nginx_vhost_path"], "/usr/local/nginx/conf/vhost/te-go-dev.shuijingwanwq.com.conf")
        self.assertEqual(profile["tls_certificate_path"], "/usr/local/nginx/conf/ssl/te-go-dev.shuijingwanwq.com.crt")
        self.assertEqual(profile["tls_key_path"], "/usr/local/nginx/conf/ssl/te-go-dev.shuijingwanwq.com.key")
        self.assertEqual(profile["playground_allowed_origin"], "https://te-go-dev.shuijingwanwq.com")
        self.assertEqual(profile["shared_assets_policy"], "shared-cloudflare")
        self.assertEqual(profile["production_public_url"], "https://te-go-dev.shuijingwanwq.com/")
        self.assertEqual(profile["cache_header"], "CF-Cache-Status")

    def test_list_cli_is_authority_derived_and_state_filtered(self):
        original = sys.argv
        try:
            sys.argv = ["production-identity.py", "--identity", str(self.identity_path), "list", "--state", "live"]
            with mock.patch("sys.stdout") as output:
                self.assertEqual(MODULE.main(), 0)
            lines = [line for call in output.write.call_args_list for line in [call.args[0].strip()] if line]
        finally:
            sys.argv = original
        expected = [profile for profile in self.identity["locales"] if profile["production_state"] == "live"]
        self.assertEqual(len(lines), len(expected))
        self.assertEqual([line.split("\t")[0] for line in lines], [profile["locale"] for profile in expected])

    def test_cdn_secret_authorities_are_frozen_without_credentials(self):
        shared = MODULE.load_identity(self.identity_path)["shared"]
        self.assertEqual(shared["cloudflare_zone_name"], "shuijingwanwq.com")
        self.assertEqual(shared["cloudflare_secret_file"], "/etc/go-tour/cloudflare.env")
        self.assertEqual(shared["edgeone_zone_name"], "shuijingwanwq.com")
        self.assertEqual(shared["edgeone_secret_file"], "/etc/go-tour/edgeone.env")
        rendered = json.dumps(shared)
        self.assertNotIn("TENCENTCLOUD_SECRET_ID", rendered)
        self.assertNotIn("TENCENTCLOUD_SECRET_KEY", rendered)

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
            sys.argv = ["production-identity.py", "--identity", str(self.identity_path), "locale", "zz-ZZ"]
            self.assertEqual(MODULE.main(), 1)
        finally:
            sys.argv = original


if __name__ == "__main__":
    unittest.main()
