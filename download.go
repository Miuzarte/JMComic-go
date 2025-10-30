package JmComic

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"iter"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/webp"
)

var ErrUnknownImageType = fmt.Errorf("unknown image type")

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

func ParseImageType(s string) ImageType {
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

func ParseImageMimeType(mimeType string) ImageType {
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
	ChapterId           int
	Name                string // "00001.webp"
	P                   int
	Data                []byte
	Type                ImageType
	IsDescrambledNeeded bool
	// IsFromCache bool // TODO(maybe
}

func (i *Image) String() string {
	return i.Name
}

func (i *Image) Decode() (image.Image, error) {
	switch i.Type {
	case IMAGE_TYPE_WEBP:
		return webp.Decode(bytes.NewReader(i.Data))
	case IMAGE_TYPE_JPEG:
		return jpeg.Decode(bytes.NewReader(i.Data))
	case IMAGE_TYPE_PNG:
		return png.Decode(bytes.NewReader(i.Data))
	case IMAGE_TYPE_GIF:
		return gif.Decode(bytes.NewReader(i.Data))
	default:
		img, typ := tryDecodeImage(i.Data)
		if typ != IMAGE_TYPE_UNKNOWN {
			i.Type = typ
			return img, nil
		}
	}
	return nil, wrapErr(ErrUnknownImageType, http.DetectContentType(i.Data))
}

type download struct {
	img *Image
	err chan error
	// cache *cacheComic // TODO(maybe
}

func (d *download) start(ctx context.Context) {
	d.err <- downloadAndDescrambleImage(ctx, d.img)
}

const DOWNLOAD_TYPE_COVER = "<COVER>"

func newCoverDownload(search *SearchResp) (dls []*download) {
	dls = make([]*download, len(search.Content))
	for i := range search.Content {
		id, err := strconv.Atoi(search.Content[i].Id)
		if err != nil {
			panic("[TODO] handle non-numeric id")
		}
		dls[i] = &download{
			img: &Image{
				ChapterId: id,
				Name:      DOWNLOAD_TYPE_COVER,
				P:         i + 1,
			},
			err: make(chan error, 1),
		}
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
		defer limiter.close()

		for _, item := range dl.items {
			select {
			case <-dl.ctx.Done():
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

// downloadAndDescrambleImage 下载图片并反混淆
func downloadAndDescrambleImage(ctx context.Context, img *Image) error {
	imgUrl := ""
	if img.Name != DOWNLOAD_TYPE_COVER {
		imgUrl = BuildImageUrl(img.ChapterId, img.Name)
	} else {
		imgUrl = BuildCoverUrl(img.ChapterId)
	}

	imgData, resp, err := Get(ctx, imgUrl)
	if err != nil {
		return err
	}

	img.Data = imgData

	nct := ""
	hct := resp.Header.Get("Content-Type")
	// dct := http.DetectContentType(imgData)
	if inSplits := strings.Split(img.Name, "."); len(inSplits) >= 2 {
		nct = inSplits[len(inSplits)-1]
		if "image/"+nct != hct {
			return fmt.Errorf("\"image/\"+nct (%s) != hct (%s)", nct, hct)
		}
	} else if img.Name == DOWNLOAD_TYPE_COVER {
		nct = DOWNLOAD_TYPE_COVER
	}

	img.Type = ParseImageMimeType(hct)

	switch nct {
	case "gif", DOWNLOAD_TYPE_COVER:
		// no de-scrambling needed
		return nil
	}

	numParts := CalcNumParts(img.ChapterId, img.Name)
	if numParts > 1 {
		data, err := DescrambleImage(img.Data, numParts)
		if err != nil {
			img.IsDescrambledNeeded = true
			return err
		}
		img.IsDescrambledNeeded = false
		img.Data = data
	}
	return nil
}

func tryDecodeImage(data []byte) (img image.Image, _ ImageType) {
	var err error
	switch http.DetectContentType(data) {
	case "image/webp":
		img, err = webp.Decode(bytes.NewReader(data))
		if err == nil {
			return img, IMAGE_TYPE_WEBP
		}
	case "image/jpeg":
		img, err = jpeg.Decode(bytes.NewReader(data))
		if err == nil {
			return img, IMAGE_TYPE_JPEG
		}
	case "image/png":
		img, err = png.Decode(bytes.NewReader(data))
		if err == nil {
			return img, IMAGE_TYPE_PNG
		}
	case "image/gif":
		img, err = gif.Decode(bytes.NewReader(data))
		if err == nil {
			return img, IMAGE_TYPE_GIF
		}
	}
	return nil, IMAGE_TYPE_UNKNOWN
}

type limiter struct {
	sem  chan struct{}
	once sync.Once
}

func newLimiter() *limiter {
	return &limiter{
		sem: make(chan struct{}, threads),
	}
}

func (l *limiter) acquire() chan<- struct{} {
	return l.sem
}

func (l *limiter) release() {
	<-l.sem
}

func (l *limiter) close() {
	l.once.Do(func() {
		close(l.sem)
	})
}
