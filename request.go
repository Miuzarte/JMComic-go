package JmComic

import (
	"bytes"
	"context"
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Miuzarte/JMComic-go/internal/constant"
)

func BuildApiHeaders(t time.Time) map[string]string {
	secret := buildSecret(t, API_SECRET_REQ)
	token := make([]byte, hex.EncodedLen(len(secret)))
	hex.Encode(token, secret)

	// 请求后返回的结果用
	// secret := buildSecret(t, [API_SECRET_RESP])
	// 来解密

	return map[string]string{
		"Authorization":            "Bearer",
		"User-Agent":               UserAgent,
		"Token":                    string(token),
		"Tokenparam":               strconv.FormatInt(t.Unix(), 10) + "," + VERSION,
		"Sec-Fetch-Storage-Access": "active",
	}
}

func BuildImageHeaders() map[string]string {
	return map[string]string{
		"Accept":         "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8",
		"Referer":        ApiHost,
		"User-Agent":     UserAgent,
		"Sec-Fetch-Dest": "image",
		"Sec-Fetch-Mode": "no-cors",
	}
}

func BuildCoverUrl(comicId int) string {
	return fmt.Sprintf("%s/media/albums/%d_3x4.jpg", ImageUrl, comicId)
}

func BuildImageUrl(chapterId int, imageName string) string {
	return fmt.Sprintf("%s/media/photos/%d/%s", ImageUrl, chapterId, imageName)
}

var httpClient = http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DisableCompression:    true, // diable gzip
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// Get 返回 body
func Get(ctx context.Context, url string) (_ []byte, _ *http.Response, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range constant.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range BuildImageHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, resp, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return body, resp, err
}

// GetApi 带解密
func GetApi(ctx context.Context, url string) ([]byte, error) {
	t := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range constant.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range BuildApiHeaders(t) {
		req.Header.Set(k, v)
	}

	// 有时'data'直接是json，不需要解密 (?
	// if _, ok := j["content"]; ok {
	// 	return b, nil
	// } else {
	// 	return b, fmt.Errorf("invalid data type: %T", data)
	// }

	return DoApi(req, t)
}

// PostApi 带解密
func PostApi(ctx context.Context, url string, data io.Reader) ([]byte, error) {
	t := time.Now()

	headers := BuildApiHeaders(t)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, data)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return DoApi(req, t)
}

// DoApi 带解密
func DoApi(req *http.Request, t time.Time) ([]byte, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return b, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	apiResp := &ApiResponse{}
	err = json.Unmarshal(b, apiResp)
	if err != nil {
		type RespErr struct {
			Code     int    `json:"code"`
			ErrorMsg string `json:"errorMsg"`
		}
		respErr := RespErr{}
		if err := json.Unmarshal(b, &respErr); err == nil {
			return nil, wrapErr(fmt.Errorf("bad response status code: %d", respErr.Code), respErr.ErrorMsg)
		}
		return nil, wrapErr(err, string(b))
	}
	return apiResp.Decrypt(t)
}

type ApiResponse struct {
	Code int    `json:"code"`
	Data string `json:"data"` // encrypted
	// ErrorMsg string `json:"errorMsg"`
}

func (ar *ApiResponse) Decrypt(t time.Time) ([]byte, error) {
	if ar.Code != http.StatusOK {
		return nil, fmt.Errorf("ApiResponse.Decrypt: unexpected status code: %d", ar.Code)
	}
	return decrypt([]byte(ar.Data), buildSecret(t, API_SECRET_RESP))
}

func decrypt(input, secret []byte) ([]byte, error) {
	// 去除可能出现在开头的 UTF-8 BOM (U+FEFF)
	input = bytes.TrimPrefix(input, []byte("\uFEFF"))

	key := make([]byte, hex.EncodedLen(len(secret)))
	hex.Encode(key, secret)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	const bs = aes.BlockSize

	enc := base64.StdEncoding
	data := make([]byte, enc.DecodedLen(len(input)))
	n, err := enc.Decode(data, input)
	if err != nil {
		return nil, err
	}
	data = data[:n]

	if m := len(data) % bs; m != 0 {
		return nil, fmt.Errorf("len(data) (%d) %% bs != 0 (%d)", len(data), m)
	}

	decrypted := make([]byte, len(data))
	for i := 0; i < len(data); i += bs {
		block.Decrypt(decrypted[i:i+bs], data[i:i+bs])
	}

	// 移除填充 (PKCS7)
	unpadded, err := pkcs7Unpad(decrypted, bs)
	if err == nil {
		return unpadded, nil
	}
	// 失败使用原始的字符串裁剪逻辑
	lastBracket := max(
		bytes.LastIndex(decrypted, []byte("}")),
		bytes.LastIndex(decrypted, []byte("]")),
	)
	if lastBracket != -1 {
		return decrypted[:lastBracket+1], nil
	}
	return decrypted, nil
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("invalid padded data length")
	}
	pad := int(data[len(data)-1])
	if pad <= 0 || pad > blockSize {
		return nil, fmt.Errorf("invalid padding size: %d", pad)
	}
	for i := range pad {
		if data[len(data)-1-i] != byte(pad) {
			return nil, fmt.Errorf("invalid padding byte at pos %d", len(data)-1-i)
		}
	}
	return data[:len(data)-pad], nil
}
