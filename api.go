package JmComic

const (
	API_SETTING = "/setting"
	API_SEARCH  = "/search"
	API_ALBUM   = "/album"
	API_CHAPTER = "/chapter"
)

type Server struct {
	Setting   []string   `json:"Setting"`
	Server    []string   `json:"Server"`
	Jm3Server [][]string `json:"jm3_Server"` // [0]: host, [1]: "綫路n"
}

type SettingAppShunt struct {
	Title string `json:"title"`
	Key   int    `json:"key"`
}

type Setting struct {
	Version     string `json:"version"`      // "1.8.2"
	TestVersion string `json:"test_version"` // "1.8.2"
	StoreLink   string `json:"store_link"`

	IosVersion     string `json:"ios_version"`      // "1.8.2"
	IosTestVersion string `json:"ios_test_version"` // "1.8.2"
	IosStoreLink   string `json:"ios_store_link"`

	Jm3Version     string `json:"jm3_version"`      // "2.0.12"
	Jm3TestVersion string `json:"jm3_test_version"` // "2.0.12"
	Jm3StoreLink   string `json:"jm3_store_link"`

	Jm3IOSVersion     string `json:"jm3_ios_version"`      // "1.0.0"
	Jm3IOSTestVersion string `json:"jm3_ios_test_version"` // "1.0.0"
	Jm3DownloadUrl    string `json:"jm3_download_url"`

	IpCountry string `json:"ipcountry"` // "CA"

	NewYearEvent   bool   `json:"newYearEvent"`
	AdCacheVersion int    `json:"ad_cache_version"`
	BundleUrl      string `json:"bundle_url"`
	DownloadUrl    string `json:"download_url"`
	AppLandingPage string `json:"app_landing_page"`

	VersionInfo    string `json:"version_info"`
	Jm3IsHotUpdate bool   `json:"jm3_is_hot_update"`
	Jm3VersionInfo string `json:"jm3_version_info"`
	IsHotUpdate    bool   `json:"is_hot_update"`

	BaseUrl     string `json:"base_url"` // "https://www.cdnaspa.vip"
	DonateUrl   string `json:"donate_url"`
	MainWebHost string `json:"main_web_host"`

	// "https://tencent.jmdanjonproxy.xyz" | "https://cdn-msp.jmdanjonproxy.xyz"
	ImgHost string `json:"img_host"`
	// "https://tencent.jmdanjonproxy.xyz/media/logo/channel_log.png?v=" | "cdn-msp"
	ApiBannerPath string `json:"api_banner_path"`
	// "https://tencent.jmdanjonproxy.xyz/media/logo/new_logo.png" | "cdn-msp"
	LogoPath string `json:"logo_path"`

	FloatAd bool `json:"float_ad"`

	IsCn      int    `json:"is_cn"`
	CnBaseURL string `json:"cn_base_url"` // "https://www.cdnaspa.vip"

	AppShunts []SettingAppShunt `json:"app_shunts"`

	FoolsDayEvent bool `json:"foolsDayEvent"`
}

type ComicBasicCategory struct {
	Id    string `json:"id"`    // "1", "2", "5"
	Title string `json:"title"` // "同人", "单本", "韩漫"
}

type ComicBasic struct {
	Id          string `json:"id"`
	Author      string `json:"author"`
	Description string `json:"description"` // null
	Name        string `json:"name"`
	Image       string `json:"image"` // ""

	Category    ComicBasicCategory `json:"category"`
	CategorySub ComicBasicCategory `json:"category_sub"`

	Liked      bool `json:"liked"`
	IsFavorite bool `json:"is_favorite"`
}

type SearchResp struct {
	SearchQuery string       `json:"search_query"`
	Total       string       `json:"total"`
	Content     []ComicBasic `json:"content"`
}

type Album struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Images      []any  `json:"images"`      // empty
	Addtime     string `json:"addtime"`     // timestamp (second)
	Description string `json:"description"` // ""
	TotalViews  string `json:"total_views"` // int
	Likes       string `json:"likes"`       // int

	Series   []any  `json:"series"`    // TODO
	SeriesId string `json:"series_id"` // TODO

	CommentTotal string `json:"comment_total"` // int

	Author []string `json:"author"`
	Tags   []string `json:"tags"`
	Works  []string `json:"works"`  // 作品 // "公主连结"
	Actors []string `json:"actors"` // 人物 // "贪吃佩可"

	RelatedList []struct {
		Id     string `json:"id"`
		Author string `json:"author"`
		Name   string `json:"name"`
		Images string `json:"images"` // ""
	} `json:"related_list"`

	Liked      bool   `json:"liked"`
	IsFavorite bool   `json:"is_favorite"`
	IsAids     bool   `json:"is_aids"`
	Price      string `json:"price"`     // ""
	Purchased  string `json:"purchased"` // ""
}

type Chapter struct {
	Id         int      `json:"id"`
	Series     []string `json:"series"` // TODO
	Tags       string   `json:"tags"`   // space separated list
	Name       string   `json:"name"`
	Images     []string `json:"images"`    // "00001.webp"
	Addtime    string   `json:"addtime"`   // timestamp (second)
	SeriesId   string   `json:"series_id"` // int
	IsFavorite bool     `json:"is_favorite"`
	Liked      bool     `json:"liked"`
}
