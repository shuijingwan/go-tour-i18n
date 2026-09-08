package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBootstrapIndexNowSubmitsProbeThenRemainingSitemapURLs(t *testing.T) {
	key := "index-now-test-key"
	var submissions []indexNowSubmission
	client := indexNowTestClient(func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case "/" + key + ".txt":
			return http.StatusOK, key
		case "/sitemap.xml":
			return http.StatusOK, testIndexNowSitemap("https://locale.example/", 3)
		case "/indexnow":
			var submission indexNowSubmission
			if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
				t.Fatal(err)
			}
			submissions = append(submissions, submission)
			return http.StatusOK, ""
		default:
			return http.StatusNotFound, ""
		}
	})
	profile := indexNowProfile{Locale: "zz-ZZ", State: "live", Hostname: "locale.example", PublicURL: "https://locale.example/"}
	result, err := bootstrapIndexNow(context.Background(), client, "https://api.indexnow.org/indexnow", profile, key)
	if err != nil {
		t.Fatal(err)
	}
	if result.SitemapURLs != 3 || result.SubmittedURLs != 3 {
		t.Fatalf("result = %+v", result)
	}
	if len(submissions) != 2 {
		t.Fatalf("submissions = %d, want 2", len(submissions))
	}
	if got := submissions[0].URLList; len(got) != 1 || got[0] != "https://locale.example/" {
		t.Fatalf("probe URLs = %v", got)
	}
	if got := submissions[1].URLList; len(got) != 2 || containsURL(got, "https://locale.example/") {
		t.Fatalf("bulk URLs = %d, probe included=%v", len(got), containsURL(got, "https://locale.example/"))
	}
	for _, submission := range submissions {
		if submission.Host != profile.Hostname || submission.Key != key || submission.KeyLocation != "https://locale.example/"+key+".txt" {
			t.Fatalf("submission = %+v", submission)
		}
	}
}

func TestBootstrapIndexNowCountsSingleProbeURLAsSubmitted(t *testing.T) {
	key := "index-now-test-key"
	posts := 0
	client := indexNowTestClient(func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case "/" + key + ".txt":
			return http.StatusOK, key
		case "/sitemap.xml":
			return http.StatusOK, testIndexNowSitemap("https://locale.example/", 1)
		case "/indexnow":
			posts++
			return http.StatusOK, ""
		}
		return http.StatusNotFound, ""
	})
	result, err := bootstrapIndexNow(context.Background(), client, "https://api.indexnow.org/indexnow", indexNowProfile{Locale: "zz-ZZ", State: "live", Hostname: "locale.example", PublicURL: "https://locale.example/"}, key)
	if err != nil {
		t.Fatal(err)
	}
	if result.SitemapURLs != 1 || result.SubmittedURLs != 1 || posts != 1 {
		t.Fatalf("result = %+v, posts = %d", result, posts)
	}
}

func TestBootstrapIndexNowFailsWhenSitemapLacksFormalRootProbe(t *testing.T) {
	key := "index-now-test-key"
	posts := 0
	client := indexNowTestClient(func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case "/" + key + ".txt":
			return http.StatusOK, key
		case "/sitemap.xml":
			return http.StatusOK, `<?xml version="1.0"?><urlset><url><loc>https://locale.example/tour/test/1</loc></url></urlset>`
		case "/indexnow":
			posts++
			return http.StatusOK, ""
		}
		return http.StatusNotFound, ""
	})
	_, err := bootstrapIndexNow(context.Background(), client, "https://api.indexnow.org/indexnow", indexNowProfile{Locale: "zz-ZZ", State: "live", Hostname: "locale.example", PublicURL: "https://locale.example/"}, key)
	if err == nil || !strings.Contains(err.Error(), "does not contain fixed probe URL https://locale.example/") {
		t.Fatalf("err = %v", err)
	}
	if posts != 0 {
		t.Fatalf("posts = %d, want 0", posts)
	}
}

func TestBootstrapIndexNowStopsWhenProbeIsPending(t *testing.T) {
	key := "index-now-test-key"
	posts := 0
	client := indexNowTestClient(func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case "/" + key + ".txt":
			return http.StatusOK, key
		case "/sitemap.xml":
			return http.StatusOK, testIndexNowSitemap("https://locale.example/", 3)
		case "/indexnow":
			posts++
			return http.StatusAccepted, ""
		}
		return http.StatusNotFound, ""
	})
	_, err := bootstrapIndexNow(context.Background(), client, "https://api.indexnow.org/indexnow", indexNowProfile{Locale: "zz-ZZ", State: "live", Hostname: "locale.example", PublicURL: "https://locale.example/"}, key)
	if err == nil || !strings.Contains(err.Error(), "probe is pending") {
		t.Fatalf("err = %v", err)
	}
	if posts != 1 {
		t.Fatalf("posts = %d, want 1", posts)
	}
}

