package JmComic

import (
	"crypto/md5"
	"encoding/json"
	"slices"
	"strconv"
	"time"

	"github.com/go-viper/mapstructure/v2"
)

func unmarshalTo[T any](data []byte, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	// fmt.Printf("%s\n", data)

	input := make(map[string]any)
	err = json.Unmarshal(data, &input)
	if err != nil {
		return nil, err
	}

	output := new(T)
	config := mapstructure.DecoderConfig{
		TagName:          "json",
		Result:           output,
		WeaklyTypedInput: true,
	}
	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		panic(err)
	}
	err = decoder.Decode(input)
	if err != nil {
		panic(err)
	}

	return output, nil
}

func buildSecret(t time.Time, secret string) []byte {
	sum := md5.Sum(slices.Concat(
		[]byte(strconv.FormatInt(t.Unix(), 10)),
		[]byte(secret),
	))
	return sum[:]
}
