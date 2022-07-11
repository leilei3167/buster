package lib

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
)

type HTTPHeader struct {
	Name  string
	Value string
}

type HTTPClient struct {
	client           *http.Client
	userAgent        string
	defaultUserAgent string
	username         string
	password         string
	headers          []HTTPHeader
	cookies          string
	method           string
	host             string
}

func NewHTTPClien(opt *HTTPOptions) (*HTTPClient, error) {
	var client HTTPClient

	//配置代理
	var proxyURLFunc func(*http.Request) (*url.URL, error)
	proxyURLFunc = http.ProxyFromEnvironment //设置默认代理
	if opt == nil {
		return nil, fmt.Errorf("options is nil")
	}

	if opt.Proxy != "" {
		proxyURL, err := url.Parse(opt.Proxy)
		if err != nil {
			return nil, fmt.Errorf("proxy URL is invalid (%w)", err)
		}
		proxyURLFunc = http.ProxyURL(proxyURL)
	}

	//配置重定向(第一个参数是即将转发的请求,via是之前执行过的请求,默认的配置会判断via的长度来控制跳转次数)
	var redirectFunc func(req *http.Request, via []*http.Request) error
	if !opt.FollowRedirect {
		redirectFunc = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		redirectFunc = nil //跟随重定向,设置为nil,会使用默认的重定向策略(最多跟随10次)
	}

	client.client = &http.Client{
		Timeout:       opt.Timeout,
		CheckRedirect: redirectFunc,
		Transport: &http.Transport{
			Proxy:               proxyURLFunc,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opt.NoTLSValidation,
			},
		}}

	client.username = opt.Username
	client.password = opt.Password
	client.userAgent = opt.UserAgent
	client.defaultUserAgent = DefaultUserAgent()
	client.headers = opt.Headers
	client.cookies = opt.Cookies
	client.method = opt.Method
	if client.method == "" {
		client.method = http.MethodGet
	}
	for _, h := range opt.Headers {
		if h.Name == "Host" {
			client.host = h.Value
			break
		}
	}
	return &client, nil
}
