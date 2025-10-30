package constant

var Headers = map[string]string{
	"Accept": "*/*",
	// "Accept-Encoding":  "gzip, deflate, br, zstd",
	"Accept-Encoding":  "identity",
	"Accept-Language":  "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7",
	"Connection":       "keep-alive",
	"Origin":           "https://localhost",
	"Referer":          "https://localhost/",
	"Sec-Fetch-Dest":   "empty",
	"Sec-Fetch-Mode":   "cors",
	"Sec-Fetch-Site":   "cross-site",
	"X-Requested-With": "com.example.app",
}

var ApiHosts = [...]string{
	"www.cdnaspa.vip",
	"www.cdnaspa.club",
	"www.cdnplaystation6.vip",
	"www.cdnplaystation6.cc",
}
