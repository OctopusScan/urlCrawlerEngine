package fingerDetect

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/B9O2/Inspector/useful"
	"github.com/B9O2/Multitasking"
	ucdt "github.com/B9O2/UCDTParser"
	"github.com/OctopusScan/httpServiceFingerScanEngine"
	"github.com/OctopusScan/urlCrawlerEngine/http"
	"github.com/OctopusScan/urlCrawlerEngine/myClient"
	"github.com/OctopusScan/urlCrawlerEngine/res"
	. "github.com/OctopusScan/urlCrawlerEngine/runtime"
	"io"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"
)

type CrawlerResAndFingerRes struct {
	CrawlerRes res.DirResult          `json:"crawler_res"`
	FingerRes  map[string][]FingerRes `json:"finger_res"`
}

type Product struct {
	Name    string   `json:"name,omitempty"`
	Matched []string `json:"match,omitempty"`
}

type FingerRes struct {
	host       string
	Group      string    `json:"group"`
	DetectPath []string  `json:"detect_path"`
	Product    []Product `json:"product"`
}

type detectObj struct {
	scheme   string
	path     []string
	rawQuery string
}

type son struct {
	scheme string
	group  string
	sons   string
}
type FingerDetect struct {
	Identifier   *httpServiceFingerScanEngine.Identifier
	headers      map[string]interface{}
	NativeClient *myClient.NativeClient
	Score        float32
}

