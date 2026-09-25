//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch हे URL चे body आणि
	// त्या page वर सापडलेल्या URLs ची slice परत करते.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl, fetcher वापरून recursive पद्धतीने crawl करते
// ; url पासून सुरू होणारी pages जास्तीत जास्त depth पर्यंत शोधते.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: URLs समांतरपणे fetch करा.
	// TODO: तीच URL दोनदा fetch करू नका.
	// या implementation मध्ये यापैकी एकही गोष्ट होत नाही:
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

// fakeFetcher हा ठरवून दिलेले परिणाम परत करणारा Fetcher आहे.
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

// fetcher हा भरलेला fakeFetcher आहे.
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
