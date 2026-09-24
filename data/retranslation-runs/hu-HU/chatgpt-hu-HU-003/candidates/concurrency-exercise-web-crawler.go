//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// A Fetch visszaadja a(z) URL törzsét és
	// az oldalon talált URL-ek szeletét.
	Fetch(url string) (body string, urls []string, err error)
}

// A Crawl a fetcher segítségével rekurzívan bejárja
// az url címtől induló oldalakat legfeljebb depth mélységig.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: Az URL-ek lekérése párhuzamosan.
	// TODO: Ugyanazt a(z) URL címet ne kérd le kétszer.
	// Ez a megvalósítás egyiket sem teszi:
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

// A fakeFetcher előre megadott eredményeket visszaadó Fetcher.
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

// A fetcher egy feltöltött fakeFetcher.
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
