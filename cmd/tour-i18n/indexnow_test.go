package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapIndexNowSubmitsProbeThenRemainingSitemapURLs(t *testing.T) {
	key := "index-now-test-key"
	var submissions []indexNowSubmission
	client := indexNowTestClient(func(r *http.Request) (int, string) {
		switch r.URL.Path {
		case "/" + key + ".txt":
			return http.StatusOK, key
		case "/sitemap.xml":
			return http.StatusOK, testIndexNowSitemap("https://locale.example", 3)
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
	if result.SitemapURLs != 3 || result.SubmittedURLs != 2 {
		t.Fatalf("result = %+v", result)
	}
	if len(submissions) != 2 {
		t.Fatalf("submissions = %d, want 2", len(submissions))
	}
	if got := submissions[0].URLList; len(got) != 1 || got[0] != "https://locale.example" {
		t.Fatalf("probe URLs = %v", got)
	}
	if got := submissions[1].URLList; len(got) != 2 || containsURL(got, "https://locale.example") {
		t.Fatalf("bulk URLs = %d, probe included=%v", len(got), containsURL(got, "https://locale.example"))
	}
	for _, submission := range submissions {
		if submission.Host != profile.Hostname || submission.Key != key || submission.KeyLocation != "https://locale.example/"+key+".txt" {
			t.Fatalf("submission = %+v", submission)
		}
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
			return http.StatusOK, testIndexNowSitemap("https://locale.example", 3)
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
		return http.StatusOK, testIndexNowSitemap("https://locale.example", indexNowMaxURLs+1)
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

func testIndexNowSitemap(origin string, count int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><urlset>`)
	for i := 0; i < count; i++ {
		if i == 0 {
			b.WriteString("<url><loc>" + origin + "</loc></url>")
			continue
		}
		fmt.Fprintf(&b, "<url><loc>%s/tour/test/%d</loc></url>", origin, i)
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
