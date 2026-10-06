package JmComic

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strconv"
	"strings"
)

type ImageType int

const (
	IMAGE_TYPE_UNKNOWN ImageType = iota

	IMAGE_TYPE_WEBP
	IMAGE_TYPE_JPEG
	IMAGE_TYPE_PNG
	IMAGE_TYPE_GIF
)

func (it ImageType) String() string {
	switch it {
	case IMAGE_TYPE_UNKNOWN:
		return "unknown"
	case IMAGE_TYPE_WEBP:
		return "webp"
	case IMAGE_TYPE_JPEG:
		return "jpeg"
	case IMAGE_TYPE_PNG:
		return "png"
	case IMAGE_TYPE_GIF:
		return "gif"
	default:
		return ""
	}
}

func parseImageType(s string) ImageType {
	switch strings.ToLower(s) {
	case "webp":
		return IMAGE_TYPE_WEBP
	case "jpeg", "jpg":
		return IMAGE_TYPE_JPEG
	case "png":
		return IMAGE_TYPE_PNG
	case "gif":
		return IMAGE_TYPE_GIF
	default:
		return IMAGE_TYPE_UNKNOWN
	}
}

func parseImageMimeType(mimeType string) ImageType {
	switch strings.ToLower(mimeType) {
	case "image/webp":
		return IMAGE_TYPE_WEBP
	case "image/jpeg":
		return IMAGE_TYPE_JPEG
	case "image/png":
		return IMAGE_TYPE_PNG
	case "image/gif":
		return IMAGE_TYPE_GIF
	default:
		return IMAGE_TYPE_UNKNOWN
	}
}

type Image struct {
	ChapterId           int    // 反混淆用
	Name                string // "00001.webp"
	P                   int
	Data                []byte
	Type                ImageType
	IsDescrambledNeeded bool
	// IsFromCache bool // TODO(maybe
}

func (i *Image) String() string {
	return i.Name + ": " + strconv.Itoa(len(i.Data))
}

type download struct {
	img *Image
	err chan error
	// preErr 构造阶段就失败 (如 id 不是数字), 不再发请求
	preErr error
	// cache *cacheComic // TODO(maybe
}

func (d *download) start(ctx context.Context) {
	if d.preErr != nil {
		d.err <- d.preErr
		return
	}
	d.err <- downloadAndDescrambleImage(ctx, d.img)
}

const DOWNLOAD_TYPE_COVER = "<COVER>"

func newCoverDownload(search *SearchResp) (dls []*download) {
	dls = make([]*download, len(search.Content))
	for i := range search.Content {
		d := &download{
			img: &Image{
				Name: DOWNLOAD_TYPE_COVER,
				// P: i + 1, // 封面不是章节里的页, 不编页码
			},
			err: make(chan error, 1),
		}
		id, err := strconv.Atoi(search.Content[i].Id)
		if err != nil {
			d.preErr = fmt.Errorf("invalid comic id %q: %w", search.Content[i].Id, err)
		} else {
			d.img.ChapterId = id
		}
		dls[i] = d
	}
	return dls
}

func newImageDownload(chapterId int, imgNames []string) (dls []*download) {
	dls = make([]*download, len(imgNames))
	for i := range imgNames {
		dls[i] = &download{
			img: &Image{
				ChapterId: chapterId,
				Name:      imgNames[i],
				P:         i + 1,
			},
			err: make(chan error, 1),
		}
	}
	return dls
}

type downloader struct {
	ctx    context.Context
	cancel context.CancelFunc
	items  []*download
}

func newDownloader(ctx context.Context, dls []*download) *downloader {
	ctx, cancel := context.WithCancel(ctx)
	return &downloader{
		ctx:    ctx,
		cancel: cancel,
		items:  dls,
	}
}

func (dl *downloader) startBackground() {
	go func() {
		limiter := newLimiter()

		for i, item := range dl.items {
			select {
			case <-dl.ctx.Done():
				// 消费端按下标顺序读 err chan,
				// 没派发出去的 item 必须补上错误, 否则永久阻塞
				for _, rest := range dl.items[i:] {
					rest.err <- dl.ctx.Err()
				}
				return
			case limiter.acquire() <- struct{}{}:
			}

			go func() {
				defer limiter.release()
				item.start(dl.ctx)
			}()
		}
	}()
}

func (dl *downloader) downloadIter() iter.Seq2[Image, error] {
	dl.startBackground()
	return func(yield func(Image, error) bool) {
		defer dl.cancel()
		for _, item := range dl.items {
			if !yield(*item.img, <-item.err) {
				return
			}
		}
	}
}

// imageTries 单个图床的重试次数
const imageTries = 2

// fetchImage 依次尝试各个图床, 每个图床重试 [imageTries] 次
func fetchImage(ctx context.Context, img *Image) (_ []byte, contentType string, err error) {
	var errs []error

	for _, host := range imageHostsSnapshot() {
		imgUrl := buildImageUrlWith(host, img.ChapterId, img.Name)
		if img.Name == DOWNLOAD_TYPE_COVER {
			imgUrl = buildCoverUrlWith(host, img.ChapterId)
		}

		for try := 1; try <= imageTries; try++ {
			var (
				data []byte
				resp *http.Response
			)
			data, resp, err = Get(ctx, imgUrl)
			if err == nil {
				switch ct := resp.Header.Get("Content-Type"); {
				case len(data) == 0:
					err = errors.New("empty body")
				case !strings.HasPrefix(ct, "image/"):
					err = fmt.Errorf("unexpected content type: %s", ct)
				default:
					return data, ct, nil
				}
			}

			errs = append(errs, fmt.Errorf("%s (try %d): %w", host, try, err))
			if ctx.Err() != nil {
				return nil, "", errors.Join(errs...)
			}
		}

		markImageHostBad(host)
	}

	return nil, "", errors.Join(errs...)
}

// downloadAndDescrambleImage 下载图片并反混淆
func downloadAndDescrambleImage(ctx context.Context, img *Image) error {
	imgData, contentType, err := fetchImage(ctx, img)
	if err != nil {
		return err
	}

	img.Data = imgData

	mimeType := strings.TrimSpace(strings.Split(contentType, ";")[0])
	img.Type = parseImageMimeType(mimeType)
	if img.Type == IMAGE_TYPE_UNKNOWN {
		if dotIndex := strings.LastIndex(img.Name, "."); dotIndex != -1 {
			img.Type = parseImageType(img.Name[dotIndex+1:])
		}
	}

	if img.Type == IMAGE_TYPE_GIF || img.Name == DOWNLOAD_TYPE_COVER {
		return nil
	}

	numParts := CalcNumParts(img.ChapterId, img.Name)
	if numParts <= 1 {
		return nil
	}

	data, err := DescrambleImage(img.Data, numParts)
	if err != nil {
		img.IsDescrambledNeeded = true
		return err
	}
	img.IsDescrambledNeeded = false
	img.Data = data
	return nil
}

type limiter struct {
	sem chan struct{}
}

func newLimiter() *limiter {
	return &limiter{
		sem: make(chan struct{}, threadCount()),
	}
}

func (l *limiter) acquire() chan<- struct{} {
	return l.sem
}

func (l *limiter) release() {
	<-l.sem
}
