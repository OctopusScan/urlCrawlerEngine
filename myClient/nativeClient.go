package myClient

import (
	"context"
	"crypto/tls"
	"golang.org/x/time/rate"
	"net"
	http2 "net/http"
	"net/url"
	"time"
)

type NativeClient struct {
	Headers       map[string]string
	Client        *http2.Client
	MaxRespSize   int64
	Limiter       *rate.Limiter
	LimiterCtx    context.Context
	MaxRetryCount int
}

type NativeClientOptions struct {
	Headers       map[string]string
	ProxyUrl      string
	MaxRedirect   int
	Timeout       time.Duration
	ConnTimeout   time.Duration
	RateLimit     int
	MaxRespSize   int64
	MaxRetryCount int
}

func NewNativeOptions(headers map[string]string, proxyUrl string, maxRedirect int, timeout, connTimeout time.Duration, rateLimit, maxRetryCount int, maxRespSize int64) NativeClientOptions {
	if headers == nil {
		headers = make(map[string]string)
	}
	return NativeClientOptions{
		Headers:       headers,
		ProxyUrl:      proxyUrl,
		MaxRedirect:   maxRedirect,
		Timeout:       timeout,
		RateLimit:     rateLimit,
		MaxRespSize:   maxRespSize,
		ConnTimeout:   connTimeout,
		MaxRetryCount: maxRetryCount,
	}
}

func NewNativeClient(proxyUrl string, maxRedirect int, timeout, connTimeout time.Duration, rateLimit int, maxRespSize int64, headers map[string]string, maxRetryCount int) (*NativeClient, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	var client *http2.Client
	proxyURL, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}
	checkRedirect := func(req *http2.Request, via []*http2.Request) error {
		if len(via) >= maxRedirect {
			return http2.ErrUseLastResponse
		}
		return nil
	}
	if proxyUrl != "" {
		client = &http2.Client{
			CheckRedirect: checkRedirect,
			Timeout:       timeout,
			Transport: &http2.Transport{
				DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
					return net.DialTimeout(network, addr, connTimeout)
				},
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				Proxy:           http2.ProxyURL(proxyURL),
			}}
	} else {
		client = &http2.Client{
			CheckRedirect: checkRedirect,
			Timeout:       timeout,
			Transport: &http2.Transport{
				DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
					return net.DialTimeout(network, addr, connTimeout)
				},
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}}
	}
	limiter := rate.NewLimiter(rate.Limit(rateLimit), rateLimit)
	return &NativeClient{
		Headers:       headers,
		Client:        client,
		Limiter:       limiter,
		MaxRespSize:   maxRespSize,
		LimiterCtx:    context.Background(),
		MaxRetryCount: maxRetryCount,
	}, nil
}
