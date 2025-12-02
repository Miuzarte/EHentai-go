package EHentai

import (
	"sync"
	"time"
)

var (
	cacheTimeout    = time.Hour // 缓存过期时间
	cacheCleanDur   = time.Hour // 每次尝试清理的间隔
	lastCleanTime   = time.Now()
	nextCleanTime   = lastCleanTime.Add(cacheCleanDur) // 下次清理时间, 会被读写操作延后
	cacheCleanTimer *time.Ticker
	cleanerOnce     sync.Once

	gDetailsCache = newRamCache[int, GalleryDetails](cacheTimeout) // 画廊详情缓存 (来自网页)
	gMetaCache    = newRamCache[int, metaCache](cacheTimeout)      // 画廊元数据缓存 (来自 api)
)

func startupCleaner() {
	// 第一次写入时启动定时清理任务, 之后不会结束
	cacheCleanTimer = time.NewTicker(cacheCleanDur)
	go func() {
		// 每 [cacheCleanDur] 尝试执行一次清理
		// 每次读写会将 [nextCleanTime] 延后
		// 每 4*[cacheCleanDur] 会强制执行一次
		for range cacheCleanTimer.C {
			if len(gDetailsCache.m) == 0 && len(gMetaCache.m) == 0 {
				continue
			}
			if !metadataCacheEnabled {
				// 缓存在后来被禁用, 此次循环清空全部
				gDetailsCache.reset()
				gMetaCache.reset()
				continue
			}
			if nextCleanTime.Before(time.Now()) || time.Since(lastCleanTime) > 4*cacheCleanDur {
				gDetailsCache.clean()
				gMetaCache.clean()
				lastCleanTime = time.Now()
				nextCleanTime = lastCleanTime.Add(cacheCleanDur)
			}
		}
	}()
}

func DetailsCacheRead(gId int) *GalleryDetails {
	if !metadataCacheEnabled {
		return nil
	}

	defer func() {
		// 延后下次清理时间
		nextCleanTime = nextCleanTime.Add(time.Minute * 10)
	}()

	return gDetailsCache.get(gId)
}

func detailsCacheWrite(gId int, g GalleryDetails) {
	if !metadataCacheEnabled {
		return
	}
	cleanerOnce.Do(startupCleaner)

	defer func() {
		// 延后下次清理时间
		nextCleanTime = nextCleanTime.Add(time.Minute * 10)
	}()

	gDetailsCache.set(gId, &g)
}

// MetaCacheRead 从缓存中读取画廊元数据与页链接
//
// gallery 与 pageUrls 不一定同时存在
//
// TODO: cache switch to [gDetailsCache]
func MetaCacheRead(gId int) *metaCache {
	if !metadataCacheEnabled {
		return nil
	}

	defer func() {
		// 延后下次清理时间
		nextCleanTime = nextCleanTime.Add(time.Minute * 10)
	}()

	return gMetaCache.get(gId)
}

// metaCacheWrite 写入元数据与页链接到缓存
//
// 写入、清理操作由内部维护
func metaCacheWrite(gId int, g *GalleryMetadata, pageUrls []string) {
	if !metadataCacheEnabled {
		return
	}
	cleanerOnce.Do(startupCleaner)

	defer func() {
		// 延后下次清理时间
		nextCleanTime = nextCleanTime.Add(time.Minute * 10)
	}()

	if mc := gMetaCache.get(gId); mc != nil {
		// 已存在 直接更新
		if g != nil {
			mc.gallery = g
		}
		if len(pageUrls) > 0 {
			mc.pageUrls = pageUrls
		}
		if mc.gallery == nil || len(mc.pageUrls) == 0 {
			return
		} else if fileCount, _ := atoi(mc.gallery.FileCount); fileCount == len(mc.pageUrls) {
			return
		}
		// 若不匹配, 重新写入
	}
	gMetaCache.set(gId, &metaCache{g, pageUrls})
}

type (
	metaCache struct {
		gallery  *GalleryMetadata
		pageUrls []string
	}

	genericCacheData[T any] struct {
		t time.Time
		v *T
	}

	ramCache[K comparable, T any] struct {
		sync.RWMutex
		timeout time.Duration
		m       map[K]genericCacheData[T]
	}
)

func newRamCache[K comparable, T any](timeout time.Duration) ramCache[K, T] {
	return ramCache[K, T]{
		timeout: timeout,
		m:       map[K]genericCacheData[T]{},
	}
}

func (rc *ramCache[K, T]) get(k K) *T {
	rc.RLock()
	defer rc.RUnlock()
	if v, ok := rc.m[k]; ok {
		if time.Since(v.t) < rc.timeout {
			return v.v
		} else {
			delete(rc.m, k)
		}
	}
	return nil
}

func (rc *ramCache[K, T]) set(k K, v *T) {
	rc.Lock()
	defer rc.Unlock()
	rc.m[k] = genericCacheData[T]{time.Now(), v}
}

func (rc *ramCache[K, T]) clean() {
	rc.Lock()
	defer rc.Unlock()
	for k, v := range rc.m {
		if time.Since(v.t) > rc.timeout {
			delete(rc.m, k)
		}
	}
}

func (rc *ramCache[K, T]) reset() {
	rc.Lock()
	defer rc.Unlock()
	rc.m = map[K]genericCacheData[T]{}
}
