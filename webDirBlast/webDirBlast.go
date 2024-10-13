package webDirBlast

import (
	"context"
	"fmt"
	"github.com/B9O2/Inspector/useful"
	"github.com/B9O2/Multitasking"
	"github.com/OctopusScan/urlCrawlerEngine/http"
	"github.com/OctopusScan/urlCrawlerEngine/myClient"
	. "github.com/OctopusScan/urlCrawlerEngine/runtime"
	"github.com/OctopusScan/urlCrawlerEngine/utils"
	"net/url"
	"strings"
)

type BlastRes struct {
	Url        string
	StatusCode int
}

type WebDirBlastPath struct {
	Data               string `json:"data"`
	AppendRedirectPath bool   `json:"append_redirect_path"`
}

type WebDirBlast struct {
	dirScanTier int
	myClient    *myClient.NativeClient
	target      string
	threads     uint
}

func NewWebDirBlast(target string, myClient *myClient.NativeClient, threads uint, dirScanTier int) WebDirBlast {
	return WebDirBlast{myClient: myClient, target: target, threads: threads, dirScanTier: dirScanTier}
}
func (w *WebDirBlast) randomCheck(redirectUrl, rawTargetUrl string) bool {
	rawTargetUrl = strings.TrimLeft(rawTargetUrl, "/") + "/"
	chechPath1 := rawTargetUrl + utils.GenerateRandomString(5)
	response1, err, _ := http.Get(chechPath1, w.myClient)
	if err == nil {
		if response1.StatusCode == 200 {
			return false
		}
	}
	redirectUrl = strings.TrimRight(redirectUrl, "/") + "/"
	checkPath2 := redirectUrl + utils.GenerateRandomString(5)
	response2, err, _ := http.Get(checkPath2, w.myClient)
	if err == nil {
		if response2.StatusCode == 200 {
			return false
		}
	}
	return true
}

func (w *WebDirBlast) DoBlast(webDirBlast []WebDirBlastPath) ([]string, error) {
	var existPath []string
	var redirectUrl string
	var redirectBaseUrl string
	resp, err, _ := http.Get(w.target, w.myClient)
	if err != nil {
		return nil, err
	}
	redirectUrl = strings.TrimRight(resp.Request.URL.String(), "/")
	redirectBaseUrl = resp.Request.URL.Scheme + "://" + resp.Request.URL.Host
	rawTargetUrl := strings.TrimRight(w.target, "/")
	rawTargetBaseParseUrl, err := url.Parse(rawTargetUrl)
	if err != nil {
		return existPath, err
	}
	rawTargerBaseUrl := rawTargetBaseParseUrl.Scheme + "://" + rawTargetBaseParseUrl.Host

	if !w.randomCheck(redirectUrl, rawTargetUrl) {
		return nil, nil
	}
	blastMT := Multitasking.NewMultitasking("blastMT", nil)
	blastMT.Register(func(dc Multitasking.DistributeController) {
		for _, v := range webDirBlast {
			var reqUrl string
			if v.AppendRedirectPath {
				for _, p := range w.splitUrlPath(redirectUrl) {
					path := "/" + strings.TrimLeft(p, "/")
					if p == "" {
						path = ""
					}
					reqUrl = redirectBaseUrl + path + "/" + strings.TrimLeft(v.Data, "/")
					dc.AddTask(reqUrl)
				}
			} else {
				for _, p := range w.splitUrlPath(rawTargetUrl) {
					path := "/" + strings.TrimLeft(p, "/")
					if p == "" {
						path = ""
					}
					reqUrl = rawTargerBaseUrl + path + "/" + strings.TrimLeft(v.Data, "/")
					dc.AddTask(reqUrl)
				}
			}
		}
	}, func(ec Multitasking.ExecuteController, a any) any {
		reqUrl := a.(string)
		MainInsp.Print(useful.INFO, useful.Text(fmt.Sprintf("dirScanUrl:%s", reqUrl)))
		response, err, _ := http.Get(reqUrl, w.myClient)
		if err != nil {
			return nil
		}
		return BlastRes{
			Url:        reqUrl,
			StatusCode: response.StatusCode,
		}
	})
	run, err := blastMT.Run(context.Background(), w.threads)
	if err != nil {
		return nil, err
	}
	for _, v := range run {
		if v != nil {
			res := v.(BlastRes)
			switch res.StatusCode {
			case 403, 406, 401, 200, 301:
				existPath = append(existPath, res.Url)
			}
		}
	}
	return existPath, nil
}
func (w *WebDirBlast) splitUrlPath(splitTarget string) []string {
	var urlpaths []string
	purl, err := url.Parse(splitTarget)
	if err != nil {
		return urlpaths
	}
	paths := strings.Split(strings.Trim(purl.Path, "/"), "/")
	if w.dirScanTier > len(paths) {
		for i := 0; i < len(paths); i++ {
			urlpaths = append(urlpaths, strings.Join(paths[0:i+1], "/"))
		}
	} else {
		for i := 0; i < w.dirScanTier; i++ {
			urlpaths = append(urlpaths, strings.Join(paths[0:i+1], "/"))
		}
	}
	return urlpaths
}
