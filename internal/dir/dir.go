package dir

import (
	"bufio"
	"buster/helper"
	"buster/lib"
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"text/tabwriter"
)

var (
	backupExtensions    = []string{"~", ".bak", ".bak2", ".old", ".1"}
	backupDotExtensions = []string{".swp"}
)

type ErrWildcard struct {
	url        string
	statusCode int
	length     int64
}

func (e *ErrWildcard) Error() string {
	return fmt.Sprintf("the server returns a status code that matches the provided options for non existing urls. %s => %d (Length: %d)", e.url, e.statusCode, e.length)

}

//GobusterDir dir模式的核心实现,实现plugin接口,直接发起HTTP请求的结构
type GobusterDir struct {
	options       *OptionsDir
	globalopts    *lib.Options
	http          *lib.HTTPClient
	requestPerRun *int
}

// NewGobusterDir 根据全局的配置,和http的配置,生成GobusterDir(实现了plugin接口)
func NewGobusterDir(globalopts *lib.Options, opts *OptionsDir) (*GobusterDir, error) {
	if globalopts == nil {
		return nil, fmt.Errorf("please provide valid global options")
	}

	if opts == nil {
		return nil, fmt.Errorf("please provide valid plugin options")
	}

	g := GobusterDir{
		options:    opts,
		globalopts: globalopts,
	}
	basicOptions := lib.BasicHTTPOptions{
		Proxy:           opts.Proxy,
		Timeout:         opts.Timeout,
		UserAgent:       opts.UserAgent,
		NoTLSValidation: opts.NoTLSValidation,
	}

	httpOpts := lib.HTTPOptions{
		BasicHTTPOptions: basicOptions,
		FollowRedirect:   opts.FollowRedirect,
		Username:         opts.Username,
		Password:         opts.Password,
		Headers:          opts.Headers,
		Cookies:          opts.Cookies,
		Method:           opts.Method,
	}
	//适用http的配置创建http的client
	h, err := lib.NewHTTPClient(&httpOpts)
	if err != nil {
		return nil, err
	}
	g.http = h
	return &g, nil
}

func (d *GobusterDir) Name() string {
	return "directory enumeration"
}

// RequestPerRun 返回经过配置后,每一次Run将会发起的Request的数量
func (d *GobusterDir) RequestPerRun() int {
	if d.requestPerRun != nil {
		return *d.requestPerRun
	}

	num := 1 + len(d.options.ExtensionsParsed.Set)
	if d.options.DiscoverBackup {
		num += len(backupExtensions)
		num += len(backupDotExtensions)

		num += len(d.options.ExtensionsParsed.Set) * len(backupExtensions)
		num += len(d.options.ExtensionsParsed.Set) * len(backupDotExtensions)
	}
	d.requestPerRun = &num
	return *d.requestPerRun

}

