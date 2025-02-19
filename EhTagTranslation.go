package EHentai

import (
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

const DATABASE_URL = `https://github.com/EhTagTranslation/Database/releases/latest/download/db.text.json`

// map[namespace]map[<tag>]name
type Database map[string]map[string]string

var database = make(Database)

// TranslateMulti 翻译多个 tag
// , 输入格式应为: namespace:tag
// , 若数据库未初始化, 则返回入参
func TranslateMulti(tags []string) []string {
	if !database.Ok() {
		return tags
	}
	return database.TranslateMulti(tags)
}

// Translate 翻译 tag
// , 输入格式应为: namespace:tag
// , 若数据库未初始化, 则返回入参
func Translate(tag string) string {
	if !database.Ok() {
		return tag
	}
	return database.Translate(tag)
}

func (db *Database) Ok() bool {
	return len(*db) != 0
}

func (db *Database) Init() error {
	data, err := db.Download(DATABASE_URL)
	if err != nil {
		return err
	}

	// tn := time.Now()
	err = db.Unmarshal(data)
	if err != nil {
		return err
	}
	// fmt.Println("database unmarshaled in", time.Since(tn))

	return nil
}

func (db *Database) Download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (db *Database) Unmarshal(data []byte) error {
	jDataArr := gjson.ParseBytes(data).Get("data").Array()

	// 内容索引, data 内为所有的 namespace
	rows := jDataArr[0]
	(*db)["rows"] = make(map[string]string)
	for namespace, values := range rows.Get("data").Map() {
		// namespace 对应的翻译
		(*db)["rows"][namespace] = values.Get("name").String()
		// 创建所有 namespace 对应的 map
		(*db)[namespace] = make(map[string]string)
	}

	// 解析每个 namespace
	for i, data := range jDataArr {
		if i == 0 {
			continue
		}
		go func(data gjson.Result) { // tag 对应的翻译
			namespace := data.Get("namespace").String()
			for tag, value := range data.Get("data").Map() {
				(*db)[namespace][tag] = value.Get("name").String()
			}
		}(data)
	}

	return nil
}

func (db *Database) Info() map[string]int {
	namespacesLen := make(map[string]int)
	for namespace, tags := range *db {
		namespacesLen[namespace] = len(tags)
	}
	return namespacesLen
}

func (db *Database) TranslateMulti(tags []string) []string {
	t := make([]string, len(tags))
	for i, tag := range tags {
		t[i] = db.Translate(tag)
	}
	return t
}

func (db *Database) Translate(tag string) string {
	s := strings.Split(tag, ":")
	if len(s) == 2 {
		if name, ok := (*db)[s[0]][s[1]]; ok {
			return name
		}
	}
	return tag
}
