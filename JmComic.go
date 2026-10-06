package JmComic

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Miuzarte/JMComic-go/internal/constant"
)

const (
	DEFAULT_IMAGE_URL   = "https://cdn-msp.jmapinodeudzn.net"
	DEFAULT_USER_AGENTS = "Mozilla/5.0 (Linux; Android 10; K; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/154.0.0.0 Mobile Safari/537.36"
	VERSION             = "2.1.7"
)

// NEW_SERVER_URLS 获取最新移动端 API 域名的地址
var NEW_SERVER_URLS = [...]string{
	"https://rup4a04-c01.tos-ap-southeast-1.bytepluses.com/newsvr-2025.txt",
	"https://rup4a04-c02.tos-cn-hongkong.bytepluses.com/newsvr-2025.txt",
	"https://rup4a04-c03.tos-cn-beijing.bytepluses.com.cn/newsvr-2025.txt",
}

const (
	API_SETTING = "/setting"
	API_SEARCH  = "/search"
	API_ALBUM   = "/album"
	API_CHAPTER = "/chapter"
)

const (
	// API_SECRET_REQ  = "18comicAPPContent" // 只有 /chapter_view_template 接口用 "18comicAPPContent"
	API_SECRET_REQ  = "185Hcomic3PAPP7R" // combine with timestamp
	API_SECRET_RESP = "185Hcomic3PAPP7R" // combine with timestamp
	SVR_SECRET      = "diosfjckwpqpdfjkvnqQjsik"
)

var svrSecret = md5.Sum([]byte(SVR_SECRET))

var (
	UserAgent = DEFAULT_USER_AGENTS

	hostMu     sync.RWMutex
	apiHosts   = slices.Clone(constant.ApiHosts[:])
	imageHosts = slices.Clone(constant.ImageHosts[:])

	syncMu      sync.Mutex
	lastSyncTry time.Time
)

// syncInterval 两次拉取域名的最小间隔
const syncInterval = 10 * time.Minute

// defaultThreads 默认下载并发数
const defaultThreads = 4

// threads 下载并发数
var threads atomic.Int64

// SetThreads 设置下载并发数
func SetThreads(n int) {
	if n <= 0 {
		n = 1
	}
	threads.Store(int64(n))
}

// threadCount 取当前并发数
func threadCount() int {
	if n := int(threads.Load()); n > 0 {
		return n
	}
	return defaultThreads
}

// SetUseEnvProxy 设置是否使用系统环境变量中的代理
//
// 默认为 true
func SetUseEnvProxy(b bool) {
	useEnvProxy.Store(b)
}

// GetApiHost 当前优先使用的 API 域名
func GetApiHost() string {
	hostMu.RLock()
	defer hostMu.RUnlock()
	return apiHosts[0]
}

// GetImageHost 当前优先使用的图床
func GetImageHost() string {
	hostMu.RLock()
	defer hostMu.RUnlock()
	return imageHosts[0]
}

func apiHostsSnapshot() []string {
	hostMu.RLock()
	defer hostMu.RUnlock()
	return slices.Clone(apiHosts)
}

func imageHostsSnapshot() []string {
	hostMu.RLock()
	defer hostMu.RUnlock()
	return slices.Clone(imageHosts)
}

// markApiHostBad 把出问题的域名挪到末尾
func markApiHostBad(host string) {
	hostMu.Lock()
	defer hostMu.Unlock()
	if len(apiHosts) > 1 && apiHosts[0] == host {
		apiHosts = append(apiHosts[1:], host)
	}
}

// markImageHostBad 把出问题的图床挪到末尾
func markImageHostBad(host string) {
	hostMu.Lock()
	defer hostMu.Unlock()
	if len(imageHosts) > 1 && imageHosts[0] == host {
		imageHosts = append(imageHosts[1:], host)
	}
}

