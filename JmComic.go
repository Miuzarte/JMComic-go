package JmComic

import (
	"context"
	"crypto/md5"
	"iter"
	"net/url"
	"strconv"
	"strings"

	"github.com/Miuzarte/JMComic-go/internal/constant"
)

const (
	DEFAULT_IMAGE_URL   = "https://cdn-msp.jmapinodeudzn.net"
	DEFAULT_USER_AGENTS = "Mozilla/5.0 (Linux; Android 10; K; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/130.0.0.0 Mobile Safari/537.36"
	VERSION             = "2.0.1"
)

const NEW_SERVER_URL = "https://rup4a04-c02.tos-cn-hongkong.bytepluses.com/newsvr-2025.txt"

const (
	API_SETTING = "/setting"
	API_SEARCH  = "/search"
	API_ALBUM   = "/album"
	API_CHAPTER = "/chapter"
)

const (
	API_SECRET_REQ  = "18comicAPPContent" // combine with timestamp
	API_SECRET_RESP = "185Hcomic3PAPP7R"  // combine with timestamp
	SVR_SECRET      = "diosfjckwpqpdfjkvnqQjsik"
)

var svrSecret = md5.Sum([]byte(SVR_SECRET))

var (
	ImageUrl  = DEFAULT_IMAGE_URL
	UserAgent = DEFAULT_USER_AGENTS
	ApiHost   = constant.ApiHosts[0] // TODO: 轮换
)

var threads = 4 // 下载并发数

// SetThreads 设置下载并发数
func SetThreads(n int) {
	if n <= 0 {
		n = 1
	}
	threads = n
}

func GetServer(ctx context.Context) (*Server, error) {
	b, _, err := Get(ctx, NEW_SERVER_URL)
	if err != nil {
		return nil, err
	}
	return unmarshalTo[Server](decrypt(b, svrSecret[:]))
}

func buildRequest(host string, apiPath string, params map[string]string) string {
	u, err := url.Parse(host)
	if err != nil {
		panic(err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if !strings.HasPrefix(apiPath, "/") {
		u.Path = "/"
	}
	u.Path += apiPath
	q := u.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func GetSetting(ctx context.Context) (*Setting, error) {
	return unmarshalTo[Setting](GetApi(ctx, buildRequest(ApiHost, API_SETTING, nil)))
}

func Search(ctx context.Context, keyword string, order string, page int) (*SearchResp, error) {
	if order == "" {
		order = "mr" // 默认按最新排序
	}
	params := map[string]string{
		"search_query": keyword,
		"o":            order,
	}
	if page > 1 {
		params["page"] = strconv.Itoa(page)
	}

	return unmarshalTo[SearchResp](GetApi(ctx, buildRequest(ApiHost, API_SEARCH, params)))
}

func GetAlbum(ctx context.Context, comicId int) (*Album, error) {
	return unmarshalTo[Album](GetApi(ctx, buildRequest(ApiHost, API_ALBUM, map[string]string{"id": strconv.Itoa(comicId)})))
}

func GetChapter(ctx context.Context, chapterId int) (*Chapter, error) {
	return unmarshalTo[Chapter](GetApi(ctx, buildRequest(ApiHost, API_CHAPTER, map[string]string{"id": strconv.Itoa(chapterId)})))
}

func DownloadCoversIter(ctx context.Context, search *SearchResp) iter.Seq2[Image, error] {
	return newDownloader(ctx, newCoverDownload(search)).downloadIter()
}

func DownloadComicIter(ctx context.Context, chapterId int) iter.Seq2[Image, error] {
	chapter, err := GetChapter(ctx, chapterId)
	if err != nil {
		return func(yield func(Image, error) bool) {
			yield(Image{}, err)
		}
	}
	if len(chapter.Images) == 0 {
		panic("[TODO] handle empty chapter images list")
	}
	return newDownloader(ctx, newImageDownload(chapterId, chapter.Images)).downloadIter()
}
