package lib

import "context"

// GobusterPlugin 可执行目录爆破的接口
type GobusterPlugin interface {
	Name() string
	RequestPerRun() int
	PreRun(context.Context) error
	Run(context.Context, string, chan<- Result) error //结果是一个接口,后续可进行更改
	GetConfigString() (string, error)
}

type Result interface {
	ResulToString() (string, error)
}
