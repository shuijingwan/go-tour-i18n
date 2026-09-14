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
	const visualViewportFixture = `(function() {
  var listeners = {resize: [], scroll: []};
  var nativeSetTimeout = window.setTimeout;
  window.supportTestToastTimerDelays = [];
  window.setTimeout = function(callback, delay) {
    if (delay === 1600) {
      window.supportTestToastTimerDelays.push(delay);
      return nativeSetTimeout(callback, 60000);
    }
    return nativeSetTimeout(callback, delay);
  };
  var viewport = {
    offsetTop: 0, offsetLeft: 0, width: 375, height: 800,
    addEventListener: function(type, listener) { listeners[type].push(listener); },
    setGeometry: function(geometry) {
      Object.keys(geometry).forEach(function(key) { viewport[key] = geometry[key]; });
      listeners.resize.forEach(function(listener) { listener(); });
      listeners.scroll.forEach(function(listener) { listener(); });
    }
  };
  Object.defineProperty(viewport, 'pageTop', {get: function() { return window.scrollY + viewport.offsetTop; }});
  Object.defineProperty(viewport, 'pageLeft', {get: function() { return window.scrollX + viewport.offsetLeft; }});
  Object.defineProperty(window, 'visualViewport', {configurable: true, value: viewport});
  window.supportTestVisualViewport = viewport;
}());`

	for _, test := range []struct {
		name         string
		locale       string
		windowSize   string
		beforeScript string
		script       string
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
    var toast = document.querySelector('[data-copy-toast]');
    var cards = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card'));
    var qrs = Array.prototype.slice.call(document.querySelectorAll('.support-qr'));
    var fits = document.documentElement.scrollWidth <= document.documentElement.clientWidth &&
      cards.length === 2 && qrs.length === 2 && cards.every(function(card) {
        var rect = card.getBoundingClientRect(); return rect.left >= 0 && rect.right <= document.documentElement.clientWidth;
      }) && qrs.every(function(qr) {
        var image = qr.getBoundingClientRect(); var card = qr.closest('.support-payment-card').getBoundingClientRect();
        return qr.complete && qr.naturalWidth > 0 && image.width <= card.width && image.right <= document.documentElement.clientWidth;
      });
    var buttonRect = button.getBoundingClientRect();
    var toastRect = toast.getBoundingClientRect();
    var toastBottomGap = window.innerHeight - toastRect.bottom;
    if (fits && buttonRect.width >= 40 && buttonRect.height >= 40 && button.querySelector('.support-copy-icon') &&
        button.textContent.trim() === '' && toast && !toast.hidden && toast.textContent === '已复制' &&
        toastRect.top >= 0 && toastRect.bottom <= window.innerHeight && toastBottomGap >= 16 &&
        document.querySelectorAll('[data-copy-toast]').length === 1 &&
        document.documentElement.getAttribute('data-support-copied-value') === 'go-dev-zh-CN') {
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
  document.documentElement.setAttribute('data-theme', 'dark');
  Object.defineProperty(navigator, 'clipboard', {configurable: true, value: {writeText: function() {
    return Promise.resolve();
  }}});
  document.querySelector('[data-copy-value="go-dev-zh-CN"]').click();
  setTimeout(function() {
    var identities = ['微信支付', '支付宝'];
    var cards = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card'));
    var frames = Array.prototype.slice.call(document.querySelectorAll('.support-qr-frame'));
    var qrs = Array.prototype.slice.call(document.querySelectorAll('.support-qr'));
    var copy = document.querySelector('.support-copy');
    var toast = document.querySelector('[data-copy-toast]');
    var toastRect = toast.getBoundingClientRect();
    var toastBottomGap = window.innerHeight - toastRect.bottom;
    var aligned = document.documentElement.scrollWidth <= document.documentElement.clientWidth &&
      cards.length === 2 && frames.length === 2 && qrs.length === 2 &&
      toast && !toast.hidden && toastRect.top >= 0 && toastRect.bottom <= window.innerHeight &&
      Math.abs(toastBottomGap - 24) <= 1 &&
      getComputedStyle(copy).color === 'rgb(216, 220, 222)' &&
      getComputedStyle(cards[0]).backgroundColor === 'rgb(32, 34, 36)' &&
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
			name:         "ja-JP-mobile",
			locale:       "ja-JP",
			windowSize:   "375,1200",
			beforeScript: visualViewportFixture,
			script: `(function() {
  var copiedValues = [];
  Object.defineProperty(navigator, 'clipboard', {configurable: true, value: {writeText: function(value) {
    copiedValues.push(value); document.documentElement.setAttribute('data-support-copied-value', copiedValues.join('|')); return Promise.resolve();
  }}});
  var values = ['go-dev-ja-JP', '0x225f14d54683b1f5bc153bc8a678cad0277096d3', 'TF2bM817pLQeN1Ykt3GEecRbTjuSsWtGdK', '1055351242', '231321605530361856'];
  var buttons = Array.prototype.slice.call(document.querySelectorAll('[data-copy-value]'));
  var toast = document.querySelector('[data-copy-toast]');
  var section = document.querySelector('.site-support');
  var initialHeight = document.documentElement.scrollHeight;
  var addressValues = values.slice(1, 3);
  var addressNodes = Array.prototype.slice.call(document.querySelectorAll('.support-address'));
  var positions = [
    {scroll: function() { window.scrollTo(0, section.offsetTop + section.offsetHeight / 2 - window.innerHeight / 2); }, viewport: {offsetTop: 0, offsetLeft: 0, width: 375, height: window.innerHeight - 160}},
    {scroll: function() { window.scrollTo(0, document.documentElement.scrollHeight - window.innerHeight - 160); }, viewport: {offsetTop: 12, offsetLeft: 0, width: 375, height: window.innerHeight - 60}},
    {scroll: function() { window.scrollTo(0, document.documentElement.scrollHeight); }, viewport: {offsetTop: 0, offsetLeft: 0, width: 375, height: window.innerHeight}}
  ];
  function frame() { return new Promise(function(resolve) { requestAnimationFrame(resolve); }); }
  function toastFitsVisualViewport() {
    var rect = toast.getBoundingClientRect();
    var viewport = window.visualViewport;
    var visualBottom = viewport.offsetTop + viewport.height;
    return !toast.hidden && rect.top >= viewport.offsetTop && rect.bottom <= visualBottom &&
      visualBottom - rect.bottom >= 16 &&
      rect.left >= viewport.offsetLeft && rect.right <= viewport.offsetLeft + viewport.width &&
      Math.abs(rect.left + rect.width / 2 - (viewport.offsetLeft + viewport.width / 2)) <= 1;
  }
  function waitForToast(copy) {
    return new Promise(function(resolve) {
      var observer = new MutationObserver(function() {
        if (!toast.hidden) {
          observer.disconnect();
          resolve();
        }
      });
      observer.observe(toast, {attributes: true, attributeFilter: ['hidden']});
      copy();
      if (!toast.hidden) {
        observer.disconnect();
        resolve();
      }
    });
  }
  function verifyPosition(checkFirstFrame) {
    return waitForToast(function() {
      buttons.forEach(function(button) { button.click(); });
    }).then(function() {
      return checkFirstFrame ? frame() : Promise.resolve();
    }).then(function() {
      if (!toastFitsVisualViewport()) {
        var rect = toast.getBoundingClientRect();
        throw new Error('toast outside visual viewport on first frame: ' + JSON.stringify({rect: {top: rect.top, right: rect.right, bottom: rect.bottom, left: rect.left}, viewport: window.visualViewport, innerHeight: window.innerHeight, hidden: toast.hidden}));
      }
      return delay(100);
    }).then(function() {
      if (!toastFitsVisualViewport()) {
        var rect = toast.getBoundingClientRect();
        throw new Error('toast outside visual viewport when stable: ' + JSON.stringify({rect: {top: rect.top, right: rect.right, bottom: rect.bottom, left: rect.left}, viewport: window.visualViewport, innerHeight: window.innerHeight, hidden: toast.hidden}));
      }
    });
  }
  function delay(milliseconds) { return new Promise(function(resolve) { setTimeout(resolve, milliseconds); }); }
  var sequence = Promise.resolve();
  positions.forEach(function(position, index) {
    sequence = sequence.then(function() {
      position.scroll();
      window.supportTestVisualViewport.setGeometry(position.viewport);
    });
    sequence = sequence.then(function() { return verifyPosition(index === 0); });
  });
  sequence.then(function() {
    var cards = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card'));
    var platformCards = Array.prototype.slice.call(document.querySelectorAll('.support-platform-card'));
    var codes = Array.prototype.slice.call(document.querySelectorAll('.support-address, .support-uid'));
    var fits = document.documentElement.scrollWidth <= document.documentElement.clientWidth && cards.length === 2 &&
      cards[1].getBoundingClientRect().top >= cards[0].getBoundingClientRect().bottom &&
      platformCards.length === 2 && platformCards[1].getBoundingClientRect().top >= platformCards[0].getBoundingClientRect().bottom &&
      cards.every(function(card) { var rect = card.getBoundingClientRect(); return rect.left >= 0 && rect.right <= document.documentElement.clientWidth; }) &&
      platformCards.every(function(card) { var rect = card.getBoundingClientRect(); return rect.left >= 0 && rect.right <= document.documentElement.clientWidth; }) &&
      codes.length === 4 && codes.every(function(code) {
        var image = code.getBoundingClientRect();
        var card = (code.closest('.support-payment-card') || code.closest('.support-platform-card')).getBoundingClientRect();
        return image.left >= card.left && image.right <= card.right && code.scrollWidth <= code.clientWidth;
      }) && addressNodes.length === 2 && addressNodes.every(function(address, index) {
        var style = getComputedStyle(address);
        return address.textContent === addressValues[index] && style.whiteSpace !== 'nowrap' && style.overflowWrap === 'anywhere';
      });
    if (!fits || buttons.length !== 5 || buttons.some(function(button) {
          var rect = button.getBoundingClientRect();
          return !button.querySelector('.support-copy-icon') || button.textContent.trim() !== '' || rect.width < 40 || rect.height < 40;
        }) || toast.textContent !== 'コピーしました' || document.querySelectorAll('[data-copy-toast]').length !== 1 ||
        document.documentElement.scrollHeight !== initialHeight || copiedValues.length !== positions.length * values.length ||
        window.supportTestToastTimerDelays.length !== positions.length * values.length ||
        window.supportTestToastTimerDelays.some(function(delay) { return delay !== 1600; })) {
      throw new Error('support mobile layout regression');
    }
    if (toastFitsVisualViewport() && !toast.hidden && document.querySelectorAll('[data-copy-toast]').length === 1) {
      document.documentElement.setAttribute('data-support-browser', 'PASS');
    }
  }).catch(function(error) {
    document.documentElement.setAttribute('data-support-browser-error', error.message);
  });
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
  var copied = ['go-dev-ja-JP'].concat(addresses, ['1055351242', '231321605530361856']);
  var buttons = Array.prototype.slice.call(document.querySelectorAll('[data-copy-value]'));
  buttons.forEach(function(button) { button.click(); });
  setTimeout(function() {
    var cards = Array.prototype.slice.call(document.querySelectorAll('.support-payment-card'));
    var platformCards = Array.prototype.slice.call(document.querySelectorAll('.support-platform-card'));
    var toast = document.querySelector('[data-copy-toast]');
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
          address.textContent === addresses[cardIndex] && addressRect.width >= detailsRect.width - 54 &&
          addressRect.width >= 390 &&
          addressRect.left >= cardRect.left && addressRect.right <= cardRect.right &&
          address.scrollWidth <= address.clientWidth && textLineCount(address) === 1 &&
          parseFloat(getComputedStyle(address).fontSize) >= parseFloat(getComputedStyle(document.body).fontSize) * 0.9 &&
          (network.textContent === 'Base' || network.textContent === 'Tron (TRC20)') && textLineCount(network) === 1 &&
          warning.getBoundingClientRect().left >= cardRect.left && warning.getBoundingClientRect().right <= cardRect.right &&
          warning.scrollWidth <= warning.clientWidth;
      }) && platformCards.length === 2 &&
      Math.abs(platformCards[0].getBoundingClientRect().width - platformCards[1].getBoundingClientRect().width) <= 1 &&
      platformCards.every(function(card) {
        var uid = card.querySelector('.support-uid');
        var button = card.querySelector('.support-copy');
        return uid.scrollWidth <= uid.clientWidth && button.getBoundingClientRect().width >= 40;
      });
    if (!readable || buttons.length !== 5 || buttons.some(function(button) { return !button.querySelector('.support-copy-icon') || button.textContent.trim() !== ''; }) ||
        !toast || toast.hidden || toast.textContent !== 'コピーしました' || document.querySelectorAll('[data-copy-toast]').length !== 1 ||
        document.documentElement.getAttribute('data-support-copied-value') !== copied.join('|')) return;
    buttons[0].click();
    setTimeout(function() {
      buttons[1].click();
      setTimeout(function() {
        if (!toast.hidden && toast.textContent === 'コピーしました' && document.querySelectorAll('[data-copy-toast]').length === 1) {
          document.documentElement.setAttribute('data-support-browser', 'PASS');
        }
      }, 500);
    }, 1200);
  }, 100);
}());`,
		},
		{
			name:         "ja-JP-copy-fallback-and-failure",
			locale:       "ja-JP",
			windowSize:   "375,900",
			beforeScript: visualViewportFixture,
			script: `(function() {
  var fallbackWorks = true;
  var fallbackGeometry;
  Object.defineProperty(navigator, 'clipboard', {configurable: true, value: {writeText: function() {
    return Promise.reject(new Error('controlled clipboard failure'));
  }}});
  document.execCommand = function() {
    fallbackGeometry = {
      scrollY: window.scrollY,
      activeElement: document.activeElement && document.activeElement.tagName,
      visualBottom: window.visualViewport.offsetTop + window.visualViewport.height
    };
    return fallbackWorks;
  };
  var buttons = Array.prototype.slice.call(document.querySelectorAll('[data-copy-value]'));
  var toast = document.querySelector('[data-copy-toast]');
  var icon = buttons[3].querySelector('.support-copy-icon');
  var section = document.querySelector('.site-support');
  window.scrollTo(0, section.offsetTop + section.offsetHeight / 2 - window.innerHeight / 2);
  window.supportTestVisualViewport.setGeometry({offsetTop: 8, offsetLeft: 0, width: 375, height: window.innerHeight - 120});
  var initialScrollY = window.scrollY;
  var initialVisualBottom = window.visualViewport.offsetTop + window.visualViewport.height;
  function verifyVisibleToast() {
    requestAnimationFrame(function() {
    var firstRect = toast.getBoundingClientRect();
    requestAnimationFrame(function() {
      var stableRect = toast.getBoundingClientRect();
      var visualBottom = window.visualViewport.offsetTop + window.visualViewport.height;
      var fallbackStable = fallbackGeometry && fallbackGeometry.scrollY === initialScrollY &&
        fallbackGeometry.activeElement === 'TEXTAREA' && fallbackGeometry.visualBottom === initialVisualBottom &&
        window.scrollY === initialScrollY && visualBottom === initialVisualBottom &&
        firstRect.top >= window.visualViewport.offsetTop && firstRect.bottom <= visualBottom &&
        stableRect.top >= window.visualViewport.offsetTop && stableRect.bottom <= visualBottom;
      if (!fallbackStable || !toast || toast.hidden || toast.textContent !== 'コピーしました' || !icon.isConnected) return;
      fallbackWorks = false;
      buttons[4].click();
      setTimeout(function() {
        if (toast.hidden && toast.textContent === '' && icon.isConnected && buttons[4].querySelector('.support-copy-icon')) {
          document.documentElement.setAttribute('data-support-browser', 'PASS');
        }
      }, 100);
    });
  });
  }
  var observer = new MutationObserver(function() {
    if (!toast.hidden) {
      observer.disconnect();
      verifyVisibleToast();
    }
  });
  observer.observe(toast, {attributes: true, attributeFilter: ['hidden']});
  buttons[3].click();
  if (!toast.hidden) {
    observer.disconnect();
    verifyVisibleToast();
  }
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
				body := recorder.Body.Bytes()
				if test.beforeScript != "" {
					injection := []byte("<script>" + test.beforeScript + "</script></head>")
					body = bytes.Replace(body, []byte("</head>"), injection, 1)
				}
				injection := []byte("<script>" + test.script + "</script></body>")
				_, _ = w.Write(bytes.Replace(body, []byte("</body>"), injection, 1))
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
				"--window-size="+test.windowSize, "--force-device-scale-factor=1", "--virtual-time-budget=15000",
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
