package JmComic

import (
	"fmt"
	"os"
	"testing"
)

const (
	testJmId       = 1026275
	testJmIdMulti1 = 519180
	testJmIdMulti2 = 521226
)

// TestSetThreads 并发数必须恒 >= 1
//
// 信号量容量为 0 时派发循环会在第一条上永久等待, 导致迭代器死锁
func TestSetThreads(t *testing.T) {
	t.Cleanup(func() { SetThreads(defaultThreads) })

	for _, n := range []int{0, -1, -100} {
		SetThreads(n)
		if got := threadCount(); got != 1 {
			t.Errorf("SetThreads(%d) 后并发数 = %d, want 1", n, got)
		}
	}

	SetThreads(8)
	if got := threadCount(); got != 8 {
		t.Errorf("SetThreads(8) 后并发数 = %d, want 8", got)
	}
	if got := cap(newLimiter().sem); got != 8 {
		t.Errorf("limiter 容量 = %d, want 8", got)
	}

	// 零值 (没调过 SetThreads) 也要兜底
	threads.Store(0)
	if got := cap(newLimiter().sem); got != defaultThreads {
		t.Errorf("并发数为 0 时 limiter 容量 = %d, want %d", got, defaultThreads)
	}
}

func TestGetServer(t *testing.T) {
	resp, err := GetServer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
	// t.Logf("%s", resp.Raw)
}

func TestGetSetting(t *testing.T) {
	resp, err := GetSetting(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
	// t.Logf("%s", resp.Raw)
}

func TestSearch(t *testing.T) {
	resp, err := Search(t.Context(), "C99", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
	// t.Logf("%s", resp.Raw)
}

func TestGetAlbum(t *testing.T) {
	resp, err := GetAlbum(t.Context(), testJmId)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
	// t.Logf("%s", resp.Raw)
}

func TestGetChapter(t *testing.T) {
	resp, err := GetChapter(t.Context(), testJmId) // 没有分章节, JmId 作唯一章节
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
	// t.Logf("%s", resp.Raw)
}

func TestDownloadComic(t *testing.T) {
	resp, err := GetChapter(t.Context(), testJmId)
	if err != nil {
		t.Fatal(err)
	}
	for img, err := range DownloadComicIter(t.Context(), resp) {
		if err != nil {
			t.Fatal(err)
		}
		f, e := os.Create(fmt.Sprintf("/home/miuzarte/git/JMComic-go/_download/%s", img.Name))
		if e != nil {
			t.Fatal(e)
		}
		f.Write(img.Data)
		f.Close()
		t.Logf("%s: %d", img.Name, len(img.Data))
		break
	}
}
