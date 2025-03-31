package EHentai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// TODO: 在 [os.Root] 得到完善后改用

const (
	DEFAULT_CACHE_DIR  = `EHentaiCache`
	METADATA_FILE_NAME = `metadata`
)

var (
	ErrFailedToGetGalleryMetadata = errors.New("failed to get gallery metadata")
	ErrMismatchPageUrls           = errors.New("mismatch page urls")
	ErrGalleryMetadataError       = errors.New("gallery metadata error")
	ErrEmptyCache                 = errors.New("empty cache")
	ErrEmptyData                  = errors.New("empty data")
	ErrInvalidCacheMetadata       = errors.New("invalid cache metadata")
	ErrInvalidPageNum             = errors.New("invalid page number")
	ErrUnknownImageType           = errors.New("unknown image type")
	ErrPageNotCached              = errors.New("page not cached")
)

// GetCache 获取画廊缓存
func GetCache(gId int) *cacheGallery {
	metaPath := filepath.Join(cacheDir, itoa(gId), METADATA_FILE_NAME)
	f, err := os.OpenFile(metaPath, os.O_RDONLY, 0o644)
	if err != nil {
		return nil
	}
	defer f.Close()

	meta := &CacheGalleryMetadata{}
	dec := json.NewDecoder(f)
	// dec.DisallowUnknownFields()
	if err := dec.Decode(meta); err != nil {
		return nil
	}

	return &cacheGallery{meta: meta}
}

// CreateCacheFromUrl 从画廊 url 覆盖创建缓存
func CreateCacheFromUrl(gUrl string) (cache *cacheGallery, err error) {
	domain := urlGetDomain(gUrl)
	gu := UrlToGallery(gUrl)

	var gallery *GalleryMetadata
	var pageUrls []string

	// 获取画廊元数据与页链接 尝试缓存
	gMeta := MetaCacheRead(gu.GalleryId)
	if gMeta != nil {
		gallery = gMeta.gallery
		pageUrls = gMeta.pageUrls
	}

	if gallery == nil {
		resp, err := PostGalleryMetadata(context.Background(), gu)
		if err != nil {
			return nil, err
		}
		if len(resp.GMetadata) == 0 {
			return nil, wrapErr(ErrFailedToGetGalleryMetadata, nil)
		}
		if resp.GMetadata[0].Error != "" {
			return nil, wrapErr(ErrGalleryMetadataError, resp.GMetadata[0].Error)
		}
		gallery = &resp.GMetadata[0]
	}
	if len(pageUrls) == 0 {
		pageUrls, err = fetchGalleryPages(context.Background(), gUrl)
		if err != nil {
			return nil, err
		}
	}

	return CreateCache(domain, gallery, pageUrls)
}

// CreateCache 覆盖创建画廊缓存元数据
//
// domain 为空时使用 [EXHENTAI_DOMAIN]
//
// pageUrls 需要完整以保证读取失败时直接下载, 不提供时尝试从缓存中读取
func CreateCache(domain Domain, gMeta *GalleryMetadata, pageUrls []string) (cache *cacheGallery, err error) {
	if domain == "" {
		domain = EXHENTAI_DOMAIN
	}
	if gMeta.Error != "" {
		return nil, wrapErr(ErrGalleryMetadataError, gMeta.Error)
	}

	if len(pageUrls) == 0 {
		meta := MetaCacheRead(gMeta.GId)
		if meta == nil {
			return nil, wrapErr(ErrNoPageUrlProvided, nil)
		}
		pageUrls = meta.pageUrls
	} else {
		// 检查 pageUrls 是否来自同一画廊
		gIds := collectGIds(pageUrls)
		if len(gIds) != 1 {
			return nil, wrapErr(ErrTooManyGIds, gIds)
		}
	}

	fileCount, _ := atoi(gMeta.FileCount)
	if fileCount != len(pageUrls) {
		// pageUrls 需要完整以保证读取失败时直接下载
		return nil, wrapErr(ErrMismatchPageUrls, fmt.Sprintf("gMeta.FileCount %d != len(pageUrls) %d", fileCount, len(pageUrls)))
	}

	gId := itoa(gMeta.GId)
	gDir := filepath.Join(cacheDir, gId)

	cgm := &CacheGalleryMetadata{}

	cgm.Url = "https://" + domain + "/g/" + gId + "/" + gMeta.Token
	cgm.Gallery = *gMeta

	// cgm.PageUrls = pageUrls
	cgm.PageUrls = make(map[string]string, len(pageUrls))
	for i, pageUrl := range pageUrls {
		cgm.PageUrls[itoa(i+1)] = pageUrl
	}

	cgm.Files.Dir = gDir
	cgm.Files.Count = 0
	cgm.Files.Pages = []CachePageInfo{}

	defer func() {
		if err != nil {
			// 创建失败时 清理空文件夹
			_ = os.Remove(gDir)
		}
	}()

	cache = &cacheGallery{meta: cgm}
	err = cache.mkdir()
	if err != nil {
		return nil, err
	}
	err = cache.updateMetadata()
	if err != nil {
		return nil, err
	}

	return cache, err
}

