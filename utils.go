package EHentai

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Miuzarte/EHentai-go/internal/utils"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/image/webp"
)

func urlGetDomain(u string) Domain {
	switch {
	case strings.Contains(u, EHENTAI_DOMAIN):
		return EHENTAI_DOMAIN
	case strings.Contains(u, EXHENTAI_DOMAIN):
		return EXHENTAI_DOMAIN
	}
	return ""
}

func UrlGetGIdGToken(u string) (domain Domain, gId string, gToken string) {
	// https://e-hentai.org/g/{gallery_id}/{gallery_token}
	// https://e-hentai.org/g/3138775/30b0285f9b
	u = strings.TrimSuffix(u, "/")
	splits := strings.Split(u, "/")
	for i, s := range splits {
		if s == "g" {
			// i   g
			// i+1 3138775
			// i+2 30b0285f9b
			if len(splits) < i+3 {
				break
			}
			return splits[i-1], splits[i+1], splits[i+2]
		}
	}
	return "", "", ""
}

func UrlToGallery(u string) Gallery {
	_, gId, gToken := UrlGetGIdGToken(u)
	gid, _ := atoi(gId)
	return Gallery{GalleryId: gid, GalleryToken: gToken}
}

func UrlGetPTokenGIdPNum(u string) (domain Domain, pToken string, gId string, pNum string) {
	// https://e-hentai.org/s/{page_token}/{gallery_id}-{pagenumber}
	// https://e-hentai.org/s/0b2127ea05/3138775-8
	u = strings.TrimSuffix(u, "/")
	splits := strings.Split(u, "/")
	for i, s := range splits {
		if s == "s" {
			// i   s
			// i+1 0b2127ea05
			// i+2 3138775-8
			if len(splits) < i+3 {
				break
			}
			tail := strings.Split(splits[i+2], "-")
			if len(tail) < 2 {
				break
			}
			return splits[i-1], splits[i+1], tail[0], tail[1]
		}
	}
	return "", "", "", ""
}

func UrlToPage(u string) Page {
	_, pToken, gId, pNum := UrlGetPTokenGIdPNum(u)
	gid, _ := atoi(gId)
	pageNum, _ := atoi(pNum)
	return Page{PageToken: pToken, GalleryId: gid, PageNum: pageNum}
}

func httpGet(ctx context.Context, url string) (resp *http.Response, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err = httpClient.Do(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
			return resp, err
		}
		return nil, err
	}

	return resp, nil
}

func httpGetDoc(ctx context.Context, url string) (doc *goquery.Document, err error) {
	again := false
AGAIN:
	resp, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err = goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	if ipBannedCheck(doc) {
		return nil, wrapErr(ErrIpBanned, nil)
	}
	if sadPandaCheck(doc) {
		if !again {
			again = true
			goto AGAIN // 可能 igneous 过期并重新下发, cookie 更新后重试
		}
		return nil, wrapErr(ErrSadPanda, nil)
	}
	return doc, nil
}

func post[T any](ctx context.Context, url string, body any) (*T, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody := new(T)
	err = json.NewDecoder(resp.Body).Decode(respBody)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func extractMainDomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return host
	}
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}

func sadPandaCheck(doc *goquery.Document) bool {
	head := doc.Find("head")
	body := doc.Find("body")
	emptyHead := head.Text() == ""
	emptyBody := body.Text() == ""
	return emptyHead && emptyBody
}

func ipBannedCheck(doc *goquery.Document) bool {
	// This IP address has been temporarily banned due to an excessive request rate.
	// This probably means you are using automated mirroring/harvesting software or share the IP address with someone who does.
	// The ban expires in 19 hours
	return strings.Contains(doc.Find("body").Text(), "This IP address has been temporarily banned")
}

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~uintptr
}

type number interface {
	integer | ~float32 | ~float64
}

func itoa[T number](i T) string {
	return strconv.FormatInt(int64(i), 10)
}

func atoi(s string) (int, error) {
	return strconv.Atoi(s)
}

func cleanOutOfRange(sLen int, indexes []int) (cleaned []int) {
	if len(indexes) == 0 ||
		slices.Max(indexes) < sLen && slices.Min(indexes) >= 0 {
		return indexes
	}
	cleaned = make([]int, 0, len(indexes))
	for _, i := range indexes {
		if i < sLen && i >= 0 {
			cleaned = append(cleaned, i)
		}
	}
	return
}

func sliceAdd(s []int, op int) []int {
	if len(s) == 0 {
		return nil
	}
	newS := make([]int, len(s))
	for i := range s {
		newS[i] = s[i] + op
	}
	return newS
}

func rearange[T any](s []T, indexes []int) (rearranged []T) {
	if len(indexes) == 0 {
		return s
	}
	rearranged = make([]T, len(indexes))
	for i := range indexes {
		rearranged[i] = s[indexes[i]]
	}
	return
}

// dirLookupExt 查找目录下的文件，返回指定扩展名的文件列表
func dirLookupExt(dirEnts []os.DirEntry, exts ...string) []os.DirEntry {
	extsMap := utils.Set[string]{}
	for _, ext := range exts {
		extsMap[strings.ToLower(ext)] = struct{}{}
	}

	des := []os.DirEntry{}
	for _, de := range dirEnts {
		if de.IsDir() {
			continue
		}
		parts := strings.Split(de.Name(), ".")
		if len(parts) < 2 {
			continue
		}
		ext := strings.ToLower(parts[len(parts)-1])
		if _, ok := extsMap[ext]; ok {
			des = append(des, de)
		}
	}
	return des
}

// 为了避免在索引缓存元数据时遇到的越界问题,
// 元数据中以 map[string]string 储存页链接
func pageUrlsToSlice(pageUrls map[string]string) (urls []string) {
	urls = make([]string, len(pageUrls))
	for i := range len(pageUrls) {
		urls[i] = pageUrls[itoa(i)]
	}
	return urls
}

// 去重收集画廊 ID
func collectGIds(pageUrls []string) (gIds utils.Set[int]) {
	gIds = make(utils.Set[int], len(pageUrls))
	for _, pageUrl := range pageUrls {
		gId := UrlToPage(pageUrl).GalleryId
		gIds[gId] = struct{}{}
	}
	return gIds
}

func tryDecodeImage(data []byte) (img image.Image, _ ImageType) {
	var err error
	switch http.DetectContentType(data) {
	case "image/webp":
		img, err = webp.Decode(bytes.NewReader(data))
		if err == nil {
			return img, IMAGE_TYPE_WEBP
		}
	case "image/jpeg":
		img, err = jpeg.Decode(bytes.NewReader(data))
		if err == nil {
			return img, IMAGE_TYPE_JPEG
		}
	case "image/png":
		img, err = png.Decode(bytes.NewReader(data))
		if err == nil {
			return img, IMAGE_TYPE_PNG
		}
	}
	return nil, IMAGE_TYPE_UNKNOWN
}
