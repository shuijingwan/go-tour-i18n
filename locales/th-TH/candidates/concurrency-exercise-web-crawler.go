//go:build OMIT

package main

import (
	"fmt"
)

type Fetcher interface {
	// Fetch คืนเนื้อหาของ URL และ
	// สไลซ์ของ URL ที่พบในหน้านั้น
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl ใช้ fetcher เพื่อเก็บข้อมูลจากหน้าแบบเรียกซ้ำ
// โดยเริ่มจาก url จนถึงความลึกสูงสุด depth
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: ดึง URL แบบขนาน
	// TODO: อย่าดึง URL เดิมซ้ำ
	// โค้ดนี้ยังไม่ได้ทำทั้งสองอย่าง
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

// fakeFetcher คือ Fetcher ที่คืนผลลัพธ์ที่เตรียมไว้ล่วงหน้า
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

// fetcher คือ fakeFetcher ที่มีข้อมูลเตรียมไว้แล้ว
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
