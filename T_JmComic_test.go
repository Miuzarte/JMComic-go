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
