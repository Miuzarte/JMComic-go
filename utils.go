package JmComic

import (
	"crypto/md5"
	"encoding/json"
	"slices"
	"strconv"
	"time"
)

func unmarshalTo[T any](data []byte) (*T, error) {
	v := new(T)
	return v, json.Unmarshal(data, v)
}

func buildSecret(t time.Time, secret string) []byte {
	sum := md5.Sum(slices.Concat(
		[]byte(strconv.FormatInt(t.Unix(), 10)),
		[]byte(secret),
	))
	return sum[:] 
}
