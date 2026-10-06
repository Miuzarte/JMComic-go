package constant

var Headers = map[string]string{
	"Accept": "*/*",
	// "Accept-Encoding":  "gzip, deflate, br, zstd",
	"Accept-Encoding":  "identity",
	"Accept-Language":  "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7",
	"Connection":       "keep-alive",
	"Sec-Fetch-Dest":   "empty",
	"Sec-Fetch-Mode":   "cors",
	"Sec-Fetch-Site":   "cross-site",
	"X-Requested-With": "com.JMComic3.app",
}

var ApiHosts = [...]string{
	"www.cdnhjk.net",
	"www.cdngwc.cc",
	"www.cdngwc.net",
	"www.cdngwc.club",
}

var ImageHosts = [...]string{
	"https://cdn-msp.jmapinodeudzn.net",
	"https://cdn-msp3.jmapinodeudzn.net",
	"https://cdn-msp.jmapiproxy1.cc",
	"https://cdn-msp.jmapiproxy2.cc",
	"https://cdn-msp2.jmapiproxy2.cc",
	"https://cdn-msp3.jmapiproxy2.cc",
}
