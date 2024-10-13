package crawler

import (
	"context"
	"fmt"
	"github.com/B9O2/Inspector/useful"
	"github.com/B9O2/Multitasking"
	"github.com/Kumengda/easyChromedp/template"
	"github.com/OctopusScan/urlCrawlerEngine/myClient"
	. "github.com/OctopusScan/urlCrawlerEngine/runtime"
	"time"
)

type Crawl interface {
	SingleCrawl(task template.JsRes, allHref []template.JsRes) []template.JsRes
	DoFinally()
	GetCrawlThreads() int
}

type BaseCrawler struct {
	dirScanTier        int
	usingDirScanRes    bool
	webDirBlastThreads uint
	maxResultNum       int
	target             string
	timeout            time.Duration
	depth              int
	host               string
	filter             []string
	nativeClient       *myClient.NativeClient
	middlewareFunc     func(i interface{}) interface{}
}

func (b *BaseCrawler) SetMiddlewareFunc(middlewareFunc func(i interface{}) interface{}) {
	b.middlewareFunc = middlewareFunc
}

// 第一次入参targets只能是一个包含一个目标的切片类型,这里第一个参数写成[]string是方便递归传参
func (b *BaseCrawler) crawAllUrl(targets []template.JsRes, crawl Crawl, ctx context.Context) []template.JsRes {
	var finalRes []template.JsRes
	var lastTargets []template.JsRes
	for {
		if b.depth == 0 {
			return targetRemoveDuplicates(finalRes)
		}
		b.depth = b.depth - 1
		targets = targetRemoveDuplicates(targets)
		if compareCompareJsRes(targets, lastTargets) {
			return targets
		}
		if len(targets) >= b.maxResultNum {
			return targets
		}
		myMT := Multitasking.NewMultitasking("crawler", nil)
		myMT.Register(func(dc Multitasking.DistributeController) {
			for _, v := range targets {
				if lastTargets == nil {
					//证明是第一次
					dc.AddTask(v)
				} else {
					if !containsString(lastTargets, v.Url) {
						dc.AddTask(v)
					}
				}
			}
		}, func(ec Multitasking.ExecuteController, i interface{}) interface{} {
			select {
			case <-ctx.Done():
				return nil
			default:

			}
			var _allHref []template.JsRes
			task := i.(template.JsRes)
			if !continueCheck(task.Url, b.host, b.filter) {
				return append(_allHref, template.JsRes{Url: task.Url, Method: "GET"})
			}
			MainInsp.Print(useful.INFO, useful.Text(fmt.Sprintf("depth:%d Req:%s", b.depth, task.Url)))
			return crawl.SingleCrawl(task, _allHref)
		})
		myMT.SetResultMiddlewares(Multitasking.NewBaseMiddleware(func(ec Multitasking.ExecuteController, i interface{}) (interface{}, error) {
			if b.middlewareFunc != nil {
				midres := b.middlewareFunc(i)
				if midres == nil {
					return i, nil
				}
				return midres, nil
			}
			return i, nil
		}))
		res, err := myMT.Run(context.Background(), uint(crawl.GetCrawlThreads()))
		if err != nil {
			return nil
		}
		for _, v := range res {
			if v != nil {
				finalRes = append(finalRes, v.([]template.JsRes)...)
			}
		}
		lastTargets = targets
		targets = finalRes
	}

}
