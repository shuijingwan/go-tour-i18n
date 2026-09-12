package tour

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHomepageSupportCopyAndResponsiveLayoutInBrowser(t *testing.T) {
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}

	for _, test := range []struct {
		name       string
		locale     string
		windowSize string
		script     string
	}{
		{
			name:       "zh-CN-mobile",
			locale:     "zh-CN",
			windowSize: "375,1200",
			script: `(function() {
  Object.defineProperty(navigator, 'clipboard', {configurable: true, value: {writeText: function(value) {
    document.documentElement.setAttribute('data-support-copied-value', value); return Promise.resolve();
  }}});
  var button = document.querySelector('[data-copy-value="go-dev-zh-CN"]');
  button.click();
  setTimeout(function() {
    var cards = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card'));
    var qrs = Array.prototype.slice.call(document.querySelectorAll('.support-qr'));
    var fits = document.documentElement.scrollWidth <= document.documentElement.clientWidth &&
      cards.length === 2 && qrs.length === 2 && cards.every(function(card) {
        var rect = card.getBoundingClientRect(); return rect.left >= 0 && rect.right <= document.documentElement.clientWidth;
      }) && qrs.every(function(qr) {
        var image = qr.getBoundingClientRect(); var card = qr.closest('.support-payment-card').getBoundingClientRect();
        return qr.complete && qr.naturalWidth > 0 && image.width <= card.width && image.right <= document.documentElement.clientWidth;
      });
    if (fits && button.textContent === '已复制' && document.documentElement.getAttribute('data-support-copied-value') === 'go-dev-zh-CN') {
      document.documentElement.setAttribute('data-support-browser', 'PASS');
    }
  }, 100);
}());`,
		},
		{
			name:       "zh-CN-desktop",
			locale:     "zh-CN",
			windowSize: "1200,1000",
			script: `(function() {
  setTimeout(function() {
    var identities = ['微信支付', '支付宝'];
    var cards = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card'));
    var frames = Array.prototype.slice.call(document.querySelectorAll('.support-qr-frame'));
    var qrs = Array.prototype.slice.call(document.querySelectorAll('.support-qr'));
    var aligned = document.documentElement.scrollWidth <= document.documentElement.clientWidth &&
      cards.length === 2 && frames.length === 2 && qrs.length === 2 &&
      cards.every(function(card, index) {
        return !card.querySelector('h3') && card.textContent.trim() === '' &&
          card.getAttribute('aria-label') === identities[index] && qrs[index].getAttribute('alt') === identities[index];
      }) &&
      Math.abs(frames[0].getBoundingClientRect().width - frames[1].getBoundingClientRect().width) <= 1 &&
      Math.abs(frames[0].getBoundingClientRect().height - frames[1].getBoundingClientRect().height) <= 1 &&
      Math.abs(frames[0].getBoundingClientRect().top - frames[1].getBoundingClientRect().top) <= 1 &&
      qrs.every(function(qr, index) {
        var card = cards[index].getBoundingClientRect();
        var frame = frames[index].getBoundingClientRect();
        var image = qr.getBoundingClientRect();
        var frameCenterX = frame.left + frame.width / 2;
        var frameCenterY = frame.top + frame.height / 2;
        return qr.complete && qr.naturalWidth > 0 && qr.naturalHeight > 0 &&
          image.left >= frame.left && image.right <= frame.right && image.top >= frame.top && image.bottom <= frame.bottom &&
          Math.abs((image.width / image.height) - (qr.naturalWidth / qr.naturalHeight)) <= 0.01 &&
          Math.abs((image.left + image.width / 2) - frameCenterX) <= 1 &&
          Math.abs((image.top + image.height / 2) - frameCenterY) <= 1 &&
          Math.abs(frameCenterX - (card.left + card.width / 2)) <= 1;
      });
    if (aligned) {
      document.documentElement.setAttribute('data-support-browser', 'PASS');
    }
  }, 100);
}());`,
		},
		{
			name:       "ja-JP-mobile",
			locale:     "ja-JP",
			windowSize: "375,1200",
			script: `(function() {
  var copiedValues = [];
  Object.defineProperty(navigator, 'clipboard', {configurable: true, value: {writeText: function(value) {
    copiedValues.push(value); document.documentElement.setAttribute('data-support-copied-value', copiedValues.join('|')); return Promise.resolve();
  }}});
  var addresses = ['0x225f14d54683b1f5bc153bc8a678cad0277096d3', 'TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK'];
  var buttons = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card .support-copy'));
  buttons.forEach(function(button) { button.click(); });
  setTimeout(function() {
    var cards = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card'));
    var codes = Array.prototype.slice.call(document.querySelectorAll('.support-address'));
    var fits = document.documentElement.scrollWidth <= document.documentElement.clientWidth && cards.length === 2 &&
      cards[1].getBoundingClientRect().top >= cards[0].getBoundingClientRect().bottom &&
      cards.every(function(card) { var rect = card.getBoundingClientRect(); return rect.left >= 0 && rect.right <= document.documentElement.clientWidth; }) &&
      codes.length === addresses.length && codes.every(function(code, index) {
        var image = code.getBoundingClientRect();
        var card = code.closest('.support-payment-card').getBoundingClientRect();
        return code.textContent === addresses[index] && image.left >= card.left && image.right <= card.right;
      });
    if (fits && buttons.length === 2 && buttons.every(function(button) { return button.textContent === 'コピーしました'; }) &&
        document.documentElement.getAttribute('data-support-copied-value') === addresses.join('|')) {
      document.documentElement.setAttribute('data-support-browser', 'PASS');
    }
  }, 100);
}());`,
		},
		{
			name:       "ja-JP-desktop",
			locale:     "ja-JP",
			windowSize: "1200,1000",
			script: `(function() {
  var copiedValues = [];
  Object.defineProperty(navigator, 'clipboard', {configurable: true, value: {writeText: function(value) {
    copiedValues.push(value); document.documentElement.setAttribute('data-support-copied-value', copiedValues.join('|')); return Promise.resolve();
  }}});
  function textLineCount(element) {
    var range = document.createRange();
    range.selectNodeContents(element);
    return range.getClientRects().length;
  }
  var addresses = ['0x225f14d54683b1f5bc153bc8a678cad0277096d3', 'TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK'];
  var buttons = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card .support-copy'));
  buttons.forEach(function(button) { button.click(); });
  setTimeout(function() {
    var cards = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card'));
    var readable = document.documentElement.scrollWidth <= document.documentElement.clientWidth && cards.length === 2 &&
      Math.abs(cards[0].getBoundingClientRect().width - cards[1].getBoundingClientRect().width) <= 1 &&
      Math.abs(cards[0].getBoundingClientRect().top - cards[1].getBoundingClientRect().top) <= 1 &&
      cards.every(function(card, cardIndex) {
        var details = card.querySelector('.support-payment-details');
        var terms = Array.prototype.slice.call(details.querySelectorAll('dt'));
        var values = Array.prototype.slice.call(details.querySelectorAll('dd'));
        var address = details.querySelector('.support-address');
        var network = values[1];
        var warning = card.querySelector('.support-warning');
        var cardRect = card.getBoundingClientRect();
        var detailsRect = details.getBoundingClientRect();
        var addressRect = address.getBoundingClientRect();
        return terms.length === 4 && values.length === 4 &&
          values.every(function(value, index) {
            var termRect = terms[index].getBoundingClientRect();
            var valueRect = value.getBoundingClientRect();
            return valueRect.width >= detailsRect.width * 0.9 &&
              Math.abs(valueRect.left - termRect.left) <= 1 && valueRect.top >= termRect.bottom;
          }) &&
          address.textContent === addresses[cardIndex] && addressRect.width >= detailsRect.width * 0.9 &&
          addressRect.left >= cardRect.left && addressRect.right <= cardRect.right &&
          address.scrollWidth <= address.clientWidth && textLineCount(address) === 1 &&
          parseFloat(getComputedStyle(address).fontSize) >= parseFloat(getComputedStyle(document.body).fontSize) * 0.9 &&
          (network.textContent === 'Base' || network.textContent === 'Tron (TRC20)') && textLineCount(network) === 1 &&
          warning.getBoundingClientRect().left >= cardRect.left && warning.getBoundingClientRect().right <= cardRect.right &&
          warning.scrollWidth <= warning.clientWidth;
      });
    if (readable && buttons.length === 2 && buttons.every(function(button) { return button.textContent === 'コピーしました'; }) &&
        document.documentElement.getAttribute('data-support-copied-value') === addresses.join('|')) {
      document.documentElement.setAttribute('data-support-browser', 'PASS');
    }
  }, 100);
}());`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := productionTestHandlerLocale(t, "http://127.0.0.1:1", test.locale)
			instrumented := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/" {
					handler.ServeHTTP(w, r)
					return
				}
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, r)
				for key, values := range recorder.Header() {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				w.WriteHeader(recorder.Code)
				injection := []byte("<script>" + test.script + "</script></body>")
				_, _ = w.Write(bytes.Replace(recorder.Body.Bytes(), []byte("</body>"), injection, 1))
			})
			listener, err := net.Listen("tcp4", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewUnstartedServer(instrumented)
			server.Listener = listener
			server.Start()
			defer server.Close()

			command := exec.Command(chrome,
				"--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage",
				"--disable-breakpad", "--disable-crash-reporter", "--disable-background-networking",
				"--disable-default-apps", "--disable-extensions", "--no-first-run", "--noerrdialogs",
				"--window-size="+test.windowSize, "--force-device-scale-factor=1", "--virtual-time-budget=3000",
				"--user-data-dir="+filepath.Join(t.TempDir(), "chrome-profile"), "--dump-dom", server.URL+"/")
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("google-chrome: %v\n%s", err, output)
			}
			if !bytes.Contains(output, []byte(`data-support-browser="PASS"`)) {
				t.Fatalf("support copy/responsive browser acceptance failed:\n%s", output)
			}
		})
	}
}
