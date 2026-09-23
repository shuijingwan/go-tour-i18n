//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Ibinabalik ng Fetch ang nilalaman ng URL at
	// isang slice ng mga URL na natagpuan sa pahinang iyon.
	Fetch(url string) (body string, urls []string, err error)
}

// Ginagamit ng Crawl ang fetcher upang paulit-ulit na i-crawl
// ang mga pahina simula sa url, hanggang sa pinakamataas na depth.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: Kunin ang mga URL nang magkakasabay.
	// TODO: Huwag kunin ang parehong URL nang dalawang beses.
	// Wala sa dalawang ito ang ginagawa ng implementasyong ito:
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

// Ang fakeFetcher ay isang Fetcher na nagbabalik ng mga nakahandang resulta.
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

// Ang fetcher ay isang fakeFetcher na may nakatalagang data.
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
