package http

import (
	"bytes"
	"github.com/OctopusScan/urlCrawlerEngine/myClient"
	"github.com/corpix/uarand"
	"io"
	"net/http"
)

func doGet(url string, client *myClient.NativeClient) (*http.Response, error, string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err, ""
	}
	for k, v := range client.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", uarand.GetRandom())
	resp, err := client.Client.Do(req)
	if err != nil {
		return nil, err, ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, client.MaxRespSize))
	if err != nil {
		return nil, err, ""
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return resp, nil, resp.Header.Get("Content-Type")
}

func Get(url string, client *myClient.NativeClient) (*http.Response, error, string) {
	client.Limiter.Wait(client.LimiterCtx)
	reqNum := 0
	for {
		resp, err, ct := doGet(url, client)
		reqNum += 1
		if err == nil {
			return resp, err, ct
		}
		if reqNum > client.MaxRetryCount {
			return resp, err, ct
		}

	}
}
