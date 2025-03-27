package EHentai

import (
	"strconv"
	"strings"
)

// alias
var (
	itoa = strconv.Itoa
	atoi = strconv.Atoi
)

type ImageType int

const (
	IMAGE_TYPE_UNKNOWN ImageType = iota

	IMAGE_TYPE_WEBP // 某个时期之后似乎都是 webp
	IMAGE_TYPE_JPEG
	IMAGE_TYPE_PNG // 好像没有
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
	default:
		return IMAGE_TYPE_UNKNOWN
	}
}

type CacheState int

const (
	CACHE_STATE_UNKNOWN CacheState = iota
	CACHE_STATE_NONE
	CACHE_STATE_PARTIAL
	CACHE_STATE_FULL
)

func (cs CacheState) String() string {
	switch cs {
	case CACHE_STATE_UNKNOWN:
		return "unknown"
	case CACHE_STATE_NONE:
		return "none"
	case CACHE_STATE_PARTIAL:
		return "partial"
	case CACHE_STATE_FULL:
		return "full"
	default:
		return ""
	}
}

type Domain = string

const (
	EHENTAI_DOMAIN  Domain = "e-hentai.org"
	EXHENTAI_DOMAIN Domain = "exhentai.org"
)

type Torrent struct {
	Hash  string `json:"hash"`
	Added string `json:"added"`
	Name  string `json:"name"`
	TSize string `json:"tsize"`
	FSize string `json:"fsize"`
}

type GalleryMetadata struct {
	GId          int       `json:"gid"`
	Token        string    `json:"token"`
	ArchiverKey  string    `json:"archiver_key"`
	Title        string    `json:"title"`
	TitleJpn     string    `json:"title_jpn"`
	Category     string    `json:"category"`
	Thumb        string    `json:"thumb"`
	Uploader     string    `json:"uploader"`
	Posted       string    `json:"posted"`
	FileCount    string    `json:"filecount"`
	FileSize     int       `json:"filesize"`
	Expunged     bool      `json:"expunged"`
	Rating       string    `json:"rating"`
	TorrentCount string    `json:"torrentcount"`
	Torrents     []Torrent `json:"torrents"`
	Tags         []string  `json:"tags"`
	ParentGId    string    `json:"parent_gid"`
	ParentKey    string    `json:"parent_key"`
	FirstGId     string    `json:"first_gid"`
	FirstKey     string    `json:"first_key"`
	Error        string    `json:"error,omitzero"`
}

func (gm *GalleryMetadata) GetCover() (url string) {
	return gm.Thumb
}

type GalleryMetadataResponse struct {
	GMetadata []GalleryMetadata `json:"gmetadata"`
}

type GalleryMetadataRequest struct {
	Method    string  `json:"method"`
	GIdList   [][]any `json:"gidlist"`
	Namespace int     `json:"namespace"`
}

// Gallery describes the gallery url
//
// https://e-hentai.org/g/{gallery_id}/{gallery_token}/
type Gallery struct {
	GalleryId    int
	GalleryToken string
}

// GIdList is the official alias of [Gallery]
type GIdList = Gallery

type TokenList struct {
	GId   int    `json:"gid"`
	Token string `json:"token"`
}

type GalleryTokenResponse struct {
	TokenLists []TokenList `json:"tokenlist"`
}

type GalleryTokenRequest struct {
	Method   string  `json:"method"`
	PageList [][]any `json:"pagelist"`
}

// Page describes the page url
//
// https://e-hentai.org/s/{page_token}/{gallery_id}-{pagenumber}
type Page struct {
	PageToken string `json:"page_token"`
	GalleryId int    `json:"gallery_id"`
	PageNum   int    `json:"page_num"`
}

// PageList is the official alias of [Page]
type PageList = Page

// coverProvider 统一搜索结果与元数据结果的封面获取
type coverProvider interface {
	GetCover() (url string)
}

type FSearchResult struct {
	Domain Domain
	Gid    int
	Token  string
	Cat    string
	Cover  string
	Rating string
	Url    string
	Tags   []string
	Title  string // 根据 cookie 中的 sk, 结果可能为英文或日文
	Pages  string
}

func (f *FSearchResult) GetCover() (url string) {
	return f.Cover
}

type Image struct {
	Data []byte
	Type ImageType
}

// PageData carrys page info and image data
type PageData struct {
	Page
	Image
}

type CachePageInfo struct {
	Num  int `json:"num"`
	Type int `json:"type"` // see [ImageType]
	Len  int `json:"len"`
}

type CachePageInfos []CachePageInfo

// Exist 判断页码是否存在
func (cpi *CachePageInfos) Exist(pageNum int) bool {
	for _, info := range *cpi {
		if info.Num == pageNum {
			return true
		}
	}
	return false
}

// Lookup 构造哈希表查找页码
//
// len(pageNums) == 0 时返回所有
//
// len(pageInfos) == len(pageNums)
// , 对应页不存在时 Len 为 0
func (cpi *CachePageInfos) Lookup(pageNums []int) (pageInfos CachePageInfos) {
	if len(*cpi) == 0 {
		return nil
	}
	if len(pageNums) == 0 {
		return *cpi
	}

	pagesMap := make(map[int]*CachePageInfo, len(*cpi))
	for i := range pageNums {
		pagesMap[(*cpi)[i].Num] = &(*cpi)[i]
	}

	pageInfos = make(CachePageInfos, 0, len(pageNums))
	for _, pageNum := range pageNums {
		if page, ok := pagesMap[pageNum]; ok {
			pageInfos = append(pageInfos, *page)
		} else {
			pageInfos = append(pageInfos, CachePageInfo{Num: pageNum, Type: 0, Len: 0})
		}
	}
	return pageInfos
}

type CacheGalleryMetadata struct {
	Url     string          `json:"url"` // 画廊 URL
	Gallery GalleryMetadata `json:"gallery"`

	Pages []string `json:"pages"` // 画廊所有页的 URL, 读取缓存图片出错时可以直接用

	Files struct {
		Dir   string         `json:"dir"`   // 画廊路径 `root/gid/`
		Count int            `json:"count"` // 缓存数量
		Pages CachePageInfos `json:"pages"`
	} `json:"files"` // 缓存文件列表
}
