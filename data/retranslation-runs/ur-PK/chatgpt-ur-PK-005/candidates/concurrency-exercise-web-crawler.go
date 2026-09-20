//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch، URL کا body واپس کرتا ہے اور
	// اس صفحے پر ملنے والے URLs کی ایک سلائس بھی۔
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl، fetcher استعمال کر کے بار بار اندرونی روابط کی پیروی کرتے ہوئے crawl کرتا ہے
// url سے شروع ہونے والے صفحات کو، زیادہ سے زیادہ depth تک۔
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: URLs کو parallel طور پر fetch کریں۔
	// TODO: ایک ہی URL کو دو بار fetch نہ کریں۔
	// یہ implementation ان دونوں کاموں میں سے کوئی نہیں کرتی:
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

// fakeFetcher ایسا Fetcher ہے جو پہلے سے تیار شدہ نتائج واپس کرتا ہے۔
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

// fetcher ایک پہلے سے بھرا ہوا fakeFetcher ہے۔
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
