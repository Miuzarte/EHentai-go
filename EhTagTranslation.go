package EHentai

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"unsafe"

	"github.com/tidwall/gjson"
)

const EHTAG_DATABASE_URL = `https://github.com/EhTagTranslation/Database/releases/latest/download/db.text.json`

// map[namespace]map[<tag>]name
type EhTagDatabase map[string]map[string]string

var ehTagDatabase EhTagDatabase

func (db *EhTagDatabase) Ok() bool {
	return *db != nil
}

func (db *EhTagDatabase) Free() {
	*db = nil
}

func (db *EhTagDatabase) Init() error {
	resp, err := http.Get(EHTAG_DATABASE_URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// tn := time.Now()
	// 切片不会发生修改
	// 解析失败时保存 json 到文件
	err = db.unmarshal(unsafe.String(unsafe.SliceData(data), len(data)))
	// fmt.Println("database unmarshaled in", time.Since(tn))
	if err != nil {
		return err
	}

	return nil
}

func (db *EhTagDatabase) doUnmarshal(data string) (int, error) {
	datasArr := gjson.Parse(data).Get("data").Array()
	if len(datasArr) == 0 {
		return 0, errors.New("empty data array")
	}
	*db = make(EhTagDatabase, len(datasArr)) // 包括 rows

	// 内容索引, data 内为所有的 namespace
	rows := datasArr[0]
	rowData := rows.Get("data").Map()
	(*db)["rows"] = make(map[string]string, len(rowData))
	for namespace, values := range rowData {
		(*db)["rows"][namespace] = values.Get("name").String()
	}

	// 解析每个 namespace
	wg := sync.WaitGroup{}
	for i, data := range datasArr {
		if i == 0 {
			continue
		}
		namespace := data.Get("namespace").String()
		namespaceData := data.Get("data").Map()
		m := make(map[string]string, len(namespaceData))
		(*db)[namespace] = m
		wg.Go(func() {
			for tag, value := range namespaceData {
				m[tag] = value.Get("name").String()
			}
		})
	}
	wg.Wait()

	return len(datasArr), nil
}

// Unmarshal 解析 json 到数据库
func (db *EhTagDatabase) Unmarshal(data string) error {
	_, err := db.doUnmarshal(data)
	return err
}

// unmarshal 解析 json 到数据库, 失败时保存 json 到文件
func (db *EhTagDatabase) unmarshal(data string) error {
	n, err := db.doUnmarshal(data)
	if err != nil && n == 0 {
		_ = os.WriteFile("EhTagTranslation_dump.txt", []byte(data), 0o644)
		return fmt.Errorf("failed to parse json, tag db dumped: %w", err)
	}
	return err
}

// Info 返回数据库统计信息
func (db *EhTagDatabase) Info() map[string]int {
	if !db.Ok() {
		return nil
	}
	namespacesLen := make(map[string]int, len(*db))
	for namespace, tags := range *db {
		namespacesLen[namespace] = len(tags)
	}
	return namespacesLen
}

func (db *EhTagDatabase) TranslateTags(tags []Tag) []Tag {
	if !db.Ok() {
		return tags
	}
	t := make([]Tag, len(tags))
	for i := range tags {
		t[i] = db.TranslateTag(tags[i])
	}
	return t
}

func (db *EhTagDatabase) TranslateTag(tag Tag) Tag {
	if !db.Ok() {
		return tag
	}
	if ns, ok := (*db)[tag.Namespace]; ok {
		tag.Namespace = (*db)["rows"][tag.Namespace]
		if name, ok := ns[tag.Name]; ok {
			tag.Name = name
		}
	}
	return tag
}

func (db *EhTagDatabase) TranslateMulti(tags []string) []string {
	if !db.Ok() {
		return tags
	}
	t := make([]string, len(tags))
	for i, tag := range tags {
		t[i] = db.Translate(tag)
	}
	return t
}

func (db *EhTagDatabase) Translate(tag string) string {
	if !db.Ok() {
		return tag
	}
	s := strings.Split(tag, ":")
	if len(s) == 2 {
		if ns, ok := (*db)[s[0]]; ok {
			if name, ok := ns[s[1]]; ok {
				return name
			}
		}
	}
	return tag
}
