package EHentai

import (
	"context"
	"net/http"
	netUrl "net/url"
	"os"
	"slices"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
	u = strings.TrimRight(u, "/")
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
	u = strings.TrimRight(u, "/")
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

func pageUrlsToGalleryId(pageUrls []string) set[string] {
	gIds := make(set[string])
	for _, u := range pageUrls {
		_, gId, _ := UrlGetGIdGToken(u)
		gIds[gId] = struct{}{}
	}
	return gIds
}

func httpGet(ctx context.Context, url *netUrl.URL) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	if cookie.Ok() {
		req.Header.Set("Cookie", cookie.String())
	}
	return http.DefaultClient.Do(req)
}

func httpGetDoc(ctx context.Context, url *netUrl.URL) (*goquery.Document, error) {
	resp, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	if sadPandaCheck(doc) {
		return nil, wrapErr(ErrSadPanda, nil)
	}
	return doc, nil
}

func sadPandaCheck(doc *goquery.Document) bool {
	return doc.Find("head").Length() == 0 && doc.Find("body").Length() == 0
}

func removeDuplicates(indexes []int) []int {
	seen := make(map[int]struct{})
	result := []int{}
	for _, index := range indexes {
		if _, ok := seen[index]; !ok {
			seen[index] = struct{}{}
			result = append(result, index)
		}
	}
	return result
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

func rearrange[T any](s []T, indexes []int) (trimed []T) {
	trimed = make([]T, 0, len(indexes))
	for _, i := range indexes {
		trimed = append(trimed, s[i])
	}
	return
}

type set[T comparable] map[T]struct{}

func removeDuplication[T comparable](s []T) []T {
	if len(s) == 0 {
		return s
	}

	m := make(set[T], len(s))
	d := make([]T, 0, len(s))
	for i := range s {
		if _, ok := m[s[i]]; ok {
			continue
		}
		m[s[i]] = struct{}{}
		d = append(d, s[i])
	}
	return d
}

// dirLookupExt 查找目录下的文件，返回指定扩展名的文件列表
func dirLookupExt(dirEnts []os.DirEntry, exts ...string) []os.DirEntry {
	extsMap := make(set[string])
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
