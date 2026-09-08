package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	indexNowEndpoint     = "https://api.indexnow.org/indexnow"
	indexNowMaxURLs      = 10000
	indexNowKeyMinLength = 8
	indexNowKeyMaxLength = 128
	indexNowGETAttempts  = 3
)

type indexNowIdentity struct {
	Locales []indexNowProfile `json:"locales"`
}

type indexNowProfile struct {
	Locale    string `json:"locale"`
	State     string `json:"production_state"`
	Hostname  string `json:"production_hostname"`
	PublicURL string `json:"production_public_url"`
}

var sitemapLocPattern = regexp.MustCompile(`<loc>([^<]+)</loc>`)
var indexNowKeyPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
var indexNowRetrySleep = time.Sleep

type indexNowSubmission struct {
	Host        string   `json:"host"`
	Key         string   `json:"key"`
	KeyLocation string   `json:"keyLocation"`
	URLList     []string `json:"urlList"`
}

func indexNowBootstrapCommand(root string, args []string) error {
	fs := flag.NewFlagSet("indexnow bootstrap", flag.ContinueOnError)
	locale := fs.String("locale", "", "live locale")
	keyFile := fs.String("key-file", "", "path to the private IndexNow key file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" || *keyFile == "" || fs.NArg() != 0 {
		return fmt.Errorf("usage: tour-i18n indexnow bootstrap --locale <locale> --key-file <path>")
	}
	if err := validateProductionIdentity(root, filepath.Join(root, "production", "identity.json")); err != nil {
		return err
	}
	profile, err := readIndexNowLiveProfile(filepath.Join(root, "production", "identity.json"), *locale)
	if err != nil {
		return err
	}
	key, err := readIndexNowKey(*keyFile)
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		// The key must be served from the exact production HTTPS root URL, not
		// merely reached after a redirect.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	result, err := bootstrapIndexNow(context.Background(), client, indexNowEndpoint, profile, key)
	if err != nil {
		return err
	}
	fmt.Printf("IndexNow bootstrap: PASS (locale=%s sitemap_urls=%d submitted_urls=%d)\n", profile.Locale, result.SitemapURLs, result.SubmittedURLs)
	return nil
}

func readIndexNowLiveProfile(path, locale string) (indexNowProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return indexNowProfile{}, err
	}
	var identity indexNowIdentity
	if err := json.Unmarshal(data, &identity); err != nil {
		return indexNowProfile{}, fmt.Errorf("malformed production identity")
	}
	matches := make([]indexNowProfile, 0, 1)
	for _, profile := range identity.Locales {
		if profile.Locale == locale {
			matches = append(matches, profile)
		}
	}
	if len(matches) != 1 {
		return indexNowProfile{}, fmt.Errorf("production identity must contain exactly one locale %s", locale)
	}
	profile := matches[0]
	if profile.State != "live" {
		return indexNowProfile{}, fmt.Errorf("indexnow bootstrap requires production_state=live, got %s", profile.State)
	}
	return profile, nil
}

func readIndexNowKey(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("read IndexNow key file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("IndexNow key file must be a regular non-symlink file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read IndexNow key file: %w", err)
	}
	key := string(data)
	if strings.HasSuffix(key, "\r\n") {
		key = strings.TrimSuffix(key, "\r\n")
	} else if strings.HasSuffix(key, "\n") {
		key = strings.TrimSuffix(key, "\n")
	}
	if key == "" || strings.ContainsAny(key, "\r\n") {
		return "", fmt.Errorf("IndexNow key file must contain one non-empty line")
	}
	if len(key) < indexNowKeyMinLength || len(key) > indexNowKeyMaxLength || !indexNowKeyPattern.MatchString(key) {
		return "", fmt.Errorf("IndexNow key must be %d..%d characters of [A-Za-z0-9-]", indexNowKeyMinLength, indexNowKeyMaxLength)
	}
	if filepath.Base(path) != key+".txt" {
		return "", fmt.Errorf("IndexNow key file name must be <key>.txt")
	}
	return key, nil
}

type indexNowBootstrapResult struct {
	SitemapURLs   int
	SubmittedURLs int
}

func bootstrapIndexNow(ctx context.Context, client *http.Client, endpoint string, profile indexNowProfile, key string) (indexNowBootstrapResult, error) {
	origin, err := parseIndexNowOrigin(profile)
	if err != nil {
		return indexNowBootstrapResult{}, err
	}
	keyLocation := origin + "/" + key + ".txt"
	if err := requireIndexNowPublicKey(ctx, client, keyLocation, key); err != nil {
		return indexNowBootstrapResult{}, err
	}
	urls, err := fetchIndexNowSitemap(ctx, client, origin+"/sitemap.xml", profile.Hostname, origin)
	if err != nil {
		return indexNowBootstrapResult{}, err
	}
	// Keep the identity's validated root URL byte-for-byte for the probe. The
	// trimmed origin is only for constructing sibling resource URLs.
	probe := profile.PublicURL
	if !containsURL(urls, probe) {
		return indexNowBootstrapResult{}, fmt.Errorf("formal sitemap does not contain fixed probe URL %s", probe)
	}
	probeStatus, err := submitIndexNow(ctx, client, endpoint, profile.Hostname, key, keyLocation, []string{probe})
	if err != nil {
		return indexNowBootstrapResult{}, err
	}
	if probeStatus == http.StatusAccepted {
		return indexNowBootstrapResult{}, fmt.Errorf("IndexNow probe is pending (HTTP 202); stop and retry after the key is available")
	}
	if probeStatus != http.StatusOK {
		return indexNowBootstrapResult{}, fmt.Errorf("IndexNow probe expected HTTP 200, got %d", probeStatus)
	}
	bulk := withoutURL(urls, probe)
	if len(bulk) == 0 {
		// The single sitemap URL was the successfully accepted probe; IndexNow
		// has no non-empty bulk payload to receive.
		return indexNowBootstrapResult{SitemapURLs: len(urls), SubmittedURLs: len(urls)}, nil
	}
	status, err := submitIndexNow(ctx, client, endpoint, profile.Hostname, key, keyLocation, bulk)
	if err != nil {
		return indexNowBootstrapResult{}, err
	}
	if status == http.StatusAccepted {
		return indexNowBootstrapResult{}, fmt.Errorf("IndexNow bulk submission is pending (HTTP 202); stop and retry later")
	}
	if status != http.StatusOK {
		return indexNowBootstrapResult{}, fmt.Errorf("IndexNow bulk submission expected HTTP 200, got %d", status)
	}
	return indexNowBootstrapResult{SitemapURLs: len(urls), SubmittedURLs: len(urls)}, nil
}

