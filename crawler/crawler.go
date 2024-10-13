package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	inspect "github.com/B9O2/Inspector"
	"github.com/B9O2/Inspector/useful"
	"github.com/Kumengda/easyChromedp/chrome"
	"github.com/Kumengda/easyChromedp/template"
	"github.com/Kumengda/httpProxyPool/httpProxyPool"
	"github.com/OctopusScan/httpServiceFingerScanEngine"
	"github.com/OctopusScan/urlCrawlerEngine/fingerDetect"
	"github.com/OctopusScan/urlCrawlerEngine/myClient"
	"github.com/OctopusScan/urlCrawlerEngine/res"
	. "github.com/OctopusScan/urlCrawlerEngine/runtime"
	"github.com/OctopusScan/urlCrawlerEngine/webDirBlast"
	"github.com/chromedp/chromedp"
	"strings"
	"time"
)

type Crawler struct {
	BaseCrawler
	crawl            Crawl
	webDirBlastPaths []webDirBlast.WebDirBlastPath
}
type Options struct {
	insp               *inspect.Inspector
	depth              int
	noCrawlerFilter    []string
	maxResultNum       int
	clientOptions      myClient.NativeClientOptions
	webDirBlastThreads uint
	usingDirScanRes    bool
	dirScanTier        int
}

