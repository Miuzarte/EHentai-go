package EHentai

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
)

const DATABASE_URL = `https://github.com/EhTagTranslation/Database/releases/latest/download/db.text.json`

// map[namespace]map[<tag>]name
type EhTagDatabase map[string]map[string]string

var ehTagDatabase = make(EhTagDatabase)

func (db *EhTagDatabase) Ok() bool {
	return len(*db) != 0
}

func (db *EhTagDatabase) Free() {
	*db = make(EhTagDatabase)
}

func (db *EhTagDatabase) Init() error {
	data, err := db.Download(DATABASE_URL)
	if err != nil {
		return err
	}

	// tn := time.Now()
	err = db.Unmarshal(data)
	// fmt.Println("database unmarshaled in", time.Since(tn))
	if err != nil {
		return err
	}

	return nil
}

func (db *EhTagDatabase) Download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("empty downloaded content")
	}
	return data, nil
}

func (db *EhTagDatabase) Unmarshal(data []byte) error {
	jDataArr := gjson.ParseBytes(data).Get("data").Array()
	if len(jDataArr) == 0 {
		fs, err := os.Open("EhTagTranslation_dump.txt")
		if err == nil {
			fs.Write(data)
			fs.Close()
		}
		return errors.New("failed to parse json")
	}

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
	wg := sync.WaitGroup{}
	wg.Add(len(jDataArr) - 1)
	for i, data := range jDataArr {
		if i == 0 {
			continue
		}
		go func() { // tag 对应的翻译
			defer wg.Done()
			namespace := data.Get("namespace").String()
			for tag, value := range data.Get("data").Map() {
				(*db)[namespace][tag] = value.Get("name").String()
			}
		}()
	}
	wg.Wait()

	return nil
}

func (db *EhTagDatabase) Info() map[string]int {
	namespacesLen := make(map[string]int)
	for namespace, tags := range *db {
		namespacesLen[namespace] = len(tags)
	}
	return namespacesLen
}

func (db *EhTagDatabase) TranslateMulti(tags []string) []string {
	t := make([]string, len(tags))
	for i, tag := range tags {
		t[i] = db.Translate(tag)
	}
	return t
}

func (db *EhTagDatabase) Translate(tag string) string {
	s := strings.Split(tag, ":")
	if len(s) == 2 {
		if name, ok := (*db)[s[0]][s[1]]; ok {
			return name
		}
	}
	return tag
}