func parseIndexNowOrigin(profile indexNowProfile) (string, error) {
	u, err := url.Parse(profile.PublicURL)
	if err != nil || u.Scheme != "https" || u.Hostname() != profile.Hostname || u.Path != "/" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("production identity has invalid HTTPS root public URL for %s", profile.Locale)
	}
	return strings.TrimSuffix(profile.PublicURL, "/"), nil
}

func requireIndexNowPublicKey(ctx context.Context, client *http.Client, keyLocation, key string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, keyLocation, nil)
	if err != nil {
		return err
	}
	response, err := indexNowSafeGET(client, request)
	if err != nil {
		return fmt.Errorf("verify public IndexNow key: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK || !matchesIndexNowPublicKey(body, key) {
		return fmt.Errorf("public IndexNow root key verification failed (HTTP %d)", response.StatusCode)
	}
	return nil
}

// matchesIndexNowPublicKey permits the one line ending produced by printf,
// while preserving the key's exact bytes and rejecting any other whitespace.
func matchesIndexNowPublicKey(body []byte, key string) bool {
	value := string(body)
	if strings.HasSuffix(value, "\r\n") {
		value = strings.TrimSuffix(value, "\r\n")
	} else if strings.HasSuffix(value, "\n") {
		value = strings.TrimSuffix(value, "\n")
	}
	return value == key
}

func fetchIndexNowSitemap(ctx context.Context, client *http.Client, sitemapURL, hostname, origin string) ([]string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sitemapURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := indexNowSafeGET(client, request)
	if err != nil {
		return nil, fmt.Errorf("fetch formal sitemap: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("formal sitemap expected HTTP 200, got %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read formal sitemap: %w", err)
	}
	if !bytes.Contains(body, []byte("<urlset")) || !bytes.Contains(body, []byte("</urlset>")) {
		return nil, fmt.Errorf("formal sitemap is not an XML urlset")
	}
	matches := sitemapLocPattern.FindAllSubmatch(body, -1)
	if len(matches) == 0 || len(matches) > indexNowMaxURLs {
		return nil, fmt.Errorf("formal sitemap URL count = %d, want 1..%d", len(matches), indexNowMaxURLs)
	}
	seen := make(map[string]struct{}, len(matches))
	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		location := string(match[1])
		u, err := url.Parse(location)
		if err != nil || u.Scheme != "https" || u.Hostname() != hostname || u.RawQuery != "" || u.Fragment != "" || (location != origin && !strings.HasPrefix(location, origin+"/")) {
			return nil, fmt.Errorf("formal sitemap has invalid URL %q", location)
		}
		if _, duplicate := seen[location]; duplicate {
			return nil, fmt.Errorf("formal sitemap has duplicate URL %q", location)
		}
		seen[location] = struct{}{}
		urls = append(urls, location)
	}
	return urls, nil
}

// indexNowSafeGET retries only safe reads with the established three-attempt,
// 1s/2s bounded backoff. POST results can be unknown and are never retried.
func indexNowSafeGET(client *http.Client, request *http.Request) (*http.Response, error) {
	for attempt := 1; attempt <= indexNowGETAttempts; attempt++ {
		response, err := client.Do(request)
		if err == nil && !indexNowTransientStatus(response.StatusCode) {
			return response, nil
		}
		if err != nil && !indexNowTransientTransport(err) {
			return nil, err
		}
		if attempt == indexNowGETAttempts {
			if err != nil {
				return nil, err
			}
			return response, nil
		}
		if err == nil {
			response.Body.Close()
		}
		indexNowRetrySleep(time.Duration(attempt) * time.Second)
	}
	panic("unreachable")
}

func indexNowTransientStatus(status int) bool {
	return status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout || status == 522 || status == 525
}

func indexNowTransientTransport(err error) bool {
	var networkError net.Error
	return errors.As(err, &networkError) && (networkError.Timeout() || networkError.Temporary())
}

func submitIndexNow(ctx context.Context, client *http.Client, endpoint, host, key, keyLocation string, urls []string) (int, error) {
	payload, err := json.Marshal(indexNowSubmission{Host: host, Key: key, KeyLocation: keyLocation, URLList: urls})
	if err != nil {
		return 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("submit to IndexNow: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4097))
	return response.StatusCode, nil
}

func containsURL(urls []string, target string) bool {
	for _, value := range urls {
		if value == target {
			return true
		}
	}
	return false
}

func withoutURL(urls []string, target string) []string {
	result := make([]string, 0, len(urls)-1)
	for _, value := range urls {
		if value != target {
			result = append(result, value)
		}
	}
	return result
}
