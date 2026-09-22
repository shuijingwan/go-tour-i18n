//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch, URL இன் body-ஐயும்
	// அந்தப் பக்கத்தில் காணப்படும் URL-களின் ஒரு துண்டத்தையும் திருப்பித் தருகிறது.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl, fetcher-ஐப் பயன்படுத்தி மீளுருவாக
// url-இல் தொடங்கும் பக்கங்களை அதிகபட்சம் depth ஆழம் வரை வலைவலம் செய்கிறது.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: URL-களை இணையாகப் பெறுங்கள்.
	// TODO: அதே URL ஐ இரண்டாவது முறையாகப் பெறாதீர்கள்.
	// இந்தச் செயலாக்கம் இவ்விரண்டையும் செய்யாது:
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

// fakeFetcher என்பது முன்கூட்டியே நிர்ணயிக்கப்பட்ட முடிவுகளைத் தரும் Fetcher ஆகும்.
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

// fetcher என்பது தரவால் நிரப்பப்பட்ட fakeFetcher ஆகும்.
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
