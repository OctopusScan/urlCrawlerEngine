package crawler

import (
	"github.com/B9O2/Inspector/useful"
	"github.com/Kumengda/easyChromedp/template"
	"github.com/Kumengda/httpProxyPool/httpProxyPool"
	"github.com/Kumengda/pageParser/parser"
	"github.com/OctopusScan/urlCrawlerEngine/http"
	"github.com/OctopusScan/urlCrawlerEngine/myClient"
	. "github.com/OctopusScan/urlCrawlerEngine/runtime"
	"golang.org/x/net/html/charset"
	"io"
	http2 "net/http"
	"net/url"
	"strings"
)

type NativeCrawler struct {
	headers      map[string]string
	proxyPool    *httpProxyPool.HttpProxyPool
	nativeClient *myClient.NativeClient
	threads      int
}

func (n *NativeCrawler) SingleCrawl(task template.JsRes, allHref []template.JsRes) []template.JsRes {
	var resp []byte
	var err error
	if n.proxyPool == nil {
		response, err, _ := http.Get(task.Url, n.nativeClient)
		if err != nil {
			MainInsp.Print(useful.ERROR, useful.Text(err.Error()))
			return nil
		}
		defer response.Body.Close()
		bodyReader, err := charset.NewReader(response.Body, response.Header.Get("Content-Type"))
		if err != nil {
			return nil
		}
		data, err := io.ReadAll(bodyReader)
		if err != nil {
			return nil
		}
		resp = data

	} else {
		req, _ := http2.NewRequest("GET", task.Url, nil)
		do, err := n.proxyPool.Do(req)
		if err != nil {
			MainInsp.Print(useful.ERROR, useful.Text(err.Error()))
			return nil
		}
		defer do.Body.Close()
		resp, _ = io.ReadAll(do.Body)
	}

	parse, err := url.Parse(task.Url)
	if err != nil {
		return nil
	}
	host := parse.Host
	scheme := parse.Scheme
	tagExtract := parser.NewTagExtract()
	tagExtract.InitTags(parser.DefaultTagRules)
	fromExtract := parser.NewFormExtract()
	tagRes := tagExtract.Extract(string(resp))
	formRes := fromExtract.Extract(string(resp))
	var allTagUrl []string
	for _, t := range tagRes {
		for _, v := range t.Attr {
			allTagUrl = append(allTagUrl, v)
		}
	}
	allTagUrl = cleanUrl(allTagUrl)
	allTagUrl = removeDuplicateStrings(allTagUrl)
	for _, v := range allTagUrl {
		allHref = append(allHref, template.JsRes{
			Url:    parseHrefData(v, scheme, host, task.Url, false),
			Method: "GET",
		})
	}
	for _, v := range formRes {
		var fromUrl string
		var newFormData []template.FormData
		isFileUpload := false
		for _, vv := range v.FormData {
			if vv.Name == "" || !checkInputType(vv.Type) {
				continue
			}
			if vv.Type == "file" {
				isFileUpload = true
			}
			newFormData = append(newFormData, template.FormData{
				Enctype: v.Enctype,
				Name:    vv.Name,
				Type:    vv.Type,
				Value:   vv.Value,
			})
		}
		if v.Action == "#" || v.Action == "/" || v.Action == "" {
			fromUrl = task.Url
		} else {
			fromUrl = parseHrefData(v.Action, scheme, host, task.Url, true)
		}
		allHref = append(allHref, template.JsRes{
			Url:          fromUrl,
			Method:       strings.ToUpper(v.Method),
			IsForm:       true,
			Param:        newFormData,
			IsFileUpload: isFileUpload,
		})
	}
	return allHref
}

func (n *NativeCrawler) GetCrawlThreads() int {
	return n.threads
}

func (n *NativeCrawler) DoFinally() {
}
