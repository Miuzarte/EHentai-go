package cmd

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Miuzarte/EHentai-go"
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/config"
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/errors"
	"github.com/Miuzarte/EHentai-go/internal/utils"
	"github.com/Miuzarte/SimpleLog"
	"github.com/spf13/cobra"
)

var searchLog = SimpleLog.New("[Search]", true, false)

const resultTemplate = //
`{{define "result"}}{{.Title}}
{{Join .Tags ", "}}
{{.Cat}} | {{.Rating}}⭐ | {{.Pages}}
{{Hyperlink .Url}}
{{end}}`

const galleryTemplate = //
`{{define "gallery"}}{{.TitleJpn}}
{{Join .Tags ", "}}
{{.Category}} | {{.Rating}}⭐ | {{.FileCount}} pages | RawSize: {{FmtBytes .FileSize}}
Posted: {{.Posted}} | TorrentCount: {{.TorrentCount}}{{if .ParentGId}}
Parent: {{.ParentGId}} | ParentKey: {{.ParentKey}}{{end}}{{if .FirstGId}}
First: {{.FirstGId}} | FirstKey: {{.FirstKey}}{{end}}
{{end}}`

const torrentTemplate = //
`{{define "torrent"}}{{range .}}{{.Name}}
{{FmtBytesStr .FSize}}
{{.Added}}
{{.Hash}}
{{FmtBytesStr .TSize}}
{{end}}{{end}}`

var tmpl *template.Template

func initTemplate() {
	tmpl = template.New("EHentai-cli")
	tmpl.Funcs(template.FuncMap{
		"Join":      strings.Join,
		"Hyperlink": utils.Hyperlink,
		"FmtBytes":  utils.FormatBytes[int],
		"FmtBytesStr": func(s string) string {
			i, _ := strconv.Atoi(s)
			return utils.FormatBytes(i)
		},
	})

	var err error
	tmpl, err = tmpl.Parse(resultTemplate)
	if err != nil {
		searchLog.Fatalf("failed to parse result template: %v", err)
	}
	tmpl, err = tmpl.Parse(galleryTemplate)
	if err != nil {
		searchLog.Fatalf("failed to parse gallery template: %v", err)
	}
	tmpl, err = tmpl.Parse(torrentTemplate)
	if err != nil {
		searchLog.Fatalf("failed to parse torrent template: %v", err)
	}
}

const searchDesc = "Search for galleries by keyword"

const searchDescLong = searchDesc +
	"\n" + "-d to get detailed information about galleries" +
	"\n" + "-e for EHentai" +
	"\n" + "-x for ExHentai" +
	"\n" + "-t for EHentai tag translation"

var (
	flagDetail           bool
	flagEh               bool
	flagEx               bool
	flagEhTagTranslation bool

	flagCat []string
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: searchDesc,
	Long:  searchDescLong,
	Args:  cobra.MinimumNArgs(1), // 任意数量实现关键字带空格时不需要引号包裹
	PreRunE: func(_ *cobra.Command, _ []string) (err error) {
		err = config.InitConfig()
		if err != nil {
			return
		}
		initTemplate()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		site := config.Search.DefaultSite
		switch {
		case flagEh:
			site = EHentai.EHENTAI_DOMAIN
		case flagEx:
			site = EHentai.EXHENTAI_DOMAIN
		}

		keyword := strings.Join(args, " ")

		cat := config.Search.Cat
		if len(flagCat) != 0 {
			cat, err = EHentai.ParseCategory(flagCat...)
			if err != nil {
				searchLog.Error("failed to parse category: ", err)
				return errors.Handled
			}
		}
		searchLog.Debug("searching site: ", site)
		searchLog.Debug("searching keyword: ", keyword)
		searchLog.Debug("searching categories: ", cat)

		var total int
		if !flagDetail {
			total, err = search(cmd.Context(), site, keyword, cat)
		} else {
			total, err = searchDetail(cmd.Context(), site, keyword, cat)
		}
		if err != nil {
			searchLog.Error("failed to search: ", err)
			return errors.Handled
		}

		searchLog.Infof("%d results in total", total)
		// TODO?: press ENTER for next page, q for quit

		return nil
	},
}

