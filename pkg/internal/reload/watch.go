package reload

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/knadh/koanf/providers/file"
)

var mu sync.Mutex
var debounceTimer *time.Timer

// debounceDuration 是防抖的时间窗口
const debounceDuration = 100 * time.Millisecond

// Watch watches the given path for changes and calls the given callback.
func Watch(ctx context.Context, path string, cb func() error) error {
	if ctx.Err() != nil {
		return nil
	}
	log := logr.FromContextOrDiscard(ctx).WithValues("path", path)
	return file.Provider(path).Watch(func(_ any, err error) {
		// 如果context已经取消，则不再处理事件
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Info("failed watching config", "error", err)
			return
		}

		// 使用防抖机制，避免重复触发
		mu.Lock()
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(debounceDuration, func() {
			// 在debounce时间后执行配置重载
			mu.Lock()
			defer mu.Unlock()

			log.Info("auto-reloading config")
			start := time.Now()
			if err := cb(); err != nil {
				log.Info("failed to reload config", "error", err)
				return
			}
			log.Info("reloaded config successfully", "duration", time.Since(start).Round(time.Millisecond).String())
		})
		mu.Unlock()
	})
}
