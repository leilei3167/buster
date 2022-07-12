package lib

import "context"

// GobusterPlugin 可执行目录爆破的接口
type GobusterPlugin interface {
	Name() string
	RequestPerRun() int

	// PreRun 在正式开始任务前,执行某些操作(如检查连接有效性等)
	PreRun(context.Context) error

	// Run 开始执行,并将结果写入到一个chan Result中
	Run(context.Context, string, chan<- Result) error
	// GetConfigString 将配置转换为string
	GetConfigString() (string, error)
}

type Result interface {
	ResulToString() (string, error)
}