func TestFetchIndexNowSitemapRejectsEmptySitemap(t *testing.T) {
	client := indexNowTestClient(func(*http.Request) (int, string) {
		return http.StatusOK, `<?xml version="1.0"?><urlset></urlset>`
	})
	_, err := fetchIndexNowSitemap(context.Background(), client, "https://locale.example/sitemap.xml", "locale.example", "https://locale.example")
	if err == nil || !strings.Contains(err.Error(), "URL count") {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchIndexNowSitemapRejectsDuplicateURL(t *testing.T) {
	client := indexNowTestClient(func(*http.Request) (int, string) {
		return http.StatusOK, `<?xml version="1.0"?><urlset><url><loc>https://locale.example</loc></url><url><loc>https://locale.example</loc></url></urlset>`
	})
	_, err := fetchIndexNowSitemap(context.Background(), client, "https://locale.example/sitemap.xml", "locale.example", "https://locale.example")
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchIndexNowSitemapRejectsMoreThanIndexNowLimit(t *testing.T) {
	client := indexNowTestClient(func(*http.Request) (int, string) {
		return http.StatusOK, testIndexNowSitemap("https://locale.example/", indexNowMaxURLs+1)
	})
	_, err := fetchIndexNowSitemap(context.Background(), client, "https://locale.example/sitemap.xml", "locale.example", "https://locale.example")
	if err == nil || !strings.Contains(err.Error(), "1..10000") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadIndexNowLiveProfileRejectsNonLiveLocale(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity.json")
	if err := os.WriteFile(path, []byte(`{"locales":[{"locale":"zz-ZZ","production_state":"first-production","production_hostname":"locale.example","production_public_url":"https://locale.example/"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := readIndexNowLiveProfile(path, "zz-ZZ")
	if err == nil || !strings.Contains(err.Error(), "production_state=live") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadIndexNowKeyRequiresMatchingPublicKeyFilename(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wrong-name.txt")
	if err := os.WriteFile(path, []byte("private-key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := readIndexNowKey(path)
	if err == nil || !strings.Contains(err.Error(), "<key>.txt") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadIndexNowKeyRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("private-key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "private-key.txt")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := readIndexNowKey(path); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadIndexNowKeyRejectsBareCarriageReturn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-key.txt")
	if err := os.WriteFile(path, []byte("private-key\r"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readIndexNowKey(path); err == nil || !strings.Contains(err.Error(), "one non-empty line") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadIndexNowKeyLengthAndCharacterContract(t *testing.T) {
	for _, test := range []struct {
		name  string
		key   string
		valid bool
	}{
		{"minimum", strings.Repeat("a", 8), true},
		{"maximum", strings.Repeat("z", 128), true},
		{"too short", strings.Repeat("a", 7), false},
		{"too long", strings.Repeat("a", 129), false},
		{"underscore", "test_key", false},
		{"whitespace", "test key", false},
		{"slash", "test/key", false},
		{"punctuation", "test.key", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), strings.ReplaceAll(test.key, "/", "-")+".txt")
			if err := os.WriteFile(path, []byte(test.key+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := readIndexNowKey(path)
			if (err == nil) != test.valid {
				t.Fatalf("readIndexNowKey(%q) err=%v, valid=%v", test.key, err, test.valid)
			}
		})
	}
}

func TestIndexNowSafeGETRetriesOnlyTransientReadFailures(t *testing.T) {
	previousSleep := indexNowRetrySleep
	indexNowRetrySleep = func(time.Duration) {}
	t.Cleanup(func() { indexNowRetrySleep = previousSleep })
	t.Run("transient server", func(t *testing.T) {
		attempts := 0
		client := indexNowTestClient(func(*http.Request) (int, string) {
			attempts++
			if attempts < 3 {
				return http.StatusServiceUnavailable, ""
			}
			return http.StatusOK, "ok"
		})
		request, _ := http.NewRequest(http.MethodGet, "https://locale.example/sitemap.xml", nil)
		response, err := indexNowSafeGET(client, request)
		if err != nil || response.StatusCode != http.StatusOK || attempts != 3 {
			t.Fatalf("status=%v err=%v attempts=%d", response, err, attempts)
		}
		response.Body.Close()
	})
	t.Run("transient transport", func(t *testing.T) {
		attempts := 0
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts < 3 {
				return nil, &net.DNSError{IsTimeout: true}
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
		})}
		request, _ := http.NewRequest(http.MethodGet, "https://locale.example/key.txt", nil)
		response, err := indexNowSafeGET(client, request)
		if err != nil || response.StatusCode != http.StatusOK || attempts != 3 {
			t.Fatalf("status=%v err=%v attempts=%d", response, err, attempts)
		}
		response.Body.Close()
	})
	t.Run("semantic status does not retry", func(t *testing.T) {
		attempts := 0
		client := indexNowTestClient(func(*http.Request) (int, string) { attempts++; return http.StatusTooManyRequests, "" })
		request, _ := http.NewRequest(http.MethodGet, "https://locale.example/sitemap.xml", nil)
		response, err := indexNowSafeGET(client, request)
		if err != nil || response.StatusCode != http.StatusTooManyRequests || attempts != 1 {
			t.Fatalf("status=%v err=%v attempts=%d", response, err, attempts)
		}
		response.Body.Close()
	})
}

func TestRequireIndexNowPublicKeyAcceptsOnlyOneOptionalLineEnding(t *testing.T) {
	key := "index-now-test-key"
	for _, test := range []struct {
		name string
		body string
		want bool
	}{
		{name: "no newline", body: key, want: true},
		{name: "LF", body: key + "\n", want: true},
		{name: "CRLF", body: key + "\r\n", want: true},
		{name: "extra content", body: key + "extra", want: false},
		{name: "extra line", body: key + "\nextra", want: false},
		{name: "two line endings", body: key + "\n\n", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := indexNowTestClient(func(*http.Request) (int, string) { return http.StatusOK, test.body })
			err := requireIndexNowPublicKey(context.Background(), client, "https://locale.example/"+key+".txt", key)
			if (err == nil) != test.want {
				t.Fatalf("err = %v, want accepted=%v", err, test.want)
			}
		})
	}
}

func testIndexNowSitemap(origin string, count int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><urlset>`)
	for i := 0; i < count; i++ {
		if i == 0 {
			b.WriteString("<url><loc>" + origin + "</loc></url>")
			continue
		}
		fmt.Fprintf(&b, "<url><loc>%stour/test/%d</loc></url>", origin, i)
	}
	b.WriteString("</urlset>")
	return b.String()
}

func indexNowTestClient(handler func(*http.Request) (int, string)) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		status, body := handler(request)
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
