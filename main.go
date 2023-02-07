package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	query "github.com/PuerkitoBio/goquery"
	aw "github.com/deanishe/awgo"
)

type Categories struct {
	Name  string `json:"cateName"`
	Sites []Site `json:"sites"`
}

type Site struct {
	Name        string
	Description string `json:"description,omitempty"`
	URL         string
	Icon        string
}

// Workflow is the main API
var wf *aw.Workflow

func init() {
	wf = aw.New()
}

func main() {
	wf.Run(run)
}

// kw <directory-name>
// kw <url-name>... 直接展示所有标题/及url有该字符的，不管分类，默认忽略大小写
func run() {
	var err error
	args := wf.Args()
	if len(args) == 0 || len(args) > 2 {
		return
	}
	defer func() {
		if err == nil {
			wf.SendFeedback()
			return
		}
	}()

	fi := args[0]

	url, b := wf.Alfred.Env.Lookup("url")
	if !b {
		return
	}

	cate := getCategoriesFromCacheOrConfig(url)
	allSites := extractAllSitesFromCategories(cate)
	cateNames := extractNameFromCategories(cate)
	names := matchFiAndCategoryNames(fi, cateNames)
	var res []Site

	if len(args) == 2 {
		se := args[1]
		log.Println("se: ", se)
		res = matchSeAndSites(fi, se, cate)
	} else {
		// 如果names不为空，则说明匹配到了分类
		if len(names) > 0 {
			res = extractSitesFromCategory(fi, cate)
		} else {
			// 如果names为空，则说明没有匹配到分类，需要匹配url
			res = matchFiAndSites(fi, allSites)
		}
	}
	generateItemsFromSites(res)
	wf.SendFeedback()
}

// 使用LoadOrStoreJSON直接从缓存中读取数据
func getCategoriesFromCacheOrConfig(url string) []Categories {
	var older []Categories

	// 判断网页是否修改，如果未修改，则直接读取
	if !determineContentIsModified(url) {
		err := wf.Cache.LoadJSON("categories", &older)
		if err != nil {
			panic(err)
		}
		return older
	}
	// 如果网页修改，则重新获取
	newer := getCategoriesFromConfigURL(url)
	err := wf.Cache.StoreJSON("categories", newer)
	if err != nil {
		panic(err)
	}
	return newer
}

// 直接从url中获取categories
func getCategoriesFromConfigURL(url string) (cate []Categories) {
	doc := FetchHTML(url)

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
				Icon:        icon,
			})
		})
		name := s.Prev().Text()
		cate = append(cate, Categories{
			Name:  name,
			Sites: sites,
		})
	})
	return cate
}

// 判断网页是否修改，通过MD5值判断
// true: 已修改
// false: 未修改
func determineContentIsModified(url string) bool {
	var oldMD5 []byte
	// 从缓存中读取旧的md5值（如果没有则重新获取）
	err := wf.Cache.LoadOrStoreJSON("md5", 0*time.Minute, func() (interface{}, error) {
		return getMD5FromURL(url), nil
	}, &oldMD5)
	if err != nil {
		return false
	}
	// 新旧md5不同，说明网页修改，则重新获取
	newMD5 := getMD5FromURL(url)
	if !bytes.Equal(oldMD5, newMD5) {
		err := wf.Cache.Store("md5", newMD5)
		if err != nil {
			panic(err)
		}
		return true
	}

	return false
}

// 从url中获取MD5值
func getMD5FromURL(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return []byte(fmt.Sprintf("%x", md5.Sum(body)))
}

// 提取categories中的name
func extractNameFromCategories(categories []Categories) []string {
	var categoryNames []string
	for _, v := range categories {
		categoryNames = append(categoryNames, v.Name)
	}
	return categoryNames
}

// 提取categories中的所有sites
func extractAllSitesFromCategories(categories []Categories) (sites []Site) {
	for _, v := range categories {
		sites = append(sites, v.Sites...)
	}
	return sites
}

// 优先匹配cate，如果匹配到就直接展示该cate下的所有site
func matchFiAndCategoryNames(fi string, categoryNames []string) []string {
	var matchFi []string
	for _, v := range categoryNames {
		if v == fi {
			matchFi = append(matchFi, v)
		}
	}
	return matchFi
}

// 根据cate的名称，提取某个cate下的所有sites
func extractSitesFromCategory(cate string, categories []Categories) (sites []Site) {
	for _, v := range categories {
		if v.Name == cate {
			sites = v.Sites
		}
	}
	return
}

// 如果匹配不到，则全局搜索所有的site
// 将fi和Sites中的Site中的Name和Url进行匹配，如果能匹配到，组装为[]Sites，，最终将数据转为json格式，如果没有则为空json
func matchFiAndSites(fi string, allSites []Site) (sites []Site) {
	for _, s := range allSites {
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(fi)) || strings.Contains(strings.ToLower(s.URL), strings.ToLower(fi)) {
			sites = append(sites, s)
		}
	}
	return
}

// kw <directory-name> <url-name>... 首字母搜索，只在该分类下搜索
// 先对fi与Categories的Name进行匹配，然后对其下的Sites的name和URL与se进行匹配
func matchSeAndSites(fi, se string, categories []Categories) (sites []Site) {
	for _, category := range categories {
		if category.Name == fi {
			for _, s := range category.Sites {
				if strings.Contains(strings.ToLower(s.Name), strings.ToLower(se)) || strings.Contains(strings.ToLower(s.URL), strings.ToLower(se)) {
					sites = append(sites, s)
				}
			}
		}
	}

	return sites
}

// 根据sitesFromCategory生成items，Site和Item相对应，其中Item的Arg为Site的URL，Title为Site的Name，Subtitle为Site的Description
func generateItemsFromSites(sites []Site) (items []aw.Item) {
	for _, s := range sites {
		wf.NewItem(s.Name).Arg(s.URL).Subtitle(s.Description).
			Valid(true).Autocomplete(s.Name).Icon(&aw.Icon{Value: s.Icon, Type: aw.IconTypeFileIcon})
	}
	return
}

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
