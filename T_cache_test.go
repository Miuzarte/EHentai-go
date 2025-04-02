package EHentai

import (
	"testing"
	"time"
)

func TestCacheMetadata(t *testing.T) {
	gMetaCache = newRamCache[int, metaCache](cacheTimeout)
	metadataCacheEnabled = true
	var err error
	var resp *GalleryMetadataResponse
	var pageUrls []string
	resp, err = PostGalleryMetadata(t.Context(), GIdList{TEST_GALLERY_GID, TEST_GALLERY_GTOKEN})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GMetadata) == 0 {
		t.Fatal("len(resp.GMetadata) == 0")
	}
	pageUrls, err = fetchGalleryPages(t.Context(), TEST_GALLERY_URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(pageUrls) == 0 {
		t.Fatal("len(pageUrls) == 0")
	}

	mc := MetaCacheRead(TEST_GALLERY_GID)
	if mc == nil {
		t.Fatal("nil meta cache")
	}
	if mc.gallery == nil {
		t.Fatal("nil meta cache.gallery")
	}
	if len(mc.pageUrls) == 0 {
		t.Fatal("empty meta cache.pageUrls")
	}
	if mc.gallery.GId != TEST_GALLERY_GID {
		t.Fatalf("meta cache.gallery.GId != %d\n", TEST_GALLERY_GID)
	}
}

func TestCacheLocalGallery(t *testing.T) {
	defer func() {
		_ = DeleteCache(TEST_GALLERY_GID)
	}()

	autoCacheEnabled = true
	var cache *cacheGallery

	// download [TEST_GALLERY_URL]
	// : [TEST_GALLERY_PAGE_0], [TEST_GALLERY_PAGE_1]
	// (write cache)
	for page, err := range DownloadGalleryIter(t.Context(), TEST_GALLERY_URL, TEST_GALLERY_PAGE_0, TEST_GALLERY_PAGE_1) {
		if err != nil {
			t.Fatal(err)
		}
		t.Log(page.String())
	}

	// 缓存的写入是异步的, 等待元数据更新
	<-time.After(time.Second)
	cache = GetCache(TEST_GALLERY_GID)
	if cache == nil {
		t.Fatal("nil cache")
	}
	// cache pages == 2
	if cache.meta.Files.Count != 2 {
		t.Fatal("cache.meta.Files.Count != 2")
	}
	for i, page := range cache.meta.Files.Pages {
		t.Logf("cache.meta.PageUrls[%d] = %v", i, page)
	}

	// download [TEST_PAGE_URL_0], [TEST_PAGE_URL_1]
	// (write cache)
	for page, err := range DownloadPagesIter(t.Context(), TEST_PAGE_URL_0, TEST_PAGE_URL_1) {
		if err != nil {
			t.Fatal(err)
		}
		t.Log(page.String())
	}

	<-time.After(time.Second)
	cache = GetCache(TEST_GALLERY_GID)
	if cache == nil {
		t.Fatal("nil cache")
	}
	// cache pages == 4
	if cache.meta.Files.Count != 4 {
		t.Fatal("cache.meta.Files.Count != 4")
	}
	for i, page := range cache.meta.Files.Pages {
		t.Logf("cache.meta.PageUrls[%d] = %v", i, page)
	}
}
