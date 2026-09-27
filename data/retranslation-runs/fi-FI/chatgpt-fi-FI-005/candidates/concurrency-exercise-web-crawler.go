//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch palauttaa URL-osoitetta vastaavan verkkosivun sisällön ja
	// viipaleen tältä sivulta löytyvistä URL-osoitteista.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl käyttää fetcher-muuttujaa ja käy rekursiivisesti läpi
// sivuja url-osoitteesta alkaen enintään depth-muuttujan määrittämään syvyyteen.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: Hae URL-osoitteet rinnakkain.
	// TODO: Älä hae samaa URL-osoitetta kahdesti.
	// Tämä toteutus ei tee kumpaakaan:
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

// fakeFetcher on Fetcher, joka palauttaa ennalta määritetyt tulokset.
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

// fetcher on valmiiksi täytetty fakeFetcher.
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
