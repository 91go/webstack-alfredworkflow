package fc

import (
	"net/http"

	query "github.com/PuerkitoBio/goquery"
)

func FetchHTML(url string) *query.Document {
	resp, err := http.Get(url)
	if err != nil {
		return &query.Document{}
	}
	defer resp.Body.Close()
	return DocQuery(resp)
}

// 请求goquery
func DocQuery(resp *http.Response) *query.Document {
	doc, err := query.NewDocumentFromReader(resp.Body)
	if err != nil {
		return &query.Document{}
	}

	return doc
}
