//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch מחזירה את גוף ה־URL;
	// היא מחזירה גם פרוסה של כתובות URL שנמצאו בדף הזה.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl משתמשת ב־fetcher כדי לסרוק באופן רקורסיבי
// דפים החל מ־url, עד לעומק המרבי depth.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: אחזרו כתובות URL במקביל.
	// TODO: אל תאחזרו את אותה כתובת URL פעמיים.
	// המימוש הזה אינו עושה אף אחד מהדברים האלה:
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

// fakeFetcher הוא Fetcher שמחזיר תוצאות קבועות מראש.
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

// fetcher הוא fakeFetcher מאוכלס.
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
