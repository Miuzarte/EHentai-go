package EHentai

import (
	"unsafe"
)

func SetCookie(memberId, passHash, igneous, sk string) {
	cookie.IpbMemberId = memberId
	cookie.IpbPassHash = passHash
	cookie.Igneous = igneous
	cookie.Sk = sk
}

// SetThreads 设置下载线程数
func SetThreads(n int) {
	threads = n
}

// SetRetryDepth 设置重试次数
// , 默认为 2
// , 直链下载失败时使用页备链重试
func SetRetryDepth(depth int) {
	retryDepth = depth
}

func InitEhTagDB() error {
	return database.Init()
}

// EHQueryFSearch 搜索 EHentai, results 只有第一页结果
func EHQueryFSearch(keyword string, categories ...Category) (total int, results []EhFSearchResult, err error) {
	return queryFSearch(EHENTAI_URL, keyword, categories...)
}

// ExHQueryFSearch 搜索 ExHentai, results 只有第一页结果
func ExHQueryFSearch(keyword string, categories ...Category) (total int, results []EhFSearchResult, err error) {
	if !cookie.Ok() {
		return 0, nil, ErrCookieNotSet
	}
	return queryFSearch(EXHENTAI_URL, keyword, categories...)
}

// EHSearchDetail 搜索 EHentai, 返回详细信息, results 只有第一页结果
func EHSearchDetail(keyword string, categories ...Category) (total int, galleries []GalleryMetadata, err error) {
	_, results, err := EHQueryFSearch(keyword, categories...)
	if err != nil {
		return
	}
	list := *(*[]GIdList)(unsafe.Pointer(&results))
	resp, err := PostGalleryMetadata(list...)
	if err != nil {
		return 0, nil, err
	}
	return total, resp.GMetadata, nil
}

// ExHSearchDetail 搜索 ExHentai, 返回详细信息, results 只有第一页结果
func ExHSearchDetail(keyword string, categories ...Category) (total int, galleries []GalleryMetadata, err error) {
	total, results, err := ExHQueryFSearch(keyword, categories...)
	if err != nil {
		return
	}
	list := *(*[]GIdList)(unsafe.Pointer(&results))
	resp, err := PostGalleryMetadata(list...)
	if err != nil {
		return 0, nil, err
	}
	return total, resp.GMetadata, nil
}

// DownloadGallery 下载画廊下所有图片, 下载失败时尝试备链
func DownloadGallery(galleryUrl string) (imgDatas [][]byte, err error) {
	err = checkDomain(galleryUrl)
	if err != nil {
		return nil, err
	}
	pageUrls, err := fetchGalleryPages(galleryUrl)
	if err != nil {
		return nil, err
	}
	return downloadPages(pageUrls...)
}

// DownloadPages 下载画廊某页的图片, 下载失败时尝试备链
func DownloadPages(pageUrls ...string) (imgDatas [][]byte, err error) {
	for _, pageUrl := range pageUrls {
		err = checkDomain(pageUrl)
		if err != nil {
			return nil, err
		}
	}
	return downloadPages(pageUrls...)
}

// FetchGalleryPageUrls 获取画廊下所有页链接
func FetchGalleryPageUrls(galleryUrl string) (pageUrls []string, err error) {
	err = checkDomain(galleryUrl)
	if err != nil {
		return nil, err
	}
	return fetchGalleryPages(galleryUrl)
}

// FetchGalleryImageUrls 获取画廊下所有图片的直链与页面备链
// , 不建议使用
// , [DownloadPages] 可使用备链自动重试
func FetchGalleryImageUrls(galleryUrl string) (imgUrls []string, bakPages []string, err error) {
	if err = checkDomain(galleryUrl); err != nil {
		return nil, nil, err
	}
	pageUrls, err := fetchGalleryPages(galleryUrl)
	if err != nil {
		return nil, nil, err
	}
	for _, pageUrl := range pageUrls {
		imgUrl, bak, err := fetchPageImageUrl(pageUrl)
		if err != nil {
			return nil, nil, err
		}
		imgUrls = append(imgUrls, imgUrl)
		bakPages = append(bakPages, bak)
	}
	return
}

// FetchPageImageUrl 获取画廊某页的直链与页面备链
// , 不建议使用
// , [DownloadPages] 可使用备链自动重试
func FetchPageImageUrl(pageUrl string) (imgUrl string, bakPage string, err error) {
	if err = checkDomain(pageUrl); err != nil {
		return "", "", err
	}
	return fetchPageImageUrl(pageUrl)
}

// DownloadImage 使用图片直链下载
// , 不建议使用
// , [DownloadPages] 可使用备链自动重试
func DownloadImages(imgUrls ...string) (imgDatas [][]byte, err error) {
	for _, url := range imgUrls {
		var data []byte
		data, err = downloadImage(url)
		if err != nil {
			return nil, err
		}
		imgDatas = append(imgDatas, data)
	}
	return imgDatas, nil
}