func DeleteCache(gId int) (err error) {
	cg := GetCache(gId)
	if cg == nil {
		return nil
	}

	// 删除所有缓存的页
	cg.DeletePages(nil)

	// 删除元数据文件
	err = os.Remove(filepath.Join(cg.meta.Files.Dir, METADATA_FILE_NAME))
	if err != nil {
		return err
	}

	// 删除缓存目录
	// 非空时无法删除
	// 不使用 [os.RemoveAll]
	err = os.Remove(cg.meta.Files.Dir)
	if err != nil {
		return err
	}

	return nil
}

// cacheGallery 画廊缓存实例
type cacheGallery struct {
	meta   *CacheGalleryMetadata
	metaMu sync.Mutex // 保护 [CacheGalleryMetadata].Files.Pages
}

func (cg *cacheGallery) mkdir() error {
	return os.MkdirAll(cg.meta.Files.Dir, 0o755)
}

func (cg *cacheGallery) updateMetadata() error {
	// 去重 排序 计数
	cg.meta.Files.Pages = removeDuplication(cg.meta.Files.Pages)
	slices.SortFunc(
		cg.meta.Files.Pages,
		func(a, b CachePageInfo) int {
			switch {
			case a.Num < b.Num:
				return -1
			case a.Num > b.Num:
				return 1
			default:
				return 0
			}
		},
	)
	cg.meta.Files.Count = len(cg.meta.Files.Pages)

	// 具体验证文件数量
	if cg.meta.Files.Count > 0 {
		pageExts := make(set[string])
		for _, pageInfo := range cg.meta.Files.Pages {
			pageExts[ImageType(pageInfo.Type).String()] = struct{}{}
		}
		exts := make([]string, 0, len(pageExts))
		for ext := range pageExts {
			exts = append(exts, ext)
		}
		dir, err := os.ReadDir(cg.meta.Files.Dir)
		if err != nil {
			return err
		}
		pageFiles := dirLookupExt(dir, exts...)
		if len(pageFiles) != cg.meta.Files.Count {
			return wrapErr(ErrInvalidCacheMetadata, fmt.Sprintf("len(pageFiles) %d != cg.meta.Files.Pages %d", len(pageFiles), cg.meta.Files.Count))
		}
	}

	// TODO: 验证文件大小

	f, err := os.OpenFile(filepath.Join(cg.meta.Files.Dir, METADATA_FILE_NAME), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	enc.SetEscapeHTML(false)
	return enc.Encode(cg.meta)
}

// Write 将画廊图片写入缓存
func (cg *cacheGallery) Write(pages ...PageData) (n int, err error) {
	defer cg.updateMetadata() // 更新元数据

	for _, page := range pages {
		if page.PageNum <= 0 {
			return n, wrapErr(ErrInvalidPageNum, page.PageNum)
		}
		if len(page.Data) == 0 {
			return n, wrapErr(ErrEmptyData, page.PageNum)
		}

		pageInfo := CachePageInfo{page.Page.PageNum, int(page.Type), len(page.Data)}

		err := os.WriteFile(cg.getPagePath(pageInfo), page.Data, 0o644)
		if err != nil {
			return n, err
		}

		cg.metaMu.Lock()
		cg.meta.Files.Pages.append(pageInfo)
		cg.meta.Files.Count = len(cg.meta.Files.Pages)
		cg.metaMu.Unlock()

		n++
	}

	return n, nil
}

// match 匹配缓存页信息
func (cg *cacheGallery) match(pageInfo CachePageInfo, page PageData) error {
	// 画廊与页码
	if page.GalleryId != cg.meta.Gallery.GId || page.PageNum != pageInfo.Num {
		return wrapErr(ErrInvalidCacheMetadata, nil)
	}

	// 页信息
	if pageInfo.Num <= 0 {
		return wrapErr(ErrInvalidPageNum, pageInfo.Num)
	}
	if pageInfo.Type <= 0 {
		return wrapErr(ErrUnknownImageType, pageInfo.Type)
	}
	if pageInfo.Len <= 0 {
		return wrapErr(ErrPageNotCached, pageInfo.Num)
	}

	return nil
}

func (cg *cacheGallery) getPagePath(pageInfo CachePageInfo) string {
	return filepath.Join(cg.meta.Files.Dir, itoa(pageInfo.Num)+"."+ImageType(pageInfo.Type).String())
}

func (cg *cacheGallery) readOne(pageInfo CachePageInfo) (pageData PageData, err error) {
	pageData = PageData{Page: UrlToPage(cg.meta.PageUrls[itoa(pageInfo.Num)])} // 根据页码获取 pageUrl

	if cg.meta.Files.Count == 0 {
		return pageData, wrapErr(ErrEmptyCache, nil)
	}

	err = cg.match(pageInfo, pageData)
	if err != nil {
		return
	}

	path := cg.getPagePath(pageInfo)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = wrapErr(ErrPageNotCached, pageData.PageNum)
		}
		return
	}
	pageData.Image = Image{Data: data, Type: ImageType(pageInfo.Type)}

	return pageData, nil
}

