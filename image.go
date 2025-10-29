package JmComic

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
	"strconv"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	"golang.org/x/image/webp"
)

// CalcNumParts 计算混淆分块数
func CalcNumParts(chapterId int, imageName string) (numParts int) {
	var modulus byte = 0

	switch {
	case chapterId < 220980:
		return 0
	case chapterId < 268850:
		return 10
	case chapterId > 421926:
		modulus = 8
	default:
		modulus = 10
	}

	hash := md5.Sum([]byte(strconv.Itoa(chapterId) + imageName))
	hashHex := make([]byte, hex.EncodedLen(len(hash)))
	hex.Encode(hashHex, hash[:])
	remainder := hashHex[len(hashHex)-1] % modulus
	return int(remainder)*2 + 2
}

var (
	jpegOption = jpeg.Options{Quality: 95}
	webpOption = nativewebp.Options{UseExtendedFormat: false}
)

// DescrambleImage 反混淆图片
func DescrambleImage(imgData []byte, num int) (_ []byte, err error) {
	if num <= 1 {
		return imgData, nil
	}

	var img image.Image
	ct := http.DetectContentType(imgData)
	switch ct {
	case "image/jpeg":
		img, err = jpeg.Decode(bytes.NewReader(imgData))
	case "image/png":
		img, err = png.Decode(bytes.NewReader(imgData))
	case "image/webp":
		img, err = webp.Decode(bytes.NewReader(imgData))
	case "application/octet-stream": // fallback
		return nil, fmt.Errorf("failed to decode image: [%X %X %X %X]", imgData[0], imgData[1], imgData[2], imgData[3])
	default:
		return nil, fmt.Errorf("unexpected image type: %s", ct)
	}
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	blockSize := height / num
	remainder := height % num

	type block struct {
		img image.Image
		h   int
	}
	blocks := make([]block, 0, num)
	currentY := 0
	for i := range num {
		h := blockSize
		if i < remainder {
			h++
		}
		// 创建目标块并从原图复制对应区域
		dstBlock := image.NewRGBA(image.Rect(0, 0, width, h))
		srcPoint := image.Point{X: 0, Y: currentY}
		draw.Draw(dstBlock, dstBlock.Bounds(), img, srcPoint, draw.Src)

		blocks = append(blocks, block{img: dstBlock, h: h})
		currentY += h
	}

	// 反向拼接
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	pasteY := 0
	for i := len(blocks) - 1; i >= 0; i-- {
		b := blocks[i]
		r := image.Rect(0, pasteY, width, pasteY+b.h)
		draw.Draw(newImg, r, b.img, image.Point{X: 0, Y: 0}, draw.Src)
		pasteY += b.h
	}

	var buf bytes.Buffer
	switch ct {
	case "image/jpeg":
		err = jpeg.Encode(&buf, newImg, &jpegOption)
	case "image/png":
		err = png.Encode(&buf, newImg)
	case "image/webp":
		err = nativewebp.Encode(&buf, newImg, &webpOption)
	default:
		panic("unreachable")
	}
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DownloadImage 下载图片并反混淆
//
// TODO: 并发与生成器
func DownloadImage(ctx context.Context, chapterId int, imageName string) ([]byte, error) {
	imgUrl := BuildImageUrl(chapterId, imageName)
	imgData, err := Get(ctx, imgUrl)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(imageName, ".gif") {
		// no de-scrambling needed for gif
		return imgData, nil
	}
	numParts := CalcNumParts(chapterId, imageName)
	if numParts > 1 {
		return DescrambleImage(imgData, numParts)
	}
	// no de-scrambling needed
	return imgData, nil
}

func DownloadCover(ctx context.Context, comicId int) ([]byte, error) {
	imgUrl := BuildCoverUrl(comicId)
	return Get(ctx, imgUrl)
}
