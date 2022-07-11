// Package cli 提供程序的执行,主要的逻辑依赖接口,便于后续拓展不同的模式
package cli

import (
	"buster/lib"
	"context"
)

func GoBuster(ctx context.Context, opts *lib.Options, plugin lib.GobusterPlugin) error {
	//TODO:
	return nil
}