// ReadOne 读取缓存
//
// 指定的 pageNum 未缓存时返回 [ErrPageNotCached]
func (cg *cacheGallery) ReadOne(pageNum int) (page PageData, err error) {
	pageInfo := cg.meta.Files.Pages.Get(pageNum)
	return cg.readOne(pageInfo)
}

// ReadIter 以迭代器模式读取缓存
//
// 不提供 pageNums 时读取所有已缓存的页
func (cg *cacheGallery) ReadIter(pageNums []int) iter.Seq2[PageData, error] {
	return func(yield func(PageData, error) bool) {
		// 从元数据中获取页信息
		pageInfos := cg.meta.Files.Pages.Lookup(pageNums)

		for _, pageInfo := range pageInfos {
			if !yield(cg.readOne(pageInfo)) {
				return
			}
		}
	}
}

// Read 读取缓存
//
// 不提供 pageNums 时读取所有已缓存的页
//
// pageNums 中遇到未缓存的页时返回 [ErrPageNotCached]
func (cg *cacheGallery) Read(pageNums []int) ([]PageData, error) {
	pageDatas := make([]PageData, 0, len(pageNums))

	pageInfos := cg.meta.Files.Pages.Lookup(pageNums)
	for _, pageInfo := range pageInfos {
		page, err := cg.readOne(pageInfo)
		if err != nil {
			return nil, err
		}
		pageDatas = append(pageDatas, page)
	}

	return pageDatas, nil
}

// ReadByPageUrl 通过 pageUrl 读取缓存
func (cg *cacheGallery) ReadByPageUrl(url string) (pageData PageData, err error) {
	pu := UrlToPage(url)
	pageInfo := cg.meta.Files.Pages.Get(pu.PageNum)
	return cg.readOne(pageInfo)
}

// ReadByPageUrl 通过 pageUrls 读取缓存
//
// pageUrls 不可为空
//
// 为了方便起见,
// len(pageDatas) == len(pageUrls) 且顺序一致,
// 是否命中缓存只需检查 len(pageDatas[i].Image.Data)
func (cg *cacheGallery) ReadByPageUrls(pageUrls []string) (pageDatas []PageData) {
	if len(pageUrls) == 0 {
		return nil
	}

	// 获取页码
	pageNums := make([]int, len(pageUrls))
	for i := range pageUrls {
		pageNums[i] = UrlToPage(pageUrls[i]).PageNum
	}

	// 查缓存
	pageInfos := cg.meta.Files.Pages.Lookup(pageNums)
	pageDatas = make([]PageData, len(pageUrls))
	for i := range pageInfos {
		if pageInfos[i].Len == 0 {
			continue
		}
		pageDatas[i], _ = cg.readOne(pageInfos[i])
	}

	return
}

// DeletePages 删除缓存的页
//
// 不提供 pageNums 时删除所有已缓存的页
func (cg *cacheGallery) DeletePages(pageNums []int) (n int) {
	defer cg.updateMetadata() // 更新元数据

	pageInfos := cg.meta.Files.Pages.Lookup(pageNums)

	for _, pageInfo := range pageInfos {
		err := os.Remove(cg.getPagePath(pageInfo))
		if err == nil {
			n++ // 记录删除成功的数量
		}

		cg.metaMu.Lock()
		cg.meta.Files.Pages.del(pageInfo.Num)
		cg.meta.Files.Count = len(cg.meta.Files.Pages)
		cg.metaMu.Unlock()
	}

	return n
}

func (cg *cacheGallery) State() CacheState {
	if cg == nil || cg.meta == nil {
		return CACHE_STATE_NONE
	}
	total := cg.meta.Files.Count
	have, _ := atoi(cg.meta.Gallery.FileCount)
	switch {
	case total < have:
		return CACHE_STATE_PARTIAL
	case total == have:
		return CACHE_STATE_FULL
	default:
		return CACHE_STATE_UNKNOWN
	}
}
