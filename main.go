package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"wsaw/fc"

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

const (
	CategoryKey = "categories"
	Md5Key      = "md5"
	EnvURL      = "url"
)

var (
	wf *aw.Workflow
	wg sync.WaitGroup
)

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
	if len(args) > 2 {
		return
	}
	defer func() {
		if err == nil {
			wf.SendFeedback()
			return
		}
	}()

	envURL, b := wf.Alfred.Env.Lookup(EnvURL)
	if !b {
		return
	}

	var res []Site

	cate := getCategoriesFromCacheOrConfig(envURL)
	allSites := extractAllSitesFromCategories(cate)
	cateNames := extractNameFromCategories(cate)

	switch len(args) {
	case 0:
		res = allSites
	case 2:
		se := args[1]
		fi := args[0]
		res = matchSeAndSites(fi, se, cate)
	case 3:
		fi := args[0]
		names := matchFiAndCategoryNames(fi, cateNames)
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
		err := wf.Cache.LoadOrStoreJSON(CategoryKey, 0*time.Minute, func() (interface{}, error) {
			return getCategoriesFromConfigURL(url), nil
		}, &older)
		if err != nil {
			panic(err)
		}
		return older
	}
	// 如果网页修改，则重新获取
	newer := getCategoriesFromConfigURL(url)
	err := wf.Cache.StoreJSON(CategoryKey, newer)
	if err != nil {
		panic(err)
	}
	return newer
}

// 直接从url中获取categories
func getCategoriesFromConfigURL(url string) (cate []Categories) {
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
			cate = append(cate, Categories{
				Name:  name,
				Sites: sites,
			})
		}()
	})
	wg.Wait()
	return cate
}

// 判断网页是否修改，通过MD5值判断
// true: 已修改
// false: 未修改
func determineContentIsModified(url string) bool {
	var oldMD5 []byte
	// 从缓存中读取旧的md5值（如果没有则重新获取）
	err := wf.Cache.LoadOrStoreJSON(Md5Key, 0*time.Minute, func() (interface{}, error) {
		return getMD5FromURL(url), nil
	}, &oldMD5)
	if err != nil {
		return false
	}
	// 新旧md5不同，说明网页修改，则重新获取
	newMD5 := getMD5FromURL(url)
	if !bytes.Equal(oldMD5, newMD5) {
		err := wf.Cache.Store(Md5Key, newMD5)
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

// 根据cate的名称，提取某个cate下的所有sites
func extractSitesFromCategory(cate string, categories []Categories) (sites []Site) {
	for _, v := range categories {
		if v.Name == cate {
			sites = v.Sites
		}
	}
	return
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
		// 注意使用aw.IconTypeImage，否则无法显示图片
		wf.NewItem(s.Name).Arg(s.URL).Subtitle(s.Description).
			Valid(true).Autocomplete(s.Name).Icon(&aw.Icon{Value: s.Icon, Type: aw.IconTypeImage})
	}
	return
}

// 与本地icon根据siteURL的hostname进行匹配，如果匹配到则直接使用本地path，如果未匹配到则下载icon到本地，再使用本地path
func getIconHostname(siteURL string) string {
	u, err := url.Parse(siteURL)
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(u.Hostname(), ".", "-")
}

// 下载icon到本地目标地址
// 本地icon的存储路径为：./icons
// 本地icon的命名规则为：hostname.png
func getLocalIcon(iconURL, siteURL string) string {
	iconName := getIconHostname(siteURL)
	filepath := wf.CacheDir() + "/icons-" + iconName + ".png"
	// 判断文件是否存在，如果存在则直接返回
	if _, err := os.Stat(filepath); err == nil {
		return filepath
	}
	// 如果不存在，则下载icon到本地
	resp, err := http.Get(iconURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	// 写入文件
	f, err := os.Create(filepath)
	if err != nil {
		return ""
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return ""
	}
	return filepath
}
