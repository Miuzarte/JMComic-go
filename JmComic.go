package JmComic

import (
	"context"
	"fmt"
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

var (
	ImageUrl  = DEFAULT_IMAGE_URL
	UserAgent = DEFAULT_USER_AGENTS
	ApiHost   = constant.ApiHosts[0] // TODO: 轮换
)

func GetServer(ctx context.Context) (*Server, error) {
	const url = "https://rup4a04-c02.tos-cn-hongkong.bytepluses.com/newsvr-2025.txt"
	const domainSecret = "diosfjckwpqpdfjkvnqQjsik"
	// 直接返回加密内容

	b, err := Get(ctx, url)
	if err != nil {
		return nil, err
	}
	data, err := decrypt(b, []byte(domainSecret))
	if err != nil {
		return nil, err
	}

	return unmarshalTo[Server](data)
}

func GetSetting(ctx context.Context) (*Setting, error) {
	// url := fmt.Sprintf("https://%s%s?app_img_shunt=%d&express=", ApiHost, API_SETTING, settings.ImageStream)
	url := fmt.Sprintf("https://%s%s", ApiHost, API_SETTING)

	data, err := GetApi(ctx, url)
	if err != nil {
		return nil, err
	}

	return unmarshalTo[Setting](data)
}

func Search(ctx context.Context, keyword string, order string, page int) (*SearchResp, error) {
	if order == "" {
		order = "mr" // 默认按最新排序
	}

	u := fmt.Sprintf("https://%s%s", ApiHost, API_SEARCH)

	url, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	q := url.Query()
	q.Add("search_query", keyword)
	q.Add("o", order) // 默认使用 'mr' (最新) 排序
	if page > 1 {
		q.Add("page", strconv.Itoa(page))
	}
	url.RawQuery = q.Encode()

	data, err := GetApi(ctx, url.String())
	if err != nil {
		return nil, err
	}

	return unmarshalTo[SearchResp](data)
}

func GetAlbum(ctx context.Context, comicId string) (*Album, error) {
	comicId = strings.TrimPrefix(strings.ToLower(comicId), "jm")

	url := fmt.Sprintf("https://%s%s?id=%s", ApiHost, API_ALBUM, comicId)

	data, err := GetApi(ctx, url)
	if err != nil {
		return nil, err
	}

	return unmarshalTo[Album](data)
}

func GetChapter(ctx context.Context, chapterId string) (*Chapter, error) {
	chapterId = strings.TrimPrefix(strings.ToLower(chapterId), "jm")

	url := fmt.Sprintf("https://%s%s?id=%s", ApiHost, API_CHAPTER, chapterId)

	data, err := GetApi(ctx, url)
	if err != nil {
		return nil, err
	}

	return unmarshalTo[Chapter](data)
}
