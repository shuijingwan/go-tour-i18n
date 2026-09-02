//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch 함수는 URL 주소의 본문과
	// 그 페이지에서 찾은 URL 슬라이스를 반환합니다.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl 함수는 fetcher 값을 사용하여
// url 주소에서 시작해 depth 값으로 지정된 최대 깊이까지 페이지를 재귀적으로 크롤링합니다.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: URL 가져오기를 병렬로 수행하기.
	// TODO: URL 값이 같으면 두 번 가져오지 않기.
	// 이 구현은 두 작업 모두 수행하지 않습니다.
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

// fakeFetcher 타입은 미리 준비된 결과를 반환하는 Fetcher 구현체입니다.
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

// fetcher 변수는 데이터가 채워진 fakeFetcher 값입니다.
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
