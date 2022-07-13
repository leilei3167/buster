package lib

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

// RequestOptions is used to pass options to a single individual request
type RequestOptions struct {
	Host       string
	Body       io.Reader
	ReturnBody bool
}

func NewHTTPClient(opt *HTTPOptions) (*HTTPClient, error) {
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

// Request 对目标发起http请求
func (client *HTTPClient) Request(ctx context.Context, fullURL string,
	opts RequestOptions) (*int, int64, http.Header, []byte, error) {
	resp, err := client.makeRequest(ctx, fullURL, opts.Host, opts.Body)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, 0, nil, nil, nil //ctx取消不做处理
		}
		return nil, 0, nil, nil, err
	}
	defer resp.Body.Close()

	var body []byte
	var length int64

	if opts.ReturnBody {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, 0, nil, nil, fmt.Errorf("could not read body: %w", err)
		}
		length = int64(len(body))
	} else {
		//TODO:目录爆破本质上是对同一个url发起多次http请求,必须将body读取完毕并关闭以确保可以复用底层的tcp连接
		//即使不需要body也需要将body内容全部读取完毕,否则无法复用连接
		length, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			return nil, 0, nil, nil, err
		}
	}
	return &resp.StatusCode, length, resp.Header, body, nil

}

func (client *HTTPClient) makeRequest(ctx context.Context, fullURL string, host string, data io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, client.method, fullURL, data)
	if err != nil {
		return nil, err
	}
	if client.cookies != "" {
		req.Header.Set("Cookie", client.cookies)
	}

	if host != "" {
		req.Host = host
	} else if client.host != "" {
		req.Host = client.host
	}

	if client.userAgent != "" {
		req.Header.Set("User-Agent", client.userAgent)
	} else {
		req.Header.Set("User-Agent", client.defaultUserAgent)
	}

	// add custom headers
	for _, h := range client.headers {
		req.Header.Set(h.Name, h.Value)
	}

	if client.username != "" {
		req.SetBasicAuth(client.username, client.password)
	}

	resp, err := client.client.Do(req)
	if err != nil {
		//注意适用As时必须传入一个指针，该指针指向接口或指向实现错误接口的类型,此处实现Error接口的时*url.Error
		var ue *url.Error
		if errors.As(err, &ue) {
			if strings.HasPrefix(ue.Err.Error(), "x509") {
				return nil, fmt.Errorf("invalid certificate:%w", ue.Err)
			}
		}
		//其他错误,如ctx取消
		return nil, err
	}
	return resp, nil
}