func (f *FingerDetect) Detect(crawlRes res.DirResult, level int, reqThreads, detectThreads uint) (map[string][]FingerRes, error) {
	var finalResult = make(map[string][]FingerRes)
	var detectObjs = make(map[string][]detectObj)
	var reqMap = make(map[string][]son)
	var allUrls []string
	var iconHost []string
	for _, v := range crawlRes.SameOriginUrl {
		allUrls = append(allUrls, v.Url)
	}
	//for _, v := range crawlRes.ExternalStaticFileLink {
	//	allUrls = append(allUrls, v.Url)
	//}
	//for _, v := range crawlRes.ExternalLink {
	//	allUrls = append(allUrls, v.Url)
	//}
	for _, v := range allUrls {
		parse, err := url.Parse(v)
		if err != nil {
			continue
		}
		var path []string
		for _, p := range strings.Split(parse.Path, "/") {
			if p != "" {
				path = append(path, p)
			}
		}
		if len(path) != 0 {
			detectObjs[parse.Host] = append(detectObjs[parse.Host], detectObj{
				scheme:   parse.Scheme,
				path:     path,
				rawQuery: parse.RawQuery,
			})
		}
		if !slices.Contains(iconHost, parse.Host) {
			detectObjs[parse.Host] = append(detectObjs[parse.Host], detectObj{
				scheme:   parse.Scheme,
				path:     []string{"favicon.ico"},
				rawQuery: "",
			})
			detectObjs[parse.Host] = append(detectObjs[parse.Host], detectObj{
				scheme:   parse.Scheme,
				path:     []string{"/"},
				rawQuery: "",
			})
			iconHost = append(iconHost, parse.Host)
		}

	}
	for k, v := range detectObjs {
		for _, v1 := range v {
			if level == 999 {
				var group string
				if v1.rawQuery == "" {
					group = strings.Join(v1.path, "/")
				} else {
					group = strings.Join(v1.path, "/") + "?" + v1.rawQuery
				}
				reqMap[k] = append(reqMap[k], son{
					scheme: v1.scheme,
					group:  group,
					sons:   "",
				})
			} else {
				if len(v1.path) >= level {
					var group string
					if level == 0 {
						group = "/"
					} else {
						group = strings.Join(v1.path[0:level], "/")
					}
					var sons string
					if v1.rawQuery == "" {
						sons = strings.Join(v1.path[level:], "/")
					} else {
						sons = strings.Join(v1.path[level:], "/") + "?" + v1.rawQuery
					}
					reqMap[k] = append(reqMap[k], son{
						scheme: v1.scheme,
						group:  group,
						sons:   sons,
					})
				} else {
					var group string
					if v1.rawQuery == "" {
						group = strings.Join(v1.path, "/")
					} else {
						group = strings.Join(v1.path, "/") + "?" + v1.rawQuery
					}
					reqMap[k] = append(reqMap[k], son{
						scheme: v1.scheme,
						group:  group,
						sons:   "",
					})
				}
			}
		}
	}

	detectMT := Multitasking.NewMultitasking("detectMT", nil)
	detectMT.Register(func(dc Multitasking.DistributeController) {
		for host, dson := range reqMap {
			var groupMap = make(map[string][]string)
			for _, v := range dson {
				var target string
				var group string
				if v.sons == "" {
					target = f.urlAppend(v.scheme, host, v.group)
					group = target
					groupMap[group] = append(groupMap[group], target)
				} else {
					target = f.urlAppend(v.scheme, host, v.group, v.sons)
					group = f.urlAppend(v.scheme, host, v.group)
					groupMap[group] = append(groupMap[group], target)
				}
			}
			for k, v := range groupMap {
				dc.AddTask(map[string]interface{}{"group": k, "paths": v, "host": host})
			}

		}
	}, func(ec Multitasking.ExecuteController, a any) any {
		paths := a.(map[string]interface{})["paths"].([]string)
		group := a.(map[string]interface{})["group"].(string)
		host := a.(map[string]interface{})["host"].(string)
		var sourceDatas []map[string][]byte
		for _, v1 := range paths {
			resp, err, _ := http.Get(v1, f.NativeClient)
			MainInsp.Print(useful.INFO, useful.Text("FingerDetectReq:"+v1))
			rawResp, err := httputil.DumpResponse(resp, true)
			if err != nil {
				MainInsp.Print(useful.ERROR, useful.Text(fmt.Sprintf("指纹检测请求错误,target:%s err:%s", v1, err.Error())))
				continue
			}
			rawHeader, err := httputil.DumpResponse(resp, false)
			if err != nil {
				MainInsp.Print(useful.ERROR, useful.Text(fmt.Sprintf("指纹检测请求错误,target:%s err:%s", v1, err.Error())))
				continue
			}
			statusLine, rawHeader, _ := bytes.Cut(rawHeader, []byte("\n"))
			if err != nil {
				MainInsp.Print(useful.ERROR, useful.Text(fmt.Sprintf("指纹检测请求错误,target:%s err:%s", v1, err.Error())))
				continue
			}
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				MainInsp.Print(useful.ERROR, useful.Text(fmt.Sprintf("指纹检测请求错误,target:%s err:%s", v1, err.Error())))
				continue
			}

			sourceDatas = append(sourceDatas, map[string][]byte{"raw": rawResp, "header": rawHeader, "status_line": statusLine, "body": respBody, "md5": []byte(getMd5(respBody))})
		}
		mrs, err := f.Identifier.Identify(detectThreads, ucdt.NewNoSourceData(sourceDatas...)...)
		if err != nil {
			MainInsp.Print(useful.ERROR, useful.Text(fmt.Sprintf("指纹检测错误,host:%s err:%s", host, err.Error())))
			return nil
		}
		var products []Product
		mrs.Range(func(tag string, result ucdt.MatchResult) bool {
			if result.Score >= f.Score {
				var matched []string
				for k, v := range result.ScoreDetail {
					if k {
						for k1, _ := range v {
							matched = append(matched, k1)
						}
					}
				}
				products = append(products, Product{
					Name:    tag,
					Matched: matched,
				})

			}
			return true
		})
		return FingerRes{
			host:       host,
			Group:      group,
			DetectPath: paths,
			Product:    products,
		}
	})

	res, _ := detectMT.Run(context.Background(), reqThreads)
	for _, v := range res {
		if v != nil {
			finalResult[v.(FingerRes).host] = append(finalResult[v.(FingerRes).host], v.(FingerRes))
		}
	}
	return finalResult, nil
}
func (f *FingerDetect) urlAppend(scheme, host string, paths ...string) string {
	var target string
	host = strings.Trim(host, "/")
	target = scheme + "://" + host
	for _, p := range paths {
		if p == "/" {
			continue
		}
		target = target + "/" + strings.Trim(p, "/")
	}
	return target
}
func getMd5(data []byte) string {
	hash := md5.Sum(data)
	md5Str := hex.EncodeToString(hash[:])
	return md5Str
}