// PreRun 在任务开始前执行(检查url连接,黑名单检查)
func (d *GobusterDir) PreRun(ctx context.Context) error {
	//对URL进行试探性连接
	if !strings.HasSuffix(d.options.URL, "/") {
		d.options.URL = d.options.URL + "/"
	}
	//舍弃结果,只检查是否能正常连接
	_, _, _, _, err := d.http.Request(ctx, d.options.URL, lib.RequestOptions{})
	if err != nil {
		return fmt.Errorf("unable to connect to %s: %w", d.options.URL, err)
	}

	guid := uuid.New()
	url := fmt.Sprintf("%s%s", d.options.URL, guid)
	if d.options.UseSlash {
		url = fmt.Sprintf("%s/", url)
	}
	wildcardResp, wildcardLength, _, _, err := d.http.Request(ctx, url, lib.RequestOptions{})
	if err != nil {
		return err
	}

	//判断是否在排除名单
	if helper.SliceContains(d.options.ExcludeLength, int(wildcardLength)) {
		// we are done and ignore the request as the length is excluded
		return nil
	}

	if d.options.StatusCodesBlacklistParsed.Length() > 0 {
		if !d.options.StatusCodesBlacklistParsed.Contains(*wildcardResp) {
			return &ErrWildcard{url: url, statusCode: *wildcardResp, length: wildcardLength}
		}
	} else if d.options.StatusCodesParsed.Length() > 0 {
		if d.options.StatusCodesParsed.Contains(*wildcardResp) {
			return &ErrWildcard{url: url, statusCode: *wildcardResp, length: wildcardLength}
		}
	} else {
		return fmt.Errorf("StatusCodes and StatusCodesBlacklist are both not set which should not happen")
	}

	return nil

}
func getBackupFilenames(word string) []string {
	ret := make([]string, len(backupExtensions)+len(backupDotExtensions))
	i := 0
	for _, b := range backupExtensions {
		ret[i] = fmt.Sprintf("%s%s", word, b)
		i++
	}
	for _, b := range backupDotExtensions {
		ret[i] = fmt.Sprintf(".%s%s", word, b)
		i++
	}

	return ret
}
func (d *GobusterDir) Run(ctx context.Context, word string, results chan<- lib.Result) error {
	suffix := ""
	if d.options.UseSlash {
		suffix = "/"
	}

	urlsToCheck := make(map[string]string)
	entity := fmt.Sprintf("%s%s", word, suffix)          //相对路径
	dirUrl := fmt.Sprintf("%s%s", d.options.URL, entity) //与url拼接成绝对路径
	urlsToCheck[entity] = dirUrl

	if d.options.DiscoverBackup { //是否找备份文件,添加到待扫描的列表中
		for _, u := range getBackupFilenames(word) {
			url := fmt.Sprintf("%s%s", d.options.URL, u)
			urlsToCheck[u] = url
		}
	}
	for ext := range d.options.ExtensionsParsed.Set {
		filename := fmt.Sprintf("%s.%s", word, ext)
		url := fmt.Sprintf("%s%s", d.options.URL, filename)
		urlsToCheck[filename] = url
		if d.options.DiscoverBackup {
			for _, u := range getBackupFilenames(filename) {
				url2 := fmt.Sprintf("%s%s", d.options.URL, u)
				urlsToCheck[u] = url2
			}
		}
	}

	for entity, url := range urlsToCheck {
		//发起http请求 获取结果
		statusCode, size, header, _, err := d.http.Request(ctx, url, lib.RequestOptions{})
		if err != nil {
			return err
		}

		if statusCode != nil {
			resultStatus := false

			//判断需要过滤的状态码
			if d.options.StatusCodesBlacklistParsed.Length() > 0 {
				if !d.options.StatusCodesBlacklistParsed.Contains(*statusCode) {
					resultStatus = true
				}
			} else if d.options.StatusCodesParsed.Length() > 0 {
				if d.options.StatusCodesParsed.Contains(*statusCode) {
					resultStatus = true
				}
			} else {
				return fmt.Errorf("StatusCodes and StatusCodesBlacklist are both not set which should not happen")
			}
			//构建结果返回
			if (resultStatus && !helper.SliceContains(d.options.ExcludeLength, int(size))) || d.globalopts.Verbose {
				results <- Result{
					URL:        d.options.URL,
					Path:       entity,
					Verbose:    d.globalopts.Verbose,
					Expanded:   d.options.Expanded,
					NoStatus:   d.options.NoStatus,
					HideLength: d.options.HideLength,
					Found:      resultStatus,
					Header:     header,
					StatusCode: *statusCode,
					Size:       size,
				}
			}
		}

	}
	return nil
}

