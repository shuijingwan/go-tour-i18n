//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch, URL-ന്റെ body-യും
	// ആ പേജിൽ കണ്ടെത്തിയ URL-കളുടെ ഒരു സ്ലൈസും തിരികെ നൽകുന്നു.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl, fetcher ഉപയോഗിച്ച്
// url മുതൽ depth എന്ന പരമാവധി ആഴം വരെ പേജുകൾ ആവർത്തിച്ച് ക്രോൾ ചെയ്യുന്നു.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: URL-കൾ സമാന്തരമായി ലഭ്യമാക്കുക.
	// TODO: ഒരേ URL രണ്ടുതവണ ലഭ്യമാക്കരുത്.
	// ഈ implementation ഇവയിൽ ഒന്നും ചെയ്യുന്നില്ല:
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

// fakeFetcher മുൻകൂട്ടി നിശ്ചയിച്ച ഫലങ്ങൾ തിരികെ നൽകുന്ന Fetcher ആണ്.
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

// fetcher മൂല്യങ്ങൾ നൽകിയ fakeFetcher ആണ്.
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
