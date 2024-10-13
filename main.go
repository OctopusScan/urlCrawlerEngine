package main

import (
	"github.com/B9O2/Inspector/useful"
	"github.com/OctopusScan/urlCrawlerEngine/crawler"
	"github.com/OctopusScan/urlCrawlerEngine/myClient"
	. "github.com/OctopusScan/urlCrawlerEngine/runtime"
	"time"
)

func main() {
	nativeClientOptions := myClient.NewNativeOptions(
		map[string]string{"Cookie": "PHPSESSID=58hdcthql1fd9h02ptfm914705; security=low"},
		"",
		5,
		5*time.Second,
		5*time.Second,
		50,
		5,
		1024*1024,
	)

	crawlerOptions := crawler.NewCrawlerOptions(
		nil,
		2,
		[]string{".*logout.*"},
		1000,
		nativeClientOptions,
		10,
		false,
		2)

	myCrawler, _ := crawler.NewCrawler("http://127.0.0.1:9077", crawlerOptions)

	chromeCrawler, _ := myCrawler.NewChromeCrawler(5, false, 20)
	myCrawler.LoadWebDirBlastPath([][]byte{[]byte("{\"data\":\"abc/def\",\"append_redirect_path\":false}")})
	//nativeCrawler, _ := myCrawler.NewNativeCrawler(10, nil)
	myCrawler.SetCrawler(chromeCrawler)
	myCrawler.SetMiddlewareFunc(func(i interface{}) interface{} {
		return i
	})
	res := myCrawler.Crawl()
	MainInsp.Print(useful.Json(res))
	//detect, err := myCrawler.NewFingerDetect("./auto_tags.toml", 1)
	//if err != nil {
	//	MainInsp.Print(LEVEL_ERROR, Text(err))
	//	return
	//}
	//finalRes, err := detect.Detect(res, 1, 20, 20)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//marshal, err := json.Marshal(finalRes)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//fileutil.WriteStringToFile("./res.json", string(marshal), false)

	//fmt.Println(len(res.ExternalLink) + len(res.ExternalStaticFileLink) + len(res.SameOriginUrl))
}
