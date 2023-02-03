package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	aw "github.com/deanishe/awgo"
	"gopkg.in/yaml.v3"
)

type Webstack struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Favicon     string `yaml:"favicon"`
	URL         string `yaml:"url"`
	Github      string `yaml:"github"`
	Footer      string `yaml:"footer"`
	Template    string `yaml:"template"`
	Content     struct {
		Categories `yaml:"categories"`
	} `yaml:"content"`
}

type Categories []struct {
	Name  string `yaml:"name"`
	Sites `yaml:"sites"`
}

type Sites []struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	URL         string `yaml:"url"`
	Icon        string `yaml:"icon,omitempty"`
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
	// log.Println("args: ", args, len(args))
	// log.Println("os.Args: ", os.Args, len(os.Args))
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
	// log.Println("fi: ", fi)

	wsURL, b := wf.Alfred.Env.Lookup("ws_url")
	if !b {
		return
	}

	// cate := getCategoriesFromConfig()
	cate := getCategoriesFromCacheOrConfig(wsURL)
	allSites := extractAllSitesFromCategories(cate)
	cateNames := extractNameFromCategories(cate)
	names := matchFiAndCategoryNames(fi, cateNames)
	var res Sites

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

// 用viper从config.yml中读取key为categories的数据
// 读取到的数据是一个数组，数组中的每个元素是一个map
// map中的key是name，value是sites
// sites是一个数组，数组中的每个元素是一个map
// map中的key是name，description，url，icon
// func getCategoriesFromConfig() Categories {
// 	viper.SetConfigName("config")
// 	viper.SetConfigType("yml")
// 	viper.AddConfigPath(".")
// 	err := viper.ReadInConfig()
// 	if err != nil {
// 		panic(err)
// 	}
// 	var webstack Webstack
// 	err = viper.Unmarshal(&webstack)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return webstack.Content.Categories
// }

// 使用LoadOrStoreJSON直接从缓存中读取数据
func getCategoriesFromCacheOrConfig(wsURL string) Categories {
	var cate Categories

	err := wf.Cache.LoadOrStoreJSON("categories", 20*time.Minute, func() (interface{}, error) {
		return getCategoriesFromConfigURL(wsURL), nil
	}, &cate)
	if err != nil {
		panic(err)
	}
	return cate
}

// 直接从url中获取categories
func getCategoriesFromConfigURL(wsURL string) Categories {
	resp, err := http.Get(wsURL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var webstack Webstack
	err = yaml.NewDecoder(resp.Body).Decode(&webstack)
	if err != nil {
		panic(err)
	}
	return webstack.Content.Categories
}

// 提取categories中的name
func extractNameFromCategories(categories Categories) []string {
	var categoryNames []string
	for _, v := range categories {
		categoryNames = append(categoryNames, v.Name)
	}
	return categoryNames
}

// 提取categories中的所有sites
func extractAllSitesFromCategories(categories Categories) []Sites {
	var sites []Sites
	for _, v := range categories {
		sites = append(sites, v.Sites)
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
func extractSitesFromCategory(cate string, categories Categories) (sites Sites) {
	for _, v := range categories {
		if v.Name == cate {
			sites = v.Sites
		}
	}
	return
}

// 如果匹配不到，则全局搜索所有的site
// 将fi和Sites中的Site中的Name和Url进行匹配，如果能匹配到，组装为[]Sites，，最终将数据转为json格式，如果没有则为空json
func matchFiAndSites(fi string, allSites []Sites) (sites Sites) {
	for _, v := range allSites {
		for _, s := range v {
			if strings.Contains(strings.ToLower(s.Name), strings.ToLower(fi)) || strings.Contains(strings.ToLower(s.URL), strings.ToLower(fi)) {
				sites = append(sites, s)
			}
		}
	}
	return
}

// kw <directory-name> <url-name>... 首字母搜索，只在该分类下搜索
// 先对fi与Categories的Name进行匹配，然后对其下的Sites的name和URL与se进行匹配
func matchSeAndSites(fi, se string, categories Categories) Sites {
	var sites Sites
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
func generateItemsFromSites(Sites Sites) (items []aw.Item) {
	for _, s := range Sites {
		wf.NewItem(s.Name).Arg(s.URL).Subtitle(s.Description).
			Valid(true).Autocomplete(s.Name)
	}
	return
}

// 从url中提取domain
// func extractDomainFromURL(siteURL string) string {
// 	u, err := url.Parse(siteURL)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return u.Host
// }

// 从url中提取domain，判断该网站根目录是否有favicon.ico，如果有则返回favicon.ico的url，如果没有则返回customIcon，作为icon的path
// icon的type为fileicon
// func getIconFromURL(url, customIcon string) string {
// 	domain := extractDomainFromURL(url)
// 	iconURL := fmt.Sprintf("http://%s/favicon.ico", domain)
// 	resp, err := http.Get(iconURL)
// 	if err != nil {
// 		panic(err)
// 	}
// 	if resp.StatusCode == 200 {
// 		return iconURL
// 	}
// 	return customIcon
// }
