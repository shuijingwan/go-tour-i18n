package tour

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"html"
	"image/png"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

func TestSupportQRAssetsPreserveOfficialPNGs(t *testing.T) {
	handler := productionTestHandlerLocale(t, "http://127.0.0.1:1", "zh-CN")
	for _, test := range []struct {
		path          string
		width, height int
		sha256        string
	}{
		{"images/support/wechat.png", 1118, 1524, "b635e10b71e2ad879965de58ba6ea3110eee20539138320cbb14228870e9b8d2"},
		{"images/support/alipay.png", 1080, 1620, "8e187bf29fa0dcd9857ba5b88e07fa6caa2c817b8ef52ce97a8c084e023dddfd"},
	} {
		data, err := fs.ReadFile(contentTour, test.path)
		if err != nil {
			t.Fatal(err)
		}
		config, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("decode %s: %v", test.path, err)
		}
		if config.Width != test.width || config.Height != test.height {
			t.Errorf("%s dimensions = %dx%d, want %dx%d", test.path, config.Width, config.Height, test.width, test.height)
		}
		digest := sha256.Sum256(data)
		if got := hex.EncodeToString(digest[:]); got != test.sha256 {
			t.Errorf("%s SHA-256 = %s, want %s", test.path, got, test.sha256)
		}

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/"+test.path, nil))
		if recorder.Code != http.StatusOK || !bytes.Equal(recorder.Body.Bytes(), data) {
			t.Errorf("GET /%s did not serve the embedded PNG byte-for-byte", test.path)
		}
		if got := recorder.Header().Get("Content-Type"); got != "image/png" {
			t.Errorf("GET /%s Content-Type = %q, want image/png", test.path, got)
		}
	}
}

func TestProjectSupportConfiguration(t *testing.T) {
	if err := validateProjectSupport(); err != nil {
		t.Fatal(err)
	}
	want := map[string]supportPaymentMethod{
		"wechat-pay":  {Identity: "wechat-pay", Name: "微信支付", Kind: "qr", Enabled: true, Audience: supportAudienceMainland, QRAsset: "images/support/wechat.png"},
		"alipay":      {Identity: "alipay", Name: "支付宝", Kind: "qr", Enabled: true, Audience: supportAudienceMainland, QRAsset: "images/support/alipay.png"},
		"unionpay":    {Identity: "unionpay", Name: "云闪付", Kind: "qr", Enabled: false, Audience: supportAudienceMainland},
		"usdc-base":   {Identity: "usdc-base", Name: "USDC", Kind: "crypto", Enabled: true, Audience: supportAudienceInternational, Asset: "USDC", Network: "Base", Address: "0x225f14d54683b1f5bc153bc8a678cad0277096d3", MinimumDeposit: "0.01 USDC", NetworkWarningKey: "support.usdc_network_only"},
		"usdt-trc20":  {Identity: "usdt-trc20", Name: "USDT", Kind: "crypto", Enabled: true, Audience: supportAudienceInternational, Asset: "USDT", Network: "Tron (TRC20)", Address: "TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK", MinimumDeposit: "0.1 USDT", NetworkWarningKey: "support.usdt_network_only"},
		"binance-uid": {Identity: "binance-uid", Name: "Binance", Kind: "platform", Enabled: true, Audience: supportAudienceInternational, UID: "1055351242"},
		"okx-uid":     {Identity: "okx-uid", Name: "OKX", Kind: "platform", Enabled: true, Audience: supportAudienceInternational, UID: "231321605530361856"},
		"wise":        {Identity: "wise", Name: "Wise", Kind: "link", Enabled: false, Audience: supportAudienceInternational},
		"patreon":     {Identity: "patreon", Name: "Patreon", Kind: "link", Enabled: false, Audience: supportAudienceInternational},
	}
	if len(ProjectSupport.Methods) != len(want) {
		t.Fatalf("payment method count = %d, want %d", len(ProjectSupport.Methods), len(want))
	}
	for _, method := range ProjectSupport.Methods {
		if expected, ok := want[method.Identity]; !ok || method != expected {
			t.Errorf("payment method %+v, want %+v", method, expected)
		}
	}
}