// normalizeHost 补全 scheme 并去掉结尾的 /
func normalizeHost(host string) string {
	host = strings.TrimSuffix(strings.TrimSpace(host), "/")
	if host == "" {
		return ""
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	return host
}

// SyncServers 从官方域名服务器拉取最新的 API 域名与图床
//
// 全部镜像都失败时保留原有域名并返回错误
func SyncServers(ctx context.Context) error {
	syncMu.Lock()
	defer syncMu.Unlock()
	return syncServers(ctx)
}

// maybeSyncServers 限频地刷新域名, 内置域名全试挂了才会走到这里
func maybeSyncServers(ctx context.Context) {
	syncMu.Lock()
	defer syncMu.Unlock()
	if time.Since(lastSyncTry) < syncInterval {
		return
	}
	lastSyncTry = time.Now()
	_ = syncServers(ctx)
}

func syncServers(ctx context.Context) error {
	var errs []error
	for _, u := range NEW_SERVER_URLS {
		srv, err := getServer(ctx, u)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", u, err))
			continue
		}
		hosts := slices.DeleteFunc(slices.Clone(srv.Server), func(s string) bool { return s == "" })
		if len(hosts) == 0 {
			errs = append(errs, fmt.Errorf("%s: empty server list", u))
			continue
		}

		hostMu.Lock()
		apiHosts = hosts
		hostMu.Unlock()

		// 顺手用新域名取一次图床
		if st, err := getSetting(ctx, hosts[0]); err == nil {
			if h := normalizeHost(st.ImgHost); h != "" {
				hostMu.Lock()
				imageHosts = append([]string{h}, slices.DeleteFunc(imageHosts, func(s string) bool { return s == h })...)
				hostMu.Unlock()
			}
		}
		return nil
	}
	return errors.Join(errs...)
}

func GetServer(ctx context.Context) (*Server, error) {
	return getServer(ctx, NEW_SERVER_URLS[0])
}

func getServer(ctx context.Context, serverUrl string) (*Server, error) {
	b, _, err := Get(ctx, serverUrl)
	if err != nil {
		return nil, err
	}
	return unmarshalTo[Server](decrypt(b, svrSecret[:]))
}

// apiRequest 依次尝试各个 API 域名, 全失败后刷新域名再试一轮
func apiRequest(ctx context.Context, apiPath string, params map[string]string) ([]byte, error) {
	var errs []error

	// try 返回是否成功; ctx 已取消则不必再试其他域名
	try := func() ([]byte, bool) {
		for _, host := range apiHostsSnapshot() {
			b, err := GetApi(ctx, buildRequest(host, apiPath, params))
			if err == nil {
				return b, true
			}
			errs = append(errs, fmt.Errorf("%s: %w", host, err))
			markApiHostBad(host)
			if ctx.Err() != nil {
				return nil, false
			}
		}
		return nil, false
	}

	if b, ok := try(); ok {
		return b, nil
	}
	if ctx.Err() != nil {
		return nil, errors.Join(errs...)
	}

	// 内置域名可能已经过期
	maybeSyncServers(ctx)
	if b, ok := try(); ok {
		return b, nil
	}
	return nil, errors.Join(errs...)
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
	return unmarshalTo[Setting](apiRequest(ctx, API_SETTING, nil))
}

// getSetting 只请求指定域名, 供 [syncServers] 内部使用, 避免递归刷新域名
func getSetting(ctx context.Context, host string) (*Setting, error) {
	return unmarshalTo[Setting](GetApi(ctx, buildRequest(host, API_SETTING, nil)))
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

	return unmarshalTo[SearchResp](apiRequest(ctx, API_SEARCH, params))
}

func GetAlbum(ctx context.Context, comicId int) (*Album, error) {
	return unmarshalTo[Album](apiRequest(ctx, API_ALBUM, map[string]string{"id": strconv.Itoa(comicId)}))
}

func GetChapter(ctx context.Context, chapterId int) (*Chapter, error) {
	return unmarshalTo[Chapter](apiRequest(ctx, API_CHAPTER, map[string]string{"id": strconv.Itoa(chapterId)}))
}

func DownloadCoversIter(ctx context.Context, search *SearchResp) iter.Seq2[Image, error] {
	return newDownloader(ctx, newCoverDownload(search)).downloadIter()
}

func DownloadComicIter(ctx context.Context, chapter *Chapter) iter.Seq2[Image, error] {
	return newDownloader(ctx, newImageDownload(chapter.Id, chapter.Images)).downloadIter()
}
