package cmd

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/Miuzarte/EHentai-go"
	"github.com/Miuzarte/EHentai-go/internal/env"
)

// 搜索完再初始化数据库
func initEhTagDb() (err error) {
	if !config.Search.EhTagTranslation {
		return nil
	}

	// 尝试获取缓存
	data, _ := readEhTagDbCache()
	if len(data) == 0 {
		searchLog.Debug("downloading EhTagTranslation database from GitHub")
		var resp *http.Response
		resp, err = http.Get(EHentai.EHTAG_DATABASE_URL)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		// 写缓存
		if err := writeEhTagDbCache(string(data)); err != nil {
			searchLog.Error("failed to write EhTagTranslation database cache: ", err)
		}
	} else {
		searchLog.Debug("EhTagTranslation database cache hit")
	}

	tn := time.Now()
	err = EHentai.UnmarshalEhTagDb(unsafe.String(unsafe.SliceData(data), len(data)))
	if err != nil {
		return
	}
	searchLog.Debugf("unmarshal EhTagTranslation database took %s", time.Since(tn))

	return nil
}

func writeEhTagDbCache(data string) (err error) {
	f := filepath.Join(env.XDir, "ehTagDB_"+strconv.FormatInt(time.Now().Unix(), 10)+".json")
	return os.WriteFile(f, []byte(data), 0o644)
}

func readEhTagDbCache() (data []byte, err error) {
	filename, t, err := findEhTagDbCache()
	if err != nil {
		return
	}
	if time.Since(t) > 24*time.Hour {
		// 超过 24 小时删除
		os.Remove(filename)
		err = os.ErrNotExist
		return
	}
	return os.ReadFile(filename)
}

func findEhTagDbCache() (path string, t time.Time, err error) {
	files, err := filepath.Glob(filepath.Join(env.XDir, "ehTagDB_*.json"))
	if err != nil {
		if err == filepath.ErrBadPattern {
			panic(err)
		}
		return
	}
	if len(files) == 0 {
		err = os.ErrNotExist
		return
	}

	slices.Sort(files)
	slices.Reverse(files) // 新到旧

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		// ehTagDB_1699999999
		i := strings.Index(name, "_")
		if i == -1 {
			continue
		}
		timestamp, err := strconv.ParseInt(name[i+1:], 10, 64)
		if err != nil {
			continue
		}
		return file, time.Unix(timestamp, 0), nil
	}
	return "", time.Time{}, os.ErrNotExist
}
