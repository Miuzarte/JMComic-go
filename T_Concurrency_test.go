package JmComic

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"
)

// TestConcurrentSharedState 并发读写包级共享状态, run with -race
//
// 覆盖的并发热点:
//   - apiHosts / imageHosts: 下载 goroutine 读, 失败时轮换写
//   - lastSyncTry: 多个请求同时失败时进 maybeSyncServers
//   - threads / useEnvProxy: 配置热重载写, 在飞的请求读
func TestConcurrentSharedState(t *testing.T) {
	// 已取消的 ctx 让 syncServers 里的请求立即失败, 不真联网
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	hostMu.Lock()
	savedApi := slices.Clone(apiHosts)
	savedImg := slices.Clone(imageHosts)
	// 头两个塞成必然失败的, 这样轮换分支每次都会被走到
	apiHosts = append([]string{"a.invalid"}, savedApi...)
	imageHosts = append([]string{"https://b.invalid"}, savedImg...)
	hostMu.Unlock()

	t.Cleanup(func() {
		hostMu.Lock()
		apiHosts, imageHosts = savedApi, savedImg
		hostMu.Unlock()
		SetThreads(defaultThreads)
		SetUseEnvProxy(true)
	})

	const rounds = 300
	var wg sync.WaitGroup

	// 写: 轮换域名 / 图床
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range rounds {
			markApiHostBad("a.invalid")
		}
	}()
	go func() {
		defer wg.Done()
		for range rounds {
			markImageHostBad("https://b.invalid")
		}
	}()

	// 读: 下载 goroutine 实际会走的那些读取路径
	//
	// newLimiter 与 Transport.Proxy 都放在读侧, 才能和下面的 SetThreads /
	// SetUseEnvProxy 形成真正的跨 goroutine 读写
	for range 4 {
		wg.Go(func() {
			req, err := http.NewRequest(http.MethodGet, "http://example.invalid/x", nil)
			if err != nil {
				t.Error(err)
				return
			}
			transport := httpClient.Transport.(*http.Transport)
			for range rounds {
				_ = apiHostsSnapshot()
				_ = imageHostsSnapshot()
				_ = GetApiHost()
				_ = GetImageHost()
				_ = BuildImageHeaders()
				_ = BuildCoverUrl(testJmId)
				_ = BuildImageUrl(testJmId, "00001.webp")
				_ = cap(newLimiter().sem)
				_, _ = transport.Proxy(req)
			}
		})
	}

	// 写: 配置热重载
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range rounds {
			SetThreads(i%16 + 1)
		}
	}()
	go func() {
		defer wg.Done()
		for i := range rounds {
			SetUseEnvProxy(i%2 == 0)
			_ = BuildApiHeaders(time.Now())
		}
	}()

	// 读写: 域名刷新 (含 lastSyncTry 的 check-then-set)
	for range 4 {
		wg.Go(func() {
			for range rounds {
				maybeSyncServers(ctx)
			}
		})
	}

	wg.Wait()
}
