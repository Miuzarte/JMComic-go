package JmComic

import (
	"context"
	"slices"
	"testing"
	"time"
)

// TestSyncServers 官方域名服务器应该能拉到可用域名
func TestSyncServers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := SyncServers(ctx); err != nil {
		t.Fatal(err)
	}
	if host := GetApiHost(); host == "" {
		t.Fatal("同步后 API 域名为空")
	}

	// 用同步到的域名应该能直接取数据
	if _, err := GetSetting(ctx); err != nil {
		t.Fatal(err)
	}
	if host := GetImageHost(); host == "" {
		t.Fatal("同步后图床为空")
	}
}

// TestApiRequestRotatesHosts 第一个域名挂掉时应该自动换下一个
func TestApiRequestRotatesHosts(t *testing.T) {
	hostMu.Lock()
	saved := slices.Clone(apiHosts)
	apiHosts = append([]string{"localhost"}, saved...) // localhost:443 连不上
	hostMu.Unlock()
	defer func() {
		hostMu.Lock()
		apiHosts = saved
		hostMu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := GetChapter(ctx, testJmId); err != nil {
		t.Fatal(err)
	}
	if GetApiHost() == "localhost" {
		t.Error("挂掉的图床未被替换")
	}
}

// TestFetchImageFallsBack 图床挂掉时应该自动换下一个
func TestFetchImageFallsBack(t *testing.T) {
	hostMu.Lock()
	saved := slices.Clone(imageHosts)
	imageHosts = append([]string{"https://localhost"}, saved...)
	hostMu.Unlock()
	defer func() {
		hostMu.Lock()
		imageHosts = saved
		hostMu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	chapter, err := GetChapter(ctx, testJmId)
	if err != nil {
		t.Fatal(err)
	}

	_, contentType, err := fetchImage(ctx, &Image{ChapterId: chapter.Id, Name: chapter.Images[0]})
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "image/webp" {
		t.Errorf("content type = %q, want image/webp", contentType)
	}
	if GetImageHost() == "https://localhost" {
		t.Error("挂掉的图床未被替换")
	}
}
