// Package cmd 提供命令的执行,以及命令行参数的解析
package cmd

import (
	"bufio"
	"buster/lib"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
)

var rootCmd = &cobra.Command{
	Use:               os.Args[0],
	SilenceUsage:      true,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

var mainCtx context.Context //用于控制起始后的所有协程

func Execute() {
	//初始化根 Ctx以及cancleFunc
	var cancel context.CancelFunc
	mainCtx, cancel = context.WithCancel(context.Background())
	defer cancel() //确保执行cancle

	//监听中断信号
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		select {
		case <-signalChan: //收到退出信号,将所有的协程退出
			fmt.Println("\n[!] Keyboard interrupt detected, terminating.")
			cancel()
		case <-mainCtx.Done():
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("rootCmd Execute Fail:", err)
		return
	}
}

//初始化全局的flag
func init() {
	rootCmd.PersistentFlags().DurationP("delay", "", 0, "Time each thread waits between requests (e.g. 1500ms)") //底层调用ParseDuration解析不同的时间格式
	rootCmd.PersistentFlags().IntP("threads", "t", 100, "Number of concurrent threads")
	rootCmd.PersistentFlags().StringP("wordlist", "w", "", "Path to the wordlist")
	rootCmd.PersistentFlags().StringP("output", "o", "", "Output file to write results to (defaults to stdout)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output (errors)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Don't print the banner and other noise")
	rootCmd.PersistentFlags().BoolP("no-progress", "z", false, "Don't display progress")
	rootCmd.PersistentFlags().Bool("no-error", false, "Don't display errors")
	rootCmd.PersistentFlags().StringP("pattern", "p", "", "File containing replacement patterns")

}

//注意,不能在init中执行,否则将无法适用-h --help命令
func configureGlobalOptions() {
	if err := rootCmd.MarkPersistentFlagRequired("wordlist"); err != nil {
		log.Fatalf("error on marking flag as required: %v", err)
	}
}

//获取全局的配置选项,供各个子命令调用
func parseGolobalOptions() (*lib.Options, error) {
	globalopts := lib.NewOptions()
	//依次读取flag,将全局配置赋值
	threads, err := rootCmd.Flags().GetInt("threads")
	if err != nil {
		return nil, fmt.Errorf("invalid value for threads:%w", err)
	}
	if threads <= 0 {
		return nil, fmt.Errorf("threads must be bigger than 0")
	}
	globalopts.Threads = threads
	//每个工作线程中,多个reques的间隔
	delay, err := rootCmd.Flags().GetDuration("delay")
	if err != nil {
		return nil, fmt.Errorf("invalid value for delay: %w", err)
	}

	if delay < 0 {
		return nil, fmt.Errorf("delay must be positive")
	}
	globalopts.Delay = delay

	//本设置为了必须字段,指定目录的字典文件
	globalopts.Wordlist, err = rootCmd.Flags().GetString("wordlist")
	if err != nil {
		return nil, fmt.Errorf("invalid value for wordlist: %w", err)
	}

	if globalopts.Wordlist == "-" {
		// STDIN
	} else if _, err2 := os.Stat(globalopts.Wordlist); os.IsNotExist(err2) {
		return nil, fmt.Errorf("wordlist file %q does not exist: %w", globalopts.Wordlist, err2)
	}

	//替换模式?
	globalopts.PatternFile, err = rootCmd.Flags().GetString("pattern")
	if err != nil {
		return nil, fmt.Errorf("invalid value for pattern: %w", err)
	}

	if globalopts.PatternFile != "" {
		if _, err = os.Stat(globalopts.PatternFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("pattern file %q does not exist: %w", globalopts.PatternFile, err)
		}
		patternFile, err := os.Open(globalopts.PatternFile)
		if err != nil {
			return nil, fmt.Errorf("could not open pattern file %q: %w", globalopts.PatternFile, err)
		}
		defer patternFile.Close()

		scanner := bufio.NewScanner(patternFile)
		for scanner.Scan() {
			globalopts.Patterns = append(globalopts.Patterns, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("could not read pattern file %q: %w", globalopts.PatternFile, err)
		}
	}

	globalopts.OutputFilename, err = rootCmd.Flags().GetString("output")
	if err != nil {
		return nil, fmt.Errorf("invalid value for output filename: %w", err)
	}

	//详细的错误信息
	globalopts.Verbose, err = rootCmd.Flags().GetBool("verbose")
	if err != nil {
		return nil, fmt.Errorf("invalid value for verbose: %w", err)
	}

	//简洁的输出
	globalopts.Quiet, err = rootCmd.Flags().GetBool("quiet")
	if err != nil {
		return nil, fmt.Errorf("invalid value for quiet: %w", err)
	}

	//不打印进度条
	globalopts.NoProgress, err = rootCmd.Flags().GetBool("no-progress")
	if err != nil {
		return nil, fmt.Errorf("invalid value for no-progress: %w", err)
	}

	//不输出错误
	globalopts.NoError, err = rootCmd.Flags().GetBool("no-error")
	if err != nil {
		return nil, fmt.Errorf("invalid value for no-error: %w", err)
	}

	return globalopts, nil
}
