//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch, URL ਦੀ ਬਾਡੀ ਅਤੇ
	// ਉਸ ਪੰਨੇ ਉੱਤੇ ਮਿਲੇ URLs ਦੀ ਸਲਾਈਸ ਵਾਪਸ ਕਰਦਾ ਹੈ।
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl, fetcher ਵਰਤ ਕੇ url ਤੋਂ ਸ਼ੁਰੂ ਹੋਣ ਵਾਲੇ ਪੰਨਿਆਂ ਨੂੰ
// ਵੱਧ ਤੋਂ ਵੱਧ depth ਤੱਕ ਰਿਕਰਸਿਵ ਢੰਗ ਨਾਲ ਕ੍ਰਾਲ ਕਰਦਾ ਹੈ।
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: URLs ਨੂੰ ਸਮਾਂਤਰ ਪ੍ਰਾਪਤ ਕਰੋ।
	// TODO: ਇੱਕੋ URL ਨੂੰ ਦੋ ਵਾਰ ਪ੍ਰਾਪਤ ਨਾ ਕਰੋ।
	// ਇਹ ਇੰਪਲੀਮੈਂਟੇਸ਼ਨ ਦੋਵਾਂ ਵਿੱਚੋਂ ਕੁਝ ਵੀ ਨਹੀਂ ਕਰਦੀ:
	if depth <= 0 {
		return
	}
	body, urls, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("found: %s %q\n", url, body)
	for _, u := range urls {
		Crawl(u, depth-1, fetcher)
	}
	return
}

func main() {
	Crawl("https://golang.org/", 4, fetcher)
}

// fakeFetcher ਇੱਕ Fetcher ਹੈ ਜੋ ਪਹਿਲਾਂ ਤੋਂ ਤਿਆਰ ਨਤੀਜੇ ਵਾਪਸ ਕਰਦਾ ਹੈ।
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher ਇੱਕ ਮੁੱਲਾਂ ਨਾਲ ਭਰਿਆ fakeFetcher ਹੈ।
var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}
