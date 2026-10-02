//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch የ URL ይዘትን ይመልሳል፣ እንዲሁም
	// በዚያ ገጽ ላይ የተገኙ URL-ዎችን የያዘ ቁራጭ።
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl ተግባር fetcher በመጠቀም ራሱን በተደጋጋሚ በመጥራት
// ከ url ጀምሮ ገጾችን እስከ depth ከፍተኛ ጥልቀት ድረስ ይጎበኛል።
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: Fetch ጥሪዎችን ለ URL-ዎች በትይዩነት አከናውን።
	// TODO: ተመሳሳዩን URL ሁለት ጊዜ አታምጣ።
	// ይህ ትግበራ ከሁለቱም አንዱንም አያደርግም፦
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

// fakeFetcher ቀድሞ የተዘጋጁ ውጤቶችን የሚመልስ Fetcher ነው።
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

// fetcher በውሂብ የተሞላ fakeFetcher ነው።
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
