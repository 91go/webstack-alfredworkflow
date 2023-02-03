
# wsaw

wsaw = Open Webstack's website By Alfred's Workflow


## Installation

- 在 [Releases](https://github.com/91go/wsaw/releases) 页面下载workflow文件
- 在该workflow的环境变量中添加webstack's的配置文件 (e.g. `url`->`https://raw.githubusercontent.com/hapihacking/webstack/gh-pages/config.yml`)

    
## Usage/Examples




## Features

相比于alfred的内置"书签搜索功能"，wsaw的优势在于：

- ~~添加搜索到的网页的icon~~ 
- *可以直接根据关键字提取webstack中收藏的网页*. 使用webstack作为数据源，更容易浏览、追踪和管理，而workflow只是作为一个高效搜索的工具（不需要打开网站，再找到对应分类，再点击对应网页，只需要关键字就可以直接跳转，目的性和效率强很多）


## FAQ

#### 报错"Failed to connect to raw.githubusercontent.com port 443: Operation timed out"

首先手动访问该`config.yml`页面，以确定是否能够访问。如果不能访问，请检查该workflow的环境变量中的`url`是否正确。如果能够访问，请为该workflow或者整个alfred服务配置代理，以保证该workflow是否能够访问该`config.yml`页面。


