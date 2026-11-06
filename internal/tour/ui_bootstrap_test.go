package tour

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

func TestSharedUIMessagesBootstrapContract(t *testing.T) {
	runtime, err := fs.ReadFile(contentTour, "js/playground.js")
	if err != nil {
		t.Fatal(err)
	}
	consumers := regexp.MustCompile(`window\.__tourUIMessages\['([^']+)'\]`).FindAllSubmatch(runtime, -1)
	if len(consumers) == 0 {
		t.Fatal("missing runtime message consumers")
	}
	for _, locale := range []string{"zh-CN", "fr-FR"} {
		catalog, err := ui.Load(locale)
		if err != nil {
			t.Fatal(err)
		}
		got, err := UIMessagesBootstrap(locale)
		if err != nil {
			t.Fatal(err)
		}
		want, err := jsI18nBootstrap(catalog)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("shared bootstrap differs from original Tour bytes", err)
		}
		var messages map[string]string
		body := strings.TrimSuffix(strings.TrimPrefix(string(got), "window.__tourUIMessages = "), ";\n")
		if err := json.Unmarshal([]byte(body), &messages); err != nil {
			t.Fatal(err)
		}
		if len(messages) != len(jsI18nKeys) {
			t.Fatal("shared key set differs from Tour authority")
		}
		for _, match := range consumers {
			key := string(match[1])
			if value, err := catalog.Plain(key); err != nil || messages[key] != value || messages[key] == "" {
				t.Fatal("runtime message not bound to target locale", locale, key, err)
			}
		}
	}
}

func TestSharedUIMessagesBootstrapFailClosed(t *testing.T) {
	if _, err := UIMessagesBootstrap("zz-UNKNOWN"); err == nil {
		t.Fatal("unsupported locale fell back to English")
	}
	catalog, err := ui.Load("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range jsI18nKeys {
		message := catalog.Messages[key]
		delete(catalog.Messages, key)
		if _, err := jsI18nBootstrap(catalog); err == nil {
			t.Fatal("missing runtime message accepted", key)
		}
		catalog.Messages[key] = message
	}
}

func TestSharedUIMessagesBootstrapScriptEscaping(t *testing.T) {
	catalog, err := ui.Load("zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	const text = "quotes: \" ' <>& </script><script>window.injected=true</script> \u2028 \u2029"
	catalog.Messages["execution.exited"] = ui.Message{Kind: "plain", Text: text}
	bootstrap, err := jsI18nBootstrap(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.ContainsAny(bootstrap, "<>&") || !bytes.Contains(bootstrap, []byte(`\u003c/script\u003e`)) {
		t.Fatal("bootstrap is unsafe inside an HTML script")
	}
	cmd := exec.Command("node", "-e", `const vm=require('node:vm'),assert=require('node:assert/strict');const p=JSON.parse(require('node:fs').readFileSync(0,'utf8'));const c={window:{}};vm.createContext(c);vm.runInContext(p.code,c);assert.equal(c.window.__tourUIMessages['execution.exited'],p.text);assert.equal(c.window.injected,undefined);`)
	input, _ := json.Marshal(map[string]string{"code": string(bootstrap), "text": text})
	cmd.Stdin = bytes.NewReader(input)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("safe bootstrap execution: %v\n%s", err, out)
	}
}