func (d *GobusterDir) GetConfigString() (string, error) {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	tw := tabwriter.NewWriter(bw, 0, 5, 3, ' ', 0)
	o := d.options
	if _, err := fmt.Fprintf(tw, "[+] Url:\t%s\n", o.URL); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(tw, "[+] Method:\t%s\n", o.Method); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(tw, "[+] Threads:\t%d\n", d.globalopts.Threads); err != nil {
		return "", err
	}

	if d.globalopts.Delay > 0 {
		if _, err := fmt.Fprintf(tw, "[+] Delay:\t%s\n", d.globalopts.Delay); err != nil {
			return "", err
		}
	}
	wordlist := "stdin (pipe)"
	if d.globalopts.Wordlist != "-" {
		wordlist = d.globalopts.Wordlist
	}
	if _, err := fmt.Fprintf(tw, "[+] Wordlist:\t%s\n", wordlist); err != nil {
		return "", err
	}

	if d.globalopts.PatternFile != "" {
		if _, err := fmt.Fprintf(tw, "[+] Patterns:\t%s (%d entries)\n", d.globalopts.PatternFile, len(d.globalopts.Patterns)); err != nil {
			return "", err
		}
	}

	if o.StatusCodesBlacklistParsed.Length() > 0 {
		if _, err := fmt.Fprintf(tw, "[+] Negative Status codes:\t%s\n", o.StatusCodesBlacklistParsed.Stringify()); err != nil {
			return "", err
		}
	} else if o.StatusCodesParsed.Length() > 0 {
		if _, err := fmt.Fprintf(tw, "[+] Status codes:\t%s\n", o.StatusCodesParsed.Stringify()); err != nil {
			return "", err
		}
	}

	if len(o.ExcludeLength) > 0 {
		if _, err := fmt.Fprintf(tw, "[+] Exclude Length:\t%s\n", helper.JoinIntSlice(d.options.ExcludeLength)); err != nil {
			return "", err
		}
	}

	if o.Proxy != "" {
		if _, err := fmt.Fprintf(tw, "[+] Proxy:\t%s\n", o.Proxy); err != nil {
			return "", err
		}
	}

	if o.Cookies != "" {
		if _, err := fmt.Fprintf(tw, "[+] Cookies:\t%s\n", o.Cookies); err != nil {
			return "", err
		}
	}

	if o.UserAgent != "" {
		if _, err := fmt.Fprintf(tw, "[+] User Agent:\t%s\n", o.UserAgent); err != nil {
			return "", err
		}
	}

	if o.HideLength {
		if _, err := fmt.Fprintf(tw, "[+] Show length:\tfalse\n"); err != nil {
			return "", err
		}
	}

	if o.Username != "" {
		if _, err := fmt.Fprintf(tw, "[+] Auth User:\t%s\n", o.Username); err != nil {
			return "", err
		}
	}

	if o.Extensions != "" {
		if _, err := fmt.Fprintf(tw, "[+] Extensions:\t%s\n", o.ExtensionsParsed.Stringify()); err != nil {
			return "", err
		}
	}

	if o.UseSlash {
		if _, err := fmt.Fprintf(tw, "[+] Add Slash:\ttrue\n"); err != nil {
			return "", err
		}
	}

	if o.FollowRedirect {
		if _, err := fmt.Fprintf(tw, "[+] Follow Redirect:\ttrue\n"); err != nil {
			return "", err
		}
	}

	if o.Expanded {
		if _, err := fmt.Fprintf(tw, "[+] Expanded:\ttrue\n"); err != nil {
			return "", err
		}
	}

	if o.NoStatus {
		if _, err := fmt.Fprintf(tw, "[+] No status:\ttrue\n"); err != nil {
			return "", err
		}
	}

	if d.globalopts.Verbose {
		if _, err := fmt.Fprintf(tw, "[+] Verbose:\ttrue\n"); err != nil {
			return "", err
		}
	}

	if _, err := fmt.Fprintf(tw, "[+] Timeout:\t%s\n", o.Timeout.String()); err != nil {
		return "", err
	}

	if err := tw.Flush(); err != nil {
		return "", fmt.Errorf("error on tostring: %w", err)
	}

	if err := bw.Flush(); err != nil {
		return "", fmt.Errorf("error on tostring: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil

}
