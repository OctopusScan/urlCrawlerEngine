package crawler

import (
	"github.com/Kumengda/easyChromedp/chrome"
	"github.com/Kumengda/easyChromedp/template"
	"time"
)

type ChromeCrawler struct {
	headers       map[string]string
	waitTime      time.Duration
	printLog      bool
	timeout       time.Duration
	chromeThreads int
	chrome        *chrome.Chrome
}

func (c *ChromeCrawler) GetCrawlThreads() int {
	return c.chromeThreads
}
func (c *ChromeCrawler) Close() {
	c.chrome.Close()
}
func (c *ChromeCrawler) DoFinally() {
	c.chrome.Close()
}

func (c *ChromeCrawler) SingleCrawl(task template.JsRes, allHref []template.JsRes) []template.JsRes {
	var chromeHeaders = make(map[string]interface{})
	for k, v := range c.headers {
		chromeHeaders[k] = v
	}
	templates, err := template.NewChromedpTemplates(
		c.printLog,
		c.timeout,
		c.waitTime,
		chromeHeaders,
		c.chrome,
	)
	if err != nil {
		return allHref
	}
	allReqHref, err := templates.GetWebsiteAllReq(task.Url)
	if err != nil {
		return nil
	}
	for _, v := range allReqHref {
		allHref = append(allHref, template.JsRes{
			Url:    v,
			Method: "GET",
			Param:  nil,
			IsForm: false,
		})

	}
	allJsHref, err := templates.GetWebsiteAllHrefByJs(task.Url)
	if err != nil {
		return nil
	}
	allHref = append(allHref, allJsHref...)
	return allHref
}
