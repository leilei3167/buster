package lib

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

//抽象一个执行爆破的对象

type Gobuster struct {
	Opts                           *Options
	RequestExpected, RequestIssued int
	RequestCountMutex              sync.RWMutex
	plugin                         GobusterPlugin //不同扫描模式实现的关键,以插件的形式体现
	resultChan                     chan Result
	errorChan                      chan error
	LogInfo, LogError              *log.Logger
}

func NewGobuster(opts *Options, plugin GobusterPlugin) (*Gobuster, error) {
	return &Gobuster{
		Opts:              opts,
		RequestExpected:   0,
		RequestIssued:     0,
		RequestCountMutex: sync.RWMutex{},
		plugin:            plugin,
		resultChan:        make(chan Result, 1),
		errorChan:         make(chan error, 1),
		LogInfo:           log.New(os.Stdout, "", log.LstdFlags),
		LogError:          log.New(os.Stdout, "[ERROR]", log.LstdFlags),
	}, nil
}

func (g *Gobuster) Results() <-chan Result {
	return g.resultChan
}

func (g *Gobuster) Errors() <-chan error {
	return g.errorChan
}
func (g *Gobuster) GetConfigString() (string, error) {
	return g.plugin.GetConfigString()
}
func (g *Gobuster) incrementRequest() {
	g.RequestCountMutex.Lock()
	defer g.RequestCountMutex.Unlock()
	g.RequestIssued += g.plugin.RequestPerRun()
}

// Run 开始解析Wordlist,生产任务;并开启指定数量的worker进行并发执行
func (g *Gobuster) Run(ctx context.Context) error {
	defer close(g.resultChan)
	defer close(g.errorChan)

	if err := g.plugin.PreRun(ctx); err != nil {
		return err
	}

	//开启worker,进行消费
	var workerGroup sync.WaitGroup
	workerGroup.Add(g.Opts.Threads)

	wordChan := make(chan string, g.Opts.Threads)
	for i := 0; i < g.Opts.Threads; i++ {
		go g.worker(ctx, wordChan, &workerGroup)
	}

	//开始生产任务
	scanner, err := g.getWordList()
	if err != nil {
		return err
	}
SCAN:
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			break SCAN
		default:
			word := scanner.Text()
			wordChan <- word

		}
	}

	//生产完毕关闭wordChan,能够使所有Worker感知到
	close(wordChan)
	workerGroup.Wait()
	return nil
}

func (g *Gobuster) worker(ctx context.Context, wordChan <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return

		case word, ok := <-wordChan:
			if !ok {
				return
			}
			g.incrementRequest()

			wordCleaned := strings.TrimSpace(word)
			//舍弃无效的
			if strings.HasPrefix(wordCleaned, "#") || len(wordCleaned) == 0 {
				break
			}

			//调用接口进行执行(结果将被放入chan Result)
			err := g.plugin.Run(ctx, wordCleaned, g.resultChan)
			if err != nil {
				//出现错误不退出
				g.errorChan <- err
			}

			//一定延迟后继续
			select {
			case <-ctx.Done():
			case <-time.After(g.Opts.Delay):
			}

		}
	}
}

func (g *Gobuster) getWordList() (*bufio.Scanner, error) {
	if g.Opts.Wordlist == "-" {
		// Read directly from stdin
		return bufio.NewScanner(os.Stdin), nil
	}
	// Pull content from the wordlist
	wordlist, err := os.Open(g.Opts.Wordlist)
	if err != nil {
		return nil, fmt.Errorf("failed to open wordlist: %w", err)
	}
	//统计文件行数,便于进度条实现
	lines, err := lineCounter(wordlist)
	if err != nil {
		return nil, fmt.Errorf("failed to get number of lines: %w", err)
	}
	g.RequestIssued = 0

	g.RequestExpected = lines
	if g.Opts.PatternFile != "" {
		g.RequestExpected += lines * len(g.Opts.Patterns)
	}
	g.RequestExpected *= g.plugin.RequestPerRun()

	//重置wordlist
	_, err = wordlist.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to rewind wordlist: %w", err)
	}
	return bufio.NewScanner(wordlist), nil
}
