package EHentai

import (
	"testing"
)

func TestCacheMetadata(t *testing.T) {
	gMetaCache = newRamCache[int, metaCache](cacheTimeout)
	metadataCacheEnabled = true
	var err error
	var resp *GalleryMetadataResponse
	var gallery GalleryDetails
	resp, err = PostGalleryMetadata(t.Context(), GIdList{TEST_GALLERY_GID, TEST_GALLERY_GTOKEN})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GMetadata) == 0 {
		t.Fatal("len(resp.GMetadata) == 0")
	}
	gallery, err = fetchGalleryDetails(t.Context(), TEST_GALLERY_URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(gallery.PageUrls) == 0 {
		t.Fatal("len(gallery.PageUrls) == 0")
	}

	dc := DetailsCacheRead(TEST_GALLERY_GID)
	if dc == nil {
		t.Fatal("nil details cache")
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