func NewCrawler(target string, crawlerOption Options) (*Crawler, error) {
	if crawlerOption.insp != nil {
		MainInsp = crawlerOption.insp
	}
	host, err := getHost(target)
	if err != nil {
		return nil, err
	}
	nativeClient, err := myClient.NewNativeClient(
		crawlerOption.clientOptions.ProxyUrl,
		crawlerOption.clientOptions.MaxRedirect,
		crawlerOption.clientOptions.Timeout,
		crawlerOption.clientOptions.ConnTimeout,
		crawlerOption.clientOptions.RateLimit,
		crawlerOption.clientOptions.MaxRespSize,
		crawlerOption.clientOptions.Headers,
		crawlerOption.clientOptions.MaxRetryCount)
	if err != nil {
		return nil, err
	}
	return &Crawler{BaseCrawler: BaseCrawler{
		timeout:            crawlerOption.clientOptions.Timeout,
		usingDirScanRes:    crawlerOption.usingDirScanRes,
		webDirBlastThreads: crawlerOption.webDirBlastThreads,
		target:             target,
		depth:              crawlerOption.depth,
		host:               host,
		filter:             crawlerOption.noCrawlerFilter,
		maxResultNum:       crawlerOption.maxResultNum,
		middlewareFunc:     nil,
		nativeClient:       nativeClient,
		dirScanTier:        crawlerOption.dirScanTier,
	},
	}, nil
}
func NewCrawlerOptions(insp *inspect.Inspector, depth int, noCrawlerFilter []string, maxResultNum int, clientOptions myClient.NativeClientOptions, webDirBlastThreads uint, usingDirScanRes bool, dirScanTier int) Options {
	return Options{
		insp:               insp,
		depth:              depth,
		noCrawlerFilter:    noCrawlerFilter,
		maxResultNum:       maxResultNum,
		clientOptions:      clientOptions,
		webDirBlastThreads: webDirBlastThreads,
		usingDirScanRes:    usingDirScanRes,
		dirScanTier:        dirScanTier,
	}
}
func (c *Crawler) LoadWebDirBlastPath(blastPaths [][]byte) {
	var webDirBlastPaths []webDirBlast.WebDirBlastPath
	if blastPaths == nil {
		c.webDirBlastPaths = webDirBlastPaths
		return
	}
	for _, v := range blastPaths {
		var oneWebDirBlastPath webDirBlast.WebDirBlastPath
		err := json.Unmarshal(v, &oneWebDirBlastPath)
		if err != nil {
			MainInsp.Print(useful.ERROR, useful.Text(fmt.Sprintf("Load WebDirBlastPath error:%s", err.Error())))
			continue
		}
		webDirBlastPaths = append(webDirBlastPaths, oneWebDirBlastPath)
	}
	c.webDirBlastPaths = webDirBlastPaths
}
func (c *Crawler) SetCrawler(crawler Crawl) {
	c.crawl = crawler
}
func (c *Crawler) doBlast(threads uint) []string {
	webDirBlaster := webDirBlast.NewWebDirBlast(c.target, c.nativeClient, threads, c.dirScanTier)
	blast, err := webDirBlaster.DoBlast(c.webDirBlastPaths)
	if err != nil {
		return nil
	}
	return blast

}
func (c *Crawler) Crawl() res.DirResult {
	blastRes := c.doBlast(c.webDirBlastThreads)
	var dirRes res.DirResult
	dirRes.Target = c.target
	var startTargets []template.JsRes
	startTargets = append(startTargets, template.JsRes{
		Url:    c.target,
		Method: "GET",
	})
	if c.usingDirScanRes {
		if blastRes != nil {
			for _, v := range blastRes {
				startTargets = append(startTargets, template.JsRes{
					Url:    v,
					Method: "GET",
				})
			}
		}
	}
	crawlRes := c.crawAllUrl(startTargets, c.crawl, context.Background())
	for _, v := range crawlRes {
		parse, err := getHost(v.Url)
		if err != nil {
			continue
		}
		if v.IsForm {
			if parse == c.host {
				dirRes.SameOriginForm = append(dirRes.SameOriginForm, v)
			} else {
				dirRes.ExternalForm = append(dirRes.ExternalForm, v)
			}
			continue
		}

		if parse == c.host {
			dirRes.SameOriginUrl = append(dirRes.SameOriginUrl, res.SameOriginUrl{BaseUrl: res.BaseUrl{Url: v.Url}})
		} else {
			if staticCheck(v.Url) {
				dirRes.ExternalStaticFileLink = append(dirRes.ExternalStaticFileLink, res.ExternalStaticFileLink{BaseUrl: res.BaseUrl{Url: v.Url}})
				continue
			}
			dirRes.ExternalLink = append(dirRes.ExternalLink, res.ExternalLink{BaseUrl: res.BaseUrl{Url: v.Url}})
		}
	}
	c.crawl.DoFinally()
	return dirRes
}
func (c *Crawler) ParamCrawl(ctx context.Context) []template.JsRes {
	var sameOriginRes []template.JsRes
	res := c.crawAllUrl([]template.JsRes{{Url: c.target, Method: "GET"}}, c.crawl, ctx)
	for _, v := range res {
		switch v.Method {
		case "GET":
			if v.IsForm && len(v.Param) == 0 {
				continue
			}
			if !v.IsForm && !strings.Contains(v.Url, "?") {
				continue
			}
		case "POST":
			if len(v.Param) == 0 {
				continue
			}
		}
		parse, err := getHost(v.Url)
		if err != nil {
			continue
		}
		if parse == c.host {
			sameOriginRes = append(sameOriginRes, v)
		}
	}
	c.crawl.DoFinally()
	return sameOriginRes
}
func (c *Crawler) NewNativeCrawler(threads int, proxyPool *httpProxyPool.HttpProxyPool) (*NativeCrawler, error) {
	return &NativeCrawler{
		headers:      c.nativeClient.Headers,
		threads:      threads,
		proxyPool:    proxyPool,
		nativeClient: c.nativeClient,
	}, nil
}
func (c *Crawler) NewFingerDetect(fingerPath string, score float32) (*fingerDetect.FingerDetect, error) {
	myIdentifier := httpServiceFingerScanEngine.NewIdentifier(nil)
	document, err := httpServiceFingerScanEngine.ParseFileDocument(fingerPath)
	if err != nil {
		return nil, err
	}
	myIdentifier.Patch(document.Probes, document.Tags)
	return &fingerDetect.FingerDetect{
		Identifier:   myIdentifier,
		NativeClient: c.nativeClient,
		Score:        score,
	}, nil
}
func (c *Crawler) NewFingerDetectWithBytesDocument(fingerData [][]byte, score float32) (*fingerDetect.FingerDetect, error) {
	var document *httpServiceFingerScanEngine.Document
	var err error
	myIdentifier := httpServiceFingerScanEngine.NewIdentifier(nil)
	for _, v := range fingerData {
		document, err = httpServiceFingerScanEngine.ParseDocument(v)
	}
	if err != nil || document == nil {
		return nil, err
	}
	myIdentifier.Patch(document.Probes, document.Tags)
	return &fingerDetect.FingerDetect{
		Identifier:   myIdentifier,
		NativeClient: c.nativeClient,
		Score:        score,
	}, nil
}
func (c *Crawler) NewChromeCrawler(waitTime time.Duration, headless bool, chromeThreads int) (*ChromeCrawler, error) {
	myChrome, err := chrome.NewChrome(
		chromedp.Flag("headless", headless),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)

	if err != nil {
		return nil, err
	}
	return &ChromeCrawler{
		headers:       c.BaseCrawler.nativeClient.Headers,
		timeout:       c.timeout,
		waitTime:      waitTime,
		printLog:      false,
		chromeThreads: chromeThreads,
		chrome:        myChrome,
	}, nil
}
