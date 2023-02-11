package main

import (
	"os"
	"sync"
	"testing"
	"wsaw/fc"

	query "github.com/PuerkitoBio/goquery"
)

// BenchmarkGetCategoriesDataWg-8   	       1	1824024404 ns/op	   0.00 MB/s	31176160 B/op	  308578 allocs/op
// BenchmarkGetCategoriesData-8     	       1	9798359362 ns/op	   0.00 MB/s	 9239320 B/op	   43095 allocs/op

// BenchmarkGetCategoriesDataWg-8   	       1	1581641318 ns/op	   0.00 MB/s	35270648 B/op	  362104 allocs/op
// BenchmarkGetCategoriesData-8     	       1	9652091662 ns/op	   0.00 MB/s	 8509280 B/op	   32882 allocs/op

// BenchmarkGetCategoriesDataWg-8   	       1	1980351262 ns/op	   0.00 MB/s	32769896 B/op	  330040 allocs/op
// BenchmarkGetCategoriesData-8     	       1	10077891044 ns/op	   0.00 MB/s	 8472400 B/op	   32952 allocs/op

// BenchmarkGetCategoriesDataWg-8   	       4	 304744851 ns/op	   0.00 MB/s	 8889888 B/op	   35965 allocs/op
// BenchmarkGetCategoriesData-8     	       1	9516506947 ns/op	   0.00 MB/s	 8442184 B/op	   32940 allocs/op
func BenchmarkGetCategoriesDataWg(b *testing.B) {
	url := "https://ws.wrss.top/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 用b.SetBytes判断GC被触发频率
		b.SetBytes(111)
		// 输出信息会有B/op和allocs/op
		b.ReportAllocs()
		getCategoriesFromConfigURLWg(url)
		// 删除文件，路径为~/documents/pwgen/cache/categories和md5
		dir := "/Users/lhgtqb7bll/documents/pwgen/cache"
		err := deleteFiles(dir)
		if err != nil {
			print(err.Error())
		}
	}
}

func BenchmarkGetCategoriesData(b *testing.B) {
	url := "https://ws.wrss.top/"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 用b.SetBytes判断GC被触发频率
		b.SetBytes(111)
		// 输出信息会有B/op和allocs/op
		b.ReportAllocs()
		getCategoriesFromConfigURLNormal(url)
		// 删除文件，路径为~/documents/pwgen/cache/categories和md5
		dir := "/Users/lhgtqb7bll/documents/pwgen/cache"
		err := deleteFiles(dir)
		if err != nil {
			print(err.Error())
		}
	}
}

func deleteFiles(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(dir + "/" + name)
		if err != nil {
			return err
		}
	}
	return nil
}

func getCategoriesFromConfigURLWg(url string) (cate []Category) {
	fc.FetchHTML(url).Find(".row").Each(func(i int, s *query.Selection) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var sites []Site
			wg2 := sync.WaitGroup{}
			s.Find(".col-sm-3").Each(func(i int, se *query.Selection) {
				wg2.Add(1)
				go func() {
					defer wg2.Done()
					siteName := se.Find(".xe-comment a strong").Text()
					siteURL, _ := se.Find(".label-info").Attr("data-original-title")
					siteDes := se.Find(".xe-comment p").Text()
					iconURL := se.Find(".xe-user-img img").AttrOr("data-src", "")

					sites = append(sites, Site{
						Name:        siteName,
						URL:         siteURL,
						Description: siteDes,
						Icon:        getLocalIcon(iconURL, siteURL),
					})
				}()
			})
			wg2.Wait()
			name := s.Prev().Text()
			cate = append(cate, Category{
				Name:  name,
				Sites: sites,
			})
		}()
	})
	wg.Wait()
	return cate
}

func getCategoriesFromConfigURLNormal(url string) (cate []Category) {
	doc := fc.FetchHTML(url)

	doc.Find(".row").Each(func(i int, s *query.Selection) {
		var sites []Site
		s.Find(".col-sm-3").Each(func(i int, se *query.Selection) {
			siteName := se.Find(".xe-comment a strong").Text()
			siteURL, _ := se.Find(".label-info").Attr("data-original-title")
			siteDes := se.Find(".xe-comment p").Text()
			icon := se.Find(".xe-user-img img").AttrOr("data-src", "")

			sites = append(sites, Site{
				Name:        siteName,
				URL:         siteURL,
				Description: siteDes,
				Icon:        getLocalIcon(icon, siteURL),
			})
			name := s.Prev().Text()
			cate = append(cate, Category{
				Name:  name,
				Sites: sites,
			})
		})
	})
	return cate
}
