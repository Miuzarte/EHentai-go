package EHentai

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"slices"
	"strings"

	"github.com/Miuzarte/EHentai-go/internal/utils"
	"golang.org/x/image/webp"
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

func ParseImageMimeType(mimeType string) ImageType {
	switch strings.ToLower(mimeType) {
	case "image/webp":
		return IMAGE_TYPE_WEBP
	case "image/jpeg":
		return IMAGE_TYPE_JPEG
	case "image/png":
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
	EHENTAI_DOMAIN  Domain = `e-hentai.org`
	EXHENTAI_DOMAIN Domain = `exhentai.org`
)

type Url = string

const (
	EHENTAI_URL  Url = `https://` + EHENTAI_DOMAIN
	EXHENTAI_URL Url = `https://` + EXHENTAI_DOMAIN
)

type Tag struct {
	Namespace string
	Name      string
}

func (t Tag) String() string {
	return t.Namespace + ":" + t.Name
}

type Tags []Tag

func (t Tags) Namespaces() (namespaces []string) {
	namespaces = make([]string, len(t))
	for i := range t {
		namespaces[i] = t[i].Namespace
	}
	s := utils.Set[string]{}
	return s.Clean(namespaces)
}

func (t Tags) Names() (names []string) {
	names = make([]string, len(t))
	for i := range t {
		names[i] = t[i].Name
	}
	return names
}

func (t Tags) Set() (ts []TagSet) {
	setPos := map[string]int{}
	for _, tag := range t {
		i, ok := setPos[tag.Namespace]
		if !ok {
			i = len(ts)
			setPos[tag.Namespace] = i
			ts = append(ts, TagSet{Namespace: tag.Namespace})
		}
		ts[i].Tags = append(ts[i].Tags, tag.Name)
	}
	return ts
}

type TagSet struct {
	Namespace string
	Tags      []string
}

// GalleryDetails 来自画廊详情页
type GalleryDetails struct {
	Domain Domain
	Gallery

	Cover    string // url
	Title    string // 英文标题
	TitleJpn string // 日文标题

	Cat      string // 分类
	Uploader string // 上传者

	Posted     string // "2006-01-02 15:04"
	Parent     int    // galleryId, 0 for "None"
	Visible    string // "Yes" / "No (Replaced)"
	Language   string // "Chinese" / "Japanese"
	Translated string // "TR" / ""
	FileSize   string // "449.9 MiB"
	Length     int    // 65
	Favorited  int    // 3745

	RatingCount int     // 271
	Rating      float64 // 4.86

	Tags []Tag

	PageUrls []string
}

func (gd *GalleryDetails) GetCover() (urls []string) {
	return []string{gd.Cover}
}

type Torrent struct {
	Hash  string `json:"hash"`
	Added string `json:"added"`
	Name  string `json:"name"`
	TSize string `json:"tsize"`
	FSize string `json:"fsize"`
}

// GalleryMetadata 来自官方 api
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

type GalleryMetadatas []GalleryMetadata

func (gms GalleryMetadatas) GetCover() (urls []string) {
	urls = make([]string, len(gms))
	for i := range gms {
		urls[i] = (gms)[i].Thumb
	}
	return urls
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

func (tl *TokenList) ToGallery() Gallery {
	return Gallery{
		GalleryId:    tl.GId,
		GalleryToken: tl.Token,
	}
}

// Page describes the page url
//
// https://e-hentai.org/s/{page_token}/{gallery_id}-{pagenumber}
type Page struct {
	PageToken string `json:"page_token"`
	GalleryId int    `json:"gallery_id"`
	PageNum   int    `json:"page_num"`
}

func (p *Page) String() string {
	return "/s/" + p.PageToken + "/" + itoa(p.GalleryId) + "-" + itoa(p.PageNum)
}

// PageList is the official alias of [Page]
type PageList = Page

// coverProviders 统一搜索结果与元数据结果的封面获取
type coverProviders interface {
	GetCover() (urls []string)
}

// FSearchResult 来自搜索结果页
type FSearchResult struct {
	Domain Domain
	Gallery

	Cat string // 分类

	Cover  string  // url
	Posted string  // "2006-01-02 15:04"
	Rating float64 // 精确到 0.5

	Url   string   // 画廊 URL
	Title string   // 根据 cookie 中的 sk, 结果可能为英文或日文
	Tags  []string // namespace:tag

	Uploader string // 上传者
	Pages    int    // 64
}

type FSearchResults []FSearchResult

func (fsr FSearchResults) GetCover() (urls []string) {
	urls = make([]string, len(fsr))
	for i := range fsr {
		urls[i] = fsr[i].Cover
	}
	return urls
}

type Image struct {
	Data    []byte
	Type    ImageType
	TypeRaw string
}

func (i *Image) String() string {
	return "image/" + i.Type.String() + " (" + itoa(len(i.Data)) + " bytes)"
}

func (i *Image) Decode() (image.Image, error) {
	switch i.Type {
	case IMAGE_TYPE_WEBP:
		return webp.Decode(bytes.NewReader(i.Data))
	case IMAGE_TYPE_JPEG:
		return jpeg.Decode(bytes.NewReader(i.Data))
	case IMAGE_TYPE_PNG:
		return png.Decode(bytes.NewReader(i.Data))
	default:
		img, typ := tryDecodeImage(i.Data)
		if typ != IMAGE_TYPE_UNKNOWN {
			i.Type = typ
			return img, nil
		}
	}
	return nil, wrapErr(ErrUnknownImageType, i.TypeRaw)
}

// PageData carrys page info and image data
type PageData struct {
	Page
	Image
	FromCache bool
}

func (pd *PageData) String() string {
	return pd.Page.String() + ": " + pd.Image.String()
}

type CachePageInfo struct {
	Num  int       `json:"num"` // page num
	Type ImageType `json:"type"`
	Len  int       `json:"len"` // file size
}

type CachePageInfos []CachePageInfo

// Exts 返回存在的图片扩展名
func (cpi *CachePageInfos) Exts() []ImageType {
	set := utils.Set[ImageType]{}
	for _, pageInfo := range *cpi {
		set.Add(pageInfo.Type)
	}
	return set.Get()
}

// Get 返回页码对应的缓存信息
//
// Len 为 0 表示不存在
func (cpi *CachePageInfos) Get(pageNum int) (pageInfo CachePageInfo) {
	for i := range *cpi {
		if (*cpi)[i].Num == pageNum {
			return (*cpi)[i]
		}
	}
	return CachePageInfo{Num: pageNum, Type: IMAGE_TYPE_UNKNOWN, Len: 0}
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
		pageInfos = make(CachePageInfos, 0, len(*cpi))
		pageInfos = append(pageInfos, (*cpi)...)
		return pageInfos
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
			pageInfos = append(pageInfos, CachePageInfo{Num: pageNum, Type: IMAGE_TYPE_UNKNOWN, Len: 0})
		}
	}
	return pageInfos
}

func (cpi *CachePageInfos) append(pageInfos ...CachePageInfo) {
	*cpi = append(*cpi, pageInfos...)
}

func (cpi *CachePageInfos) del(pageNum int) {
	for i := range *cpi {
		if (*cpi)[i].Num == pageNum {
			*cpi = slices.Delete(*cpi, i, i+1)
			return
		}
	}
}

type CacheGalleryMetadata struct {
	Url     string          `json:"url"` // 画廊 URL
	Gallery GalleryMetadata `json:"gallery"`

	// PageUrls []string `json:"page_urls"`
	// 画廊所有页的 URL, 读取缓存图片出错时可以直接用
	// 保证一定存在
	PageUrls map[string]string `json:"page_urls"`

	Files struct {
		Dir      string         `json:"dir"`       // 画廊路径 `root/gid/`
		Count    int            `json:"count"`     // 缓存数量
		TotalLen int            `json:"total_len"` // 缓存总大小
		Pages    CachePageInfos `json:"pages"`     // 缓存的页码列表
	} `json:"files"` // 缓存文件列表
}
