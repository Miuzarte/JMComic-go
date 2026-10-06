package JmComic

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/deepteams/webp"
)

// CalcNumParts 计算混淆分块数
func CalcNumParts(chapterId int, imageName string) (numParts int) {
	dotIndex := strings.Index(imageName, ".")
	if dotIndex == -1 {
		return 0
	}
	imageName = imageName[:dotIndex]

	var modulus byte = 0

	switch {
	case chapterId < 220980:
		return 0
	case chapterId < 268850:
		return 10
	case chapterId < 421926:
		modulus = 10
	default:
		modulus = 8
	}

	hash := md5.Sum([]byte(strconv.Itoa(chapterId) + imageName))
	hashHex := make([]byte, hex.EncodedLen(len(hash)))
	hex.Encode(hashHex, hash[:])
	remainder := hashHex[len(hashHex)-1] % modulus
	return int(remainder)*2 + 2
}

var (
	jpegOption = jpeg.Options{Quality: 95}
	webpOption = webp.EncoderOptions{
		Quality: 85, // 图源本身就是有损 webp
		Method:  4,
	}
)

// DescrambleImage 反混淆图片
//
// 分块规则与官方客户端一致: 源图最底部一块吸收 height%num 的余数,
// 再自下而上倒序拼接, 参考 _py/eighteencomic.py 的 SegmentationPicture
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
	case "application/x-gzip":
		// 已在 [httpClient] 与 [constant.Header] 处请求不压缩
		var decompressed []byte
		decompressed, err = gunzip(imgData)
		if err != nil {
			return nil, err
		}
		return DescrambleImage(decompressed, num)
	default:
		return nil, fmt.Errorf("unexpected image type: %s", ct)
	}
	if err != nil {
		return nil, err
	}

	descrambled := reverseBlocks(img, num)

	var buf bytes.Buffer
	switch ct {
	case "image/jpeg":
		err = jpeg.Encode(&buf, descrambled, &jpegOption)
	case "image/png":
		err = png.Encode(&buf, descrambled)
	case "image/webp":
		err = webp.Encode(&buf, descrambled, &webpOption)
	default:
		return nil, fmt.Errorf("unexpected image type: %s", ct)
	}
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// reverseBlocks 把图片按 num 分块后倒序拼接
//
// 设 c = height/num, over = height%num,
// 源图最底下一块高度为 c+over, 其余为 c,
// 依次拼到输出顶部, 这样既不丢像素也不留透明行
func reverseBlocks(src image.Image, num int) *image.NRGBA {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	dst := image.NewNRGBA(image.Rect(0, 0, width, height))

	c, over := height/num, height%num
	for i := range num {
		h := c
		ySrc := height - c*(i+1) - over
		yDst := c * i
		if i == 0 {
			h += over // 源图最底下一块, 多出余数
		} else {
			yDst += over
		}
		draw.Draw(dst,
			image.Rect(0, yDst, width, yDst+h),
			src, image.Pt(bounds.Min.X, bounds.Min.Y+ySrc), draw.Src)
	}

	return dst
}

func gunzip(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip new reader: %w", err)
	}
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	return decompressed, nil
}

func DownloadCover(ctx context.Context, comicId int) ([]byte, error) {
	imgUrl := BuildCoverUrl(comicId)
	b, _, err := Get(ctx, imgUrl)
	return b, err
}
