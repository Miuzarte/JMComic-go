package JmComic

import "testing"

const testJmId = "JM1218574"

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
	resp, err := Search(t.Context(), "C99", "", 0)
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
	resp, err := GetChapter(t.Context(), "1218574") // 没有分章节, JmId 作唯一章节
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", *resp)
	// t.Logf("%s", resp.Raw)
}
