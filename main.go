package main

import (
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

type Category struct {
	Name  string `json:"cateName"`
	Sites []Site `json:"sites"`
}

type Categories []Category

type Site struct {
	Name        string
	Description string `json:"description,omitempty"`
	URL         string
	Icon        string
}

const (
	CategoryKey = "categories.json"
	EnvURL      = "url"
	EnvExpire   = "expire"
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

	siteURL := wf.Config.GetString(EnvURL, "https://ws.wrss.top/")
	expire := wf.Config.GetInt(EnvExpire, 12)

	res := make([]Site, 0)
	cates := make(Categories, 0)
	cates.getCategoriesFromCacheOrConfig(siteURL, expire)

	switch len(args) {
	case 0:
		res = cates.extractAllSitesFromCategories()
	case 1:
		fi := args[0]
		names := cates.matchFiAndCategoryNames(fi)
		// 如果names不为空，则说明匹配到了分类
		if len(names) > 0 {
			res = cates.extractSitesFromCategory(fi)
		} else {
			// 如果names为空，则说明没有匹配到分类，需要匹配url
			res = cates.matchFiAndSites(fi)
		}
	case 2:
		se := args[1]
		fi := args[0]
		res = cates.matchSeAndSites(fi, se)
	}

	generateItemsFromSites(res)
	wf.SendFeedback()
}

// 使用LoadOrStoreJSON直接从缓存中读取数据
func (categories *Categories) getCategoriesFromCacheOrConfig(url string, expire int) {
	// 默认直接从缓存中读取
	err := wf.Cache.LoadOrStoreJSON(CategoryKey, time.Duration(expire)*time.Hour, func() (interface{}, error) {
		return categories.getCategoriesFromConfigURL(url), nil
	}, &categories)
	if err != nil {
		panic(err)
	}
	// 判断网页是否修改，如果未修改，则直接读取
	// isModified := determineContentIsModified(url)
	// if !isModified {
	// 	err := wf.Cache.LoadOrStoreJSON(CategoryKey, 0*time.Minute, func() (interface{}, error) {
	// 		return categories.getCategoriesFromConfigURL(url), nil
	// 	}, &categories)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	return
	// }
	// // 如果网页修改，则重新获取
	// newer := categories.getCategoriesFromConfigURL(url)
	// err := wf.Cache.StoreJSON(CategoryKey, newer)
	// if err != nil {
	// 	panic(err)
	// }
	// categories = &newer
}

// 直接从url中获取categories
func (categories *Categories) getCategoriesFromConfigURL(url string) Categories {
	fc.FetchHTML(url).Find(".row").Each(func(i int, s *query.Selection) {
		var sites []Site
		s.Find(".col-sm-3").Each(func(i int, se *query.Selection) {
			siteName := se.Find(".xe-comment a strong").Text()
			siteURL, _ := se.Find(".label-info").Attr("data-original-title")
			siteDes := se.Find(".xe-comment p").Text()
			iconURL := se.Find(".xe-user-img img").AttrOr("data-src", "")

			wg.Add(1)
			go saveIcon(iconURL, siteURL)

			sites = append(sites, Site{
				Name:        siteName,
				URL:         siteURL,
				Description: siteDes,
				Icon:        getLocalIconPath(siteURL),
			})
		})
		name := s.Prev().Text()
		*categories = append(*categories, Category{
			Name:  name,
			Sites: sites,
		})
		wg.Wait()
	})
	return *categories
}

// 提取categories中的name
func (categories *Categories) extractNameFromCategories() []string {
	var categoryNames []string
	for _, v := range *categories {
		categoryNames = append(categoryNames, v.Name)
	}
	return categoryNames
}

// 提取categories中的所有sites
func (categories *Categories) extractAllSitesFromCategories() (sites []Site) {
	for _, v := range *categories {
		sites = append(sites, v.Sites...)
	}
	return sites
}

// 根据cate的名称，提取某个cate下的所有sites
func (categories *Categories) extractSitesFromCategory(cate string) (sites []Site) {
	for _, v := range *categories {
		if v.Name == cate {
			sites = v.Sites
		}
	}
	return
}

// 优先匹配cate，如果匹配到就直接展示该cate下的所有site
func (categories *Categories) matchFiAndCategoryNames(fi string) []string {
	var matchFi []string
	for _, v := range categories.extractNameFromCategories() {
		if v == fi {
			matchFi = append(matchFi, v)
		}
	}
	return matchFi
}

// 如果匹配不到，则全局搜索所有的site
// 将fi和Sites中的Site中的Name和Url进行匹配，如果能匹配到，组装为[]Sites，，最终将数据转为json格式，如果没有则为空json
func (categories *Categories) matchFiAndSites(fi string) (sites []Site) {
	for _, s := range categories.extractAllSitesFromCategories() {
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(fi)) || strings.Contains(strings.ToLower(s.URL), strings.ToLower(fi)) {
			sites = append(sites, s)
		}
	}
	return
}

// kw <directory-name> <url-name>... 首字母搜索，只在该分类下搜索
// 先对fi与Categories的Name进行匹配，然后对其下的Sites的name和URL与se进行匹配
func (categories *Categories) matchSeAndSites(fi, se string) (sites []Site) {
	for _, category := range *categories {
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

func saveIcon(iconURL, siteURL string) bool {
	defer wg.Done()
	filepath := getLocalIconPath(siteURL)
	if exist(filepath) {
		return true
	}
	// 如果不存在，则下载icon到本地
	resp, err := http.Get(iconURL)
	if err != nil {
		return false
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)
	// 写入文件
	f, err := os.Create(filepath)
	if err != nil {
		return false
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)
	_, err = io.Copy(f, resp.Body)
	return err == nil
}

// 下载icon到本地目标地址
// 本地icon的存储路径为：./icons
// 本地icon的命名规则为：hostname.png
func getLocalIconPath(siteURL string) string {
	iconName := getIconHostname(siteURL)
	filepath := wf.CacheDir() + "/icons-" + iconName + ".png"

	return filepath
}

func exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

// 从url中获取MD5值
//func getMD5FromURL(url string) []byte {
//	resp, err := http.Get(url)
//	if err != nil {
//		panic(err)
//	}
//	defer func(Body io.ReadCloser) {
//		err := Body.Close()
//		if err != nil {
//
//		}
//	}(resp.Body)
//	body, err := io.ReadAll(resp.Body)
//	if err != nil {
//		panic(err)
//	}
//	return []byte(fmt.Sprintf("%x", md5.Sum(body)))
//}

// 判断网页是否修改，通过MD5值判断
// true: 已修改
// false: 未修改
//func determineContentIsModified(url string) bool {
//	var oldMD5 []byte
//	// 从缓存中读取旧的md5值（如果没有则重新获取）
//	err := wf.Cache.LoadOrStoreJSON(Md5Key, 0*time.Minute, func() (interface{}, error) {
//		return getMD5FromURL(url), nil
//	}, &oldMD5)
//	if err != nil {
//		return false
//	}
//	// 新旧md5不同，说明网页修改，则重新获取
//	newMD5 := getMD5FromURL(url)
//	if !bytes.Equal(oldMD5, newMD5) {
//		err := wf.Cache.Store(Md5Key, newMD5)
//		if err != nil {
//			panic(err)
//		}
//		return true
//	}
//
//	return false
//}
