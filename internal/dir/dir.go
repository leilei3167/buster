package dir

import (
	"buster/lib"
	"context"
	"fmt"
)

type ErrWildcard struct {
	url        string
	statusCode int
	length     int64
}

func (e *ErrWildcard) Error() string {
	return fmt.Sprintf("the server returns a status code that matches the provided options for non existing urls. %s => %d (Length: %d)", e.url, e.statusCode, e.length)

}

// GobusterDir dir模式的核心实现,实现plugin接口
type GobusterDir struct {
	options       *OptionsDir
	globalopts    *lib.Options
	http          *lib.HTTPClient
	requestPerRun *int
}

func (g GobusterDir) Name() string {
	return "directory enumeration"
}

func (g GobusterDir) RequestPerRun() int {
	//TODO implement me
	panic("implement me")
}

func (g GobusterDir) PreRun(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (g GobusterDir) Run(ctx context.Context, s string, results chan<- lib.Result) error {
	//TODO implement me
	panic("implement me")
}

func (g GobusterDir) GetConfigString() (string, error) {
	//TODO implement me
	panic("implement me")
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
	h, err := lib.NewHTTPClien(&httpOpts)
	if err != nil {
		return nil, err
	}
	g.http = h
	return &g, nil
}
