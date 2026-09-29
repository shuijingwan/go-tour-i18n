//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch grąžina URL turinį ir
	// tame puslapyje rastų URL pjūvį.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl naudoja fetcher, kad rekursyviai nuskaitytų
// puslapius, pradėdamas nuo url; didžiausią nuskaitymo gylį nustato parametras depth.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: Gauti URL lygiagrečiai.
	// TODO: Negauti to paties URL du kartus.
	// Ši realizacija neatlieka nė vieno iš šių dalykų:
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

// fakeFetcher yra Fetcher, grąžinantis iš anksto paruoštus rezultatus.
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

// fetcher yra duomenimis užpildytas fakeFetcher.
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
