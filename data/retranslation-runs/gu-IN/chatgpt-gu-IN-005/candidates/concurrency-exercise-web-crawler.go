//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch, URL નો બોડી ભાગ અને
	// તે પાના પર મળેલા URL ની સ્લાઇસ પરત કરે છે.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl, fetcher નો ઉપયોગ કરીને
// url થી શરૂ થતાં પાનાંઓને depth ની મહત્તમ ઊંડાઈ સુધી પુનરાવર્તિત રીતે ક્રૉલ કરે છે.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: એકથી વધુ URL સમાંતર રીતે મેળવો.
	// TODO: એક જ URL ફરીથી મેળવશો નહીં.
	// આ અમલીકરણ આ બંનેમાંથી કંઈ કરતું નથી:
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

// fakeFetcher એ તૈયાર પરિણામો પરત કરતું Fetcher છે.
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

// fetcher એ ડેટાથી ભરેલું fakeFetcher છે.
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
