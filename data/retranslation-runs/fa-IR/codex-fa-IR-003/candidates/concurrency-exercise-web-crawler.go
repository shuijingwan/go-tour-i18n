//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch بدنهٔ URL و
	// اسلایسی از URLهای یافت‌شده در آن صفحه را برمی‌گرداند.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl با استفاده از fetcher، صفحه‌ها را به‌صورت بازگشتی
// از url آغاز می‌کند و تا حداکثر depth می‌خزد.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: URLها را به‌صورت موازی واکشی کنید.
	// TODO: یک URL را دو بار واکشی نکنید.
	// این پیاده‌سازی هیچ‌کدام را انجام نمی‌دهد:
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

// fakeFetcher یک Fetcher است که نتایج ازپیش‌آماده برمی‌گرداند.
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

// fetcher یک fakeFetcher پرشده است.
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
