package EHentai

import (
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
	ErrMismatchPageUrls     = errors.New("mismatch page urls")
	ErrGalleryMetadataError = errors.New("gallery metadata error")
	ErrEmptyCache           = errors.New("empty cache")
	ErrEmptyData            = errors.New("empty data")
	ErrInvalidCacheMetadata = errors.New("invalid cache metadata")
	ErrInvalidPageNum       = errors.New("invalid page number")
	ErrUnknownImageType     = errors.New("unknown image type")
	ErrPageNotCached        = errors.New("page not cached")
)

// GetCache 获取画廊缓存
func GetCache(gId string) *cacheGallery {
	metaPath := filepath.Join(cacheDir, gId, METADATA_FILE_NAME)
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

// CreateCache 覆盖创建画廊缓存元数据
//
// domain 为空时使用 [EXHENTAI_DOMAIN]
//
// pageUrls 需要完整以保证读取失败时直接下载
func CreateCache(domain Domain, gMeta *GalleryMetadata, pageUrls []string) (cg *cacheGallery, err error) {
	if domain == "" {
		domain = EXHENTAI_DOMAIN
	}
	if gMeta.Error != "" {
		return nil, wrapErr(ErrGalleryMetadataError, gMeta.Error)
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

	cgm.Pages = pageUrls

	cgm.Files.Dir = gDir
	cgm.Files.Count = 0
	cgm.Files.Pages = []CachePageInfo{}

	defer func() {
		if err != nil {
			// 创建失败时 清理空文件夹
			_ = os.Remove(gDir)
		}
	}()

	cg = &cacheGallery{meta: cgm}
	err = cg.mkdir()
	if err != nil {
		return nil, err
	}
	err = cg.updateMetadata()
	if err != nil {
		return nil, err
	}

	return cg, err
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
	n = 0 // 写入数
	for _, page := range pages {
		if page.PageNum <= 0 {
			return n, wrapErr(ErrInvalidPageNum, page.PageNum)
		}
		if len(page.Data) == 0 {
			return n, wrapErr(ErrEmptyData, page.PageNum)
		}

		err := os.WriteFile(
			filepath.Join(cg.meta.Files.Dir, itoa(page.Page.PageNum)+"."+page.Type.String()),
			page.Data, 0o644,
		)
		if err != nil {
			return n, err
		}

		cg.metaMu.Lock()
		cg.meta.Files.Pages = append(cg.meta.Files.Pages, CachePageInfo{page.Page.PageNum, int(page.Type), len(page.Data)})
		cg.metaMu.Unlock()

		n++
	}

	cg.updateMetadata() // 更新元数据
	return n, nil
}

// match 匹配缓存页信息
func (cg *cacheGallery) match(pageInfo *CachePageInfo, page *PageData) error {
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

func (cg *cacheGallery) getPagePath(pageInfo *CachePageInfo) string {
	return filepath.Join(cg.meta.Files.Dir, itoa(pageInfo.Num)+"."+ImageType(pageInfo.Type).String())
}

// ReadIter 以迭代器模式读取缓存
func (cg *cacheGallery) ReadIter(pageNums ...int) iter.Seq2[PageData, error] {
	return func(yield func(PageData, error) bool) {
		if cg.meta.Files.Count == 0 { // 无缓存
			yield(PageData{}, ErrEmptyCache)
			return
		}

		// 从元数据中获取页信息
		pageInfos := cg.meta.Files.Pages.Lookup(pageNums)

		for _, pageInfo := range pageInfos {
			page := PageData{Page: UrlToPage(cg.meta.Pages[pageInfo.Num])}
			// 匹配缓存页信息
			err := cg.match(&pageInfo, &page)
			if err != nil {
				if yield(page, err) {
					continue
				}
				return
			}

			path := cg.getPagePath(&pageInfo)
			data, err := os.ReadFile(path)
			if err != nil {
				if yield(page, err) {
					continue
				}
				return
			}
			page.Image = Image{Data: data, Type: ImageType(pageInfo.Type)}

			if !yield(page, nil) {
				return
			}
		}
	}
}

// Read 读取缓存
func (cg *cacheGallery) Read(pageNums ...int) ([]PageData, error) {
	pageDatas := make([]PageData, 0, len(pageNums))

	pageInfos := cg.meta.Files.Pages.Lookup(pageNums)
	for _, pageInfo := range pageInfos {
		page := PageData{Page: UrlToPage(cg.meta.Pages[pageInfo.Num])}
		err := cg.match(&pageInfo, &page)
		if err != nil {
			return nil, err
		}

		path := cg.getPagePath(&pageInfo)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		page.Image = Image{Data: data, Type: ImageType(pageInfo.Type)}

		pageDatas = append(pageDatas, page)
	}

	return pageDatas, nil
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
		return -1
	}
}
