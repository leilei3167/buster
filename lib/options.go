package lib

import "time"

// Options 全局的配置,用于容纳rootCmd的flag选项
type Options struct {
	Threads        int
	Wordlist       string
	PatternFile    string
	Patterns       []string
	OutputFilename string
	NoStatus       bool
	NoProgress     bool
	NoError        bool
	Quiet          bool
	Verbose        bool
	Delay          time.Duration
}

func NewOptions() *Options {
	return new(Options)
}