func TestProjectSupportValidationRejectsInvalidEnabledMethods(t *testing.T) {
	original := ProjectSupport
	original.Methods = append([]supportPaymentMethod(nil), ProjectSupport.Methods...)
	t.Cleanup(func() { ProjectSupport = original })

	for _, test := range []struct {
		name   string
		mutate func([]supportPaymentMethod)
		want   string
	}{
		{"missing crypto address", func(methods []supportPaymentMethod) { methods[3].Address = "" }, "incomplete"},
		{"QR method with non-QR field", func(methods []supportPaymentMethod) { methods[0].Network = "wrong" }, "mixes non-QR fields"},
		{"missing platform UID", func(methods []supportPaymentMethod) { methods[5].UID = "" }, "platform payment method"},
		{"platform method with address", func(methods []supportPaymentMethod) { methods[5].Address = "wrong" }, "platform payment method"},
		{"duplicate identity", func(methods []supportPaymentMethod) { methods[1].Identity = methods[0].Identity }, "duplicate"},
		{"invalid audience", func(methods []supportPaymentMethod) { methods[0].Audience = "geoip" }, "invalid audience"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ProjectSupport = original
			ProjectSupport.Methods = append([]supportPaymentMethod(nil), original.Methods...)
			test.mutate(ProjectSupport.Methods)
			if err := validateProjectSupport(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateProjectSupport() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestSupportAudienceAndLocaleReference(t *testing.T) {
	for _, language := range languageRegistry {
		view, err := supportForLocale(language.Locale)
		if err != nil {
			t.Fatal(err)
		}
		wantAudience := supportAudienceInternational
		wantMethods := []string{"usdc-base", "usdt-trc20", "binance-uid", "okx-uid"}
		if language.Locale == "zh-CN" {
			wantAudience = supportAudienceMainland
			wantMethods = []string{"wechat-pay", "alipay"}
		}
		if view.Audience != wantAudience {
			t.Errorf("%s audience = %q, want %q", language.Locale, view.Audience, wantAudience)
		}
		if view.Reference != "go-dev-"+language.Locale {
			t.Errorf("%s reference = %q", language.Locale, view.Reference)
		}
		if len(view.Methods) != len(wantMethods) {
			t.Fatalf("%s method count = %d, want %d", language.Locale, len(view.Methods), len(wantMethods))
		}
		for i, method := range view.Methods {
			if method.Identity != wantMethods[i] || !method.Enabled || method.Audience != wantAudience {
				t.Errorf("%s method[%d] = %+v", language.Locale, i, method)
			}
		}
	}
}

func TestHomepageSupportMethodsAreAudienceSpecific(t *testing.T) {
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		locale string
		want   []string
		absent []string
	}{
		{
			locale: "zh-CN",
			want: []string{
				"支持与运营成本", "广告说明", "基础设施费用和项目持续维护成本", "支持简体中文版本", "go-dev-zh-CN", `data-copy-value="go-dev-zh-CN"`,
				`aria-label="复制支持备注" title="复制支持备注"`, `class="support-copy-icon"`,
				`data-payment-method="wechat-pay" aria-label="微信支付"`, `src="/images/support/wechat.png" alt="微信支付"`,
				`data-payment-method="alipay" aria-label="支付宝"`, `src="/images/support/alipay.png" alt="支付宝"`,
			},
			absent: []string{"支持此翻译", "<h3>微信支付</h3>", "<h3>支付宝</h3>", "unionpay", "云闪付", "usdc-base", "usdt-trc20", "binance-uid", "okx-uid", "Binance", "OKX", "Wise", "Patreon", "0x225f14d54683b1f5bc153bc8a678cad0277096d3", "TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK"},
		},
		{
			locale: "ja-JP",
			want: []string{
				"サポートと運営費", "広告について", "インフラ費用とプロジェクトの継続的なメンテナンス費用", "日本語版を支援する", "go-dev-ja-JP", `data-copy-value="go-dev-ja-JP"`,
				`data-payment-method="usdc-base"`, "USDC", "Base", "0.01 USDC", "0x225f14d54683b1f5bc153bc8a678cad0277096d3",
				`data-copy-value="0x225f14d54683b1f5bc153bc8a678cad0277096d3"`, `data-payment-method="usdt-trc20"`, "Tron (TRC20)", "0.1 USDT", "TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK",
				`data-copy-value="TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK"`, "受取側の最低金額", "プラットフォーム内送金（任意）",
				`data-payment-method="binance-uid"`, "1055351242", `data-payment-method="okx-uid"`, "231321605530361856",
				`aria-label="UID をコピー" title="UID をコピー"`, `src="/tour/static/js/support.js"`,
			},
			absent: []string{"wechat-pay", "alipay", "unionpay", "微信支付", "支付宝", "云闪付", "Wise", "Patreon", "images/support/wechat.png", "images/support/alipay.png"},
		},
	} {
		t.Run(test.locale, func(t *testing.T) {
			catalog, err := ui.Load(test.locale)
			if err != nil {
				t.Fatal(err)
			}
			localized := metadata
			localized.Locale = test.locale
			home, err := renderHome(catalog, localized)
			if err != nil {
				t.Fatal(err)
			}
			page := string(home)
			for _, want := range test.want {
				if !strings.Contains(page, want) {
					t.Errorf("homepage does not contain %q", want)
				}
			}
			for _, absent := range test.absent {
				if strings.Contains(page, absent) {
					t.Errorf("homepage unexpectedly contains %q", absent)
				}
			}
			howTitle, _ := catalog.Plain("site.how_it_works")
			continueTitle, _ := catalog.Plain("site.continue_learning_title")
			how := strings.Index(page, "<h2>"+howTitle+"</h2>")
			support := strings.Index(page, `class="site-section site-support"`)
			continued := strings.Index(page, "<h2>"+continueTitle+"</h2>")
			if how < 0 || support <= how || continued <= support {
				t.Errorf("support section order is invalid: how=%d support=%d continue=%d", how, support, continued)
			}
		})
	}

	production := metadata
	production.Development = false
	production.Locale = "ja-JP"
	production.PublishedAt = "2026-08-12T07:23:34Z"
	catalog, err := ui.Load("ja-JP")
	if err != nil {
		t.Fatal(err)
	}
	home, err := renderHome(catalog, production)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(home), `src="`+assets.BaseURL+`/tour/static/js/support.js"`) {
		t.Fatal("international production homepage does not use the shared support script")
	}
}

func TestHomepageSupportCopyStateIsLocalizedForEveryLocale(t *testing.T) {
	metadata, err := loadSiteMetadata(contentTour)
	if err != nil {
		t.Fatal(err)
	}
	for _, language := range languageRegistry {
		catalog, err := ui.Load(language.Locale)
		if err != nil {
			t.Fatal(err)
		}
		localized := metadata
		localized.Locale = language.Locale
		home, err := renderHome(catalog, localized)
		if err != nil {
			t.Fatal(err)
		}
		copied, err := catalog.Plain("support.copied")
		if err != nil {
			t.Fatal(err)
		}
		copyReference, _ := catalog.Plain("support.copy_reference")
		copyAddress, _ := catalog.Plain("support.copy_address")
		copyUID, _ := catalog.Plain("support.copy_uid")
		page := string(home)
		wantButtons := 5
		if language.Locale == "zh-CN" {
			wantButtons = 1
		}
		if got := strings.Count(page, `data-copy-success="`+copied+`"`); got != wantButtons {
			t.Errorf("%s localized copied state count = %d, want %d", language.Locale, got, wantButtons)
		}
		if !strings.Contains(page, `data-copy-value="go-dev-`+language.Locale+`"`) {
			t.Errorf("%s homepage is missing its complete locale reference copy value", language.Locale)
		}
		if language.Locale != "en" && copied == "Copied" {
			t.Errorf("%s retained the English copied state", language.Locale)
		}
		if got := strings.Count(page, `class="support-copy-icon"`); got != wantButtons {
			t.Errorf("%s copy icon count = %d, want %d", language.Locale, got, wantButtons)
		}
		if got := strings.Count(page, `data-copy-toast`); got != 1 {
			t.Errorf("%s copy toast count = %d, want 1", language.Locale, got)
		}
		copyReference = html.EscapeString(copyReference)
		copyAddress = html.EscapeString(copyAddress)
		copyUID = html.EscapeString(copyUID)
		if got := strings.Count(page, `aria-label="`+copyReference+`" title="`+copyReference+`"`); got != 1 {
			t.Errorf("%s reference copy accessible label count = %d, want 1", language.Locale, got)
		}
		wantAddress, wantUID := 2, 2
		if language.Locale == "zh-CN" {
			wantAddress, wantUID = 0, 0
		}
		if got := strings.Count(page, `aria-label="`+copyAddress+`" title="`+copyAddress+`"`); got != wantAddress {
			t.Errorf("%s address copy accessible label count = %d, want %d", language.Locale, got, wantAddress)
		}
		if got := strings.Count(page, `aria-label="`+copyUID+`" title="`+copyUID+`"`); got != wantUID {
			t.Errorf("%s UID copy accessible label count = %d, want %d", language.Locale, got, wantUID)
		}
	}
}

func TestSupportScriptUsesClipboardAPIWithFallback(t *testing.T) {
	file, err := contentTour.Open("tour/static/js/support.js")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	for _, want := range []string{
		"navigator.clipboard.writeText(value)", "document.execCommand('copy')", "data-copy-value", "data-copy-success",
		"data-copy-toast", "toast.hidden = false", "window.clearTimeout(toastTimer)", "window.setTimeout(hideToast, 1600)",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("support copy script does not contain %q", want)
		}
	}
	if strings.Contains(script, "visualViewport") {
		t.Fatal("support copy script still contains visualViewport compensation")
	}
	if strings.Contains(script, "button.textContent") {
		t.Fatal("support copy script still replaces the icon button content")
	}
}

func TestSupportCopyStylesKeepIconAndLongValuesAccessible(t *testing.T) {
	file, err := contentTour.Open("tour/static/css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	for _, want := range []string{
		"overflow-wrap: anywhere", ".support-copy:focus-visible", "width: 44px", "height: 44px",
		"width: 22px", "height: 22px", "position: fixed", "left: 50%",
		"bottom: calc(24px + env(safe-area-inset-bottom))", ".support-copy-toast[hidden]",
		"top: 50%", "bottom: auto", "transform: translate(-50%, -50%)", "max-width: 1008px",
		".support-payment-grid,\n    .support-platform-grid {\n        gap: 12px;",
		"[data-theme='dark'] .support-copy", "[data-theme='dark'] .support-copy-toast",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("support CSS does not contain %q", want)
		}
	}
	for _, stale := range []string{"--support-visual-viewport-", "margin-right: -24px", "margin-left: -24px"} {
		if strings.Contains(css, stale) {
			t.Errorf("support CSS still contains stale layout rule %q", stale)
		}
	}
}

func TestREADMEContainsCompleteProjectSupportMethods(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	readme := string(data)
	for _, want := range []string{
		"微信支付", "支付宝", "_content/images/support/wechat.png", "_content/images/support/alipay.png",
		"USDC", "Base", "0x225f14d54683b1f5bc153bc8a678cad0277096d3", "0.01 USDC",
		"USDT", "Tron (TRC20)", "TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK", "0.1 USDT",
		"Binance", "1055351242", "OKX", "231321605530361856", "go-dev-project",
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("README support section does not contain %q", want)
		}
	}
	for _, forbidden := range []string{"Binance Pay", "云闪付", "Wise", "Patreon"} {
		if strings.Contains(readme, forbidden) {
			t.Errorf("README unexpectedly contains %q", forbidden)
		}
	}
}
