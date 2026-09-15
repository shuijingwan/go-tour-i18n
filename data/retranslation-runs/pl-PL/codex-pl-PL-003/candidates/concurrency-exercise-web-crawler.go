//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch zwraca treść URL oraz
	// wycinek adresów URL znalezionych na tej stronie.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl używa fetcher do rekurencyjnego przeszukiwania
// stron, zaczynając od url, do głębokości nie większej niż depth.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: Wywołuj Fetch równolegle dla adresów URL.
	// TODO: Nie pobieraj dwukrotnie tego samego URL.
	// Ta implementacja nie robi żadnej z tych rzeczy:
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

// fakeFetcher jest implementacją Fetcher zwracającą przygotowane wyniki.
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

// fetcher jest wypełnioną danymi wartością fakeFetcher.
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
