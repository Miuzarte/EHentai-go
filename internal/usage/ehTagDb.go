package usage

import (
	"log"
	"time"

	ehentai "github.com/Miuzarte/EHentai-go"
)

// 初始化 [EhTagTranslation](github.com/EhTagTranslation/Database) 数据库
func UsageEhTagDb() {
	tStart := time.Now()
	// 在 AMD Ryzen 5600x(6c12t) 上, 解析数据大概耗时 4ms
	// 要更新的话再调用一次
	err := ehentai.InitEhTagDb()
	if err != nil {
		log.Fatalln(err)
	}
	// 总时间包括从 GitHub 下载数据
	log.Printf("InitEhTagDb took %s\n", time.Since(tStart))

	// 释放数据库
	ehentai.FreeEhTagDb()
}
