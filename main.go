package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strings"
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

// 用于翻译结果的图标显示
type icon struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// 用于单条结果
type Item struct {
	Subtitle     string `json:"subtitle"` // 固定的小字的标题
	Title        string `json:"title"`
	Arg          string `json:"arg"`
	Icon         icon   `json:"icon"`
	Valid        bool   `json:"valid"`
	Autocomplete string `json:"autocomplete"`
}

// 结果集
type Items struct {
	Items []Item `json:"items"`
}

// kw <directory-name>
// kw <url-name>... 直接展示所有标题/及url有该字符的，不管分类，默认忽略大小写
func main() {
	// TODO 从alfred的配置文件中读取webstack的config.yml的url

	if len(os.Args) > 3 {
		os.Exit(1)
	}
	fi := os.Args[1]

	// TODO 相比于alfred的书签管理，提供icon
	cate := getCategoriesFromConfig()
	allSites := extractAllSitesFromCategories(cate)
	cateNames := extractNameFromCategories(cate)
	names := matchFiAndCategoryNames(fi, cateNames)
	res := Sites{}

	if len(os.Args) == 3 {
		se := os.Args[2]
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

	items := generateItemsFromSites(res)
	itemsToJSON := ItemsToJSON(items)
	// 格式化打印json
	// var prettyJSON bytes.Buffer
	// error := json.Indent(&prettyJSON, []byte(itemsToJSON), "", "\t")
	// if error != nil {
	// 	log.Println("JSON parse error: ", error)
	// 	return
	// }
	//
	// log.Println("CSP Violation:", string(prettyJSON.Bytes()))

	fmt.Println(itemsToJSON)
}

// 用viper从config.yml中读取key为categories的数据
// 读取到的数据是一个数组，数组中的每个元素是一个map
// map中的key是name，value是sites
// sites是一个数组，数组中的每个元素是一个map
// map中的key是name，description，url，icon
func getCategoriesFromConfig() Categories {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	var webstack Webstack
	err = viper.Unmarshal(&webstack)
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
func generateItemsFromSites(Sites Sites) (items []Item) {
	for _, s := range Sites {
		item := Item{
			Arg:      s.URL,
			Title:    s.Name,
			Subtitle: s.Description,
			// TODO Icon:       s.Icon,
			Valid:        true,
			Autocomplete: s.Name,
		}
		items = append(items, item)
	}
	return
}

// 将[]Items转为json格式
func ItemsToJSON(items []Item) string {
	bytes, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
