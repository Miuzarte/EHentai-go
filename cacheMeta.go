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
	cleanerOnce     = sync.Once{}
	gMetaCache      = newRamCache[int, metaCache](cacheTimeout) // 画廊元数据缓存
)

// MetaCacheRead 从缓存中读取画廊元数据与页链接
//
// gallery 与 pageUrls 不一定同时存在
func MetaCacheRead(gId int) (*metaCache, bool) {
	if !metadataCacheEnabled {
		return nil, false
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
	cleanerOnce.Do(func() {
		// 第一次写入时启动定时清理任务, 之后不会结束
		cacheCleanTimer = time.NewTicker(cacheCleanDur)
		go func() {
			// 每 [cacheCleanDur] 尝试执行一次清理
			// 每次读写会将 [nextCleanTime] 延后
			// 每 4*[cacheCleanDur] 会强制执行一次
			for range cacheCleanTimer.C {
				if len(gMetaCache.m) == 0 {
					continue
				}
				if !metadataCacheEnabled {
					// 缓存在后来被禁用, 此次循环清空全部
					gMetaCache = newRamCache[int, metaCache](cacheTimeout)
					continue
				}
				if nextCleanTime.Before(time.Now()) || time.Since(lastCleanTime) > 4*cacheCleanDur {
					gMetaCache.clean()
					lastCleanTime = time.Now()
					nextCleanTime = lastCleanTime.Add(cacheCleanDur)
				}
			}
		}()
	})

	defer func() {
		// 延后下次清理时间
		nextCleanTime = nextCleanTime.Add(time.Minute * 10)
	}()

	if mc, ok := gMetaCache.get(gId); ok {
		// 已存在 直接更新
		if g != nil {
			mc.gallery = g
		}
		if len(pageUrls) > 0 {
			mc.pageUrls = pageUrls
		}
		if mc.match() {
			return
		}
		// 若不匹配, 重新写入
	}
	gMetaCache.set(gId, &metaCache{g, pageUrls})
}

type metaCache struct {
	gallery  *GalleryMetadata
	pageUrls []string
}

// match 检查元数据与页数量是否匹配
func (mc *metaCache) match() bool {
	if mc.gallery == nil || len(mc.pageUrls) == 0 {
		// 不完整时跳过
		return true
	}

	fileCount, _ := atoi(mc.gallery.FileCount)
	return fileCount == len(mc.pageUrls)
}

type ramCache[K comparable, T any, PT *T] struct {
	timeout time.Duration
	sync.RWMutex
	m map[K]*struct {
		t time.Time
		v PT
	}
}

func newRamCache[K comparable, T any, PT *T](timeout time.Duration) ramCache[K, T, PT] {
	return ramCache[K, T, PT]{
		timeout: timeout,
		m: make(map[K]*struct {
			t time.Time
			v PT
		}),
	}
}

func (rc *ramCache[K, T, PT]) get(k K) (PT, bool) {
	rc.RLock()
	defer rc.RUnlock()
	if v, ok := rc.m[k]; ok {
		if time.Since(v.t) < rc.timeout {
			return v.v, true
		} else {
			delete(rc.m, k)
		}
	}
	return nil, false
}

func (rc *ramCache[K, T, PT]) set(k K, v PT) {
	rc.Lock()
	defer rc.Unlock()
	rc.m[k] = &struct {
		t time.Time
		v PT
	}{time.Now(), v}
}

func (rc *ramCache[K, T, PT]) clean() {
	rc.Lock()
	defer rc.Unlock()
	for k, v := range rc.m {
		if time.Since(v.t) > rc.timeout {
			delete(rc.m, k)
		}
	}
}
