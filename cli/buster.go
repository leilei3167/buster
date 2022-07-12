// Package cli 提供程序的执行,主要的逻辑依赖接口,便于后续拓展不同的模式
package cli

import (
	"buster/lib"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const ruler = "==============================================================="
const cliProgressUpdate = 500 * time.Millisecond

func banner() {
	fmt.Printf("Gobuster v%s\n", lib.VERSION)
	fmt.Println("by OJ Reeves (@TheColonial) & Christian Mehlmauer (@firefart)")
}

type outputType struct {
	Mu              sync.RWMutex //是否需要用地址?
	MaxCharsWritten int
}

// GoBuster 适用解析好的配置,以及配置好的plugin实例,开始执行任务
func GoBuster(ctx context.Context, opts *lib.Options, plugin lib.GobusterPlugin) error {
	//用于控制此函数开始之后的调用链
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	//根据配置项,插件构建爆破执行实例
	gobuster, err := lib.NewGobuster(opts, plugin)
	if err != nil {
		return err
	}

	//分别开启各个处理阶段,开启工作流
	var wg sync.WaitGroup
	var o = new(outputType)

	wg.Add(1)
	go resultWorker(gobuster, opts.OutputFilename, &wg, o)

	wg.Add(1)
	go errorWorker(gobuster, &wg, o)

	/*//是否开启进度条
	if !opts.Quiet && !opts.NoProgress {
		wg.Add(1)
		//TODO:processbar
	}*/

	//fan-out
	err = gobuster.Run(ctxCancel)

	cancel()
	wg.Wait() //等待所有的工作完毕
	if err != nil {
		return err
	}

	if !opts.Quiet {
		// clear stderr progress
		fmt.Fprintf(os.Stderr, "\r%s\n", rightPad("", " ", o.MaxCharsWritten))
		fmt.Println(ruler)
		gobuster.LogInfo.Println("Finished")
		fmt.Println(ruler)
	}

	return nil
}
func rightPad(s string, padStr string, overallen int) string {
	strLen := len(s)
	if overallen <= strLen {
		return s
	}
	toPad := overallen - strLen - 1
	pad := strings.Repeat(padStr, toPad)
	return fmt.Sprintf("%s%s", s, pad)
}
func writeToFile(f *os.File, s string) error {
	_, err := f.WriteString(fmt.Sprintf("%s\n", s))
	if err != nil {
		return fmt.Errorf("[!] Unable to write to file %w", err)
	}
	return nil
}

func resultWorker(g *lib.Gobuster, filename string, wg *sync.WaitGroup, output *outputType) {
	defer wg.Done()
	var f *os.File
	var err error
	if filename != "" {
		f, err = os.Create(filename)
		if err != nil {
			g.LogError.Fatalf("error on creating output file:%v", err)
		}
		defer f.Close()
	}
	//调用接口的Results方法,获取结果通道并range获得每一个result接口值
	for r := range g.Results() {
		s, err := r.ResulToString()
		if err != nil {
			g.LogError.Fatal(err)
		}
		if s != "" {
			s = strings.TrimSpace(s)
			output.Mu.Lock()
			w, _ := fmt.Printf("\r%s\n", rightPad(s, " ", output.MaxCharsWritten))
			if (w - 1) > output.MaxCharsWritten {
				output.MaxCharsWritten = w - 1
			}
			output.Mu.Unlock()
			if f != nil {
				err = writeToFile(f, s)
				if err != nil {
					g.LogError.Fatalf("error on writing output file:%v", err)
				}
			}
		}
	}
}

func errorWorker(g *lib.Gobuster, wg *sync.WaitGroup, output *outputType) {
	defer wg.Done()

	for e := range g.Errors() {
		if !g.Opts.Quiet && !g.Opts.NoError { //Quiet和error同时不为false时打印错误
			output.Mu.Lock()
			g.LogError.Printf("[!] %v", e)
			output.Mu.Unlock()
		}
	}

}