func init() {
	// TODO: viper binding
	searchCmd.Flags().BoolVarP(&flagDetail, "detail", "d", false, "Fetch detailed information about galleries")
	searchCmd.Flags().BoolVarP(&flagEh, "eh", "e", false, "Force use EHentai")
	searchCmd.Flags().BoolVarP(&flagEx, "ex", "x", false, "Force use ExHentai")
	searchCmd.MarkFlagsMutuallyExclusive("eh", "ex")
	searchCmd.Flags().BoolVarP(&flagEhTagTranslation, "eh-tag", "t", false, "Force use EHentai tag translation")
	searchCmd.Flags().StringSliceVar(&flagCat, "cat", nil, "Specify categories (e.g. --cat DOUJINSHI,MANGA)")
	rootCmd.AddCommand(searchCmd)
}

// 搜索完再初始化数据库,
// 以免无结果时浪费时间
func initEhTagDb() (err error) {
	// TODO: database cache
	if config.Search.UseEhTagTranslation || flagEhTagTranslation {
		searchLog.Debug("init EhTagTranslation database...")
		tn := time.Now()
		err = EHentai.InitEhTagDB()
		if err != nil {
			return
		}
		searchLog.Debugf("init EhTagTranslation database took %s", time.Since(tn))
	}
	return nil
}

func search(ctx context.Context, site EHentai.Domain, keyword string, categories ...EHentai.Category) (total int, err error) {
	var results EHentai.FSearchResults
	switch site {
	case EHentai.EHENTAI_DOMAIN:
		total, results, err = EHentai.EHSearch(ctx, keyword, categories...)
	case EHentai.EXHENTAI_DOMAIN:
		total, results, err = EHentai.ExHSearch(ctx, keyword, categories...)
	}
	if err != nil {
		return 0, err
	}
	if err := initEhTagDb(); err != nil {
		searchLog.Error("failed to init EhTagTranslation database: ", err)
	}

	for i := range results {
		// 倒序打印
		sn := len(results) - i
		result := results[sn-1]

		// 汉化 tags
		if config.Search.UseEhTagTranslation || flagEhTagTranslation {
			result.Tags = EHentai.TranslateMulti(result.Tags)
		}

		fmt.Printf("\x1b[96m%d.\x1b[m\n", sn)
		if err := tmpl.ExecuteTemplate(os.Stdout, "result", result); err != nil {
			searchLog.Panicf("failed to execute result template: %v", err)
		}
	}

	return
}

func searchDetail(ctx context.Context, site EHentai.Domain, keyword string, categories ...EHentai.Category) (total int, err error) {
	var galleries EHentai.GalleryMetadatas
	switch site {
	case EHentai.EHENTAI_DOMAIN:
		total, galleries, err = EHentai.EHSearchDetail(ctx, keyword, categories...)
	case EHentai.EXHENTAI_DOMAIN:
		total, galleries, err = EHentai.ExHSearchDetail(ctx, keyword, categories...)
	}
	if err != nil {
		return 0, err
	}
	if err := initEhTagDb(); err != nil {
		searchLog.Error("failed to init EhTagTranslation database: ", err)
	}

	for i := range galleries {
		// 倒序打印
		sn := len(galleries) - i
		gallery := galleries[sn-1]

		fmt.Printf("\x1b[96m%d.\x1b[m\n", sn)

		if gallery.Error != "" {
			fmt.Println(gallery.Error)
			continue
		}

		// 格式化发布时间
		postedTs, _ := strconv.Atoi(gallery.Posted)
		if postedTs != 0 {
			posted := time.Unix(int64(postedTs), 0)
			gallery.Posted = posted.Format("2006-01-02 15:04:05")
		}
		// 汉化 tags
		if config.Search.UseEhTagTranslation || flagEhTagTranslation {
			gallery.Tags = EHentai.TranslateMulti(gallery.Tags)
		}

		// 填充模板
		if err := tmpl.ExecuteTemplate(os.Stdout, "gallery", gallery); err != nil {
			searchLog.Fatalf("failed to execute gallery template: %v", err)
		}
		url := fmt.Sprintf("https://%s/g/%d/%s/", site, gallery.GId, gallery.Token)
		// 列出所有种子
		if config.Search.ShowTorrentDetails {
			if err := tmpl.ExecuteTemplate(os.Stdout, "torrent", gallery.Torrents); err != nil {
				searchLog.Fatalf("failed to execute torrent template: %v", err)
			}
		}
		// 详细信息中没有 url, 另外输出
		fmt.Println(utils.Hyperlink(url))
	}

	return
}
