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
	"github.com/Miuzarte/EHentai-go/internal/utils"
	"github.com/Miuzarte/SimpleLog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	// TODO: read custom template dynamically
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

const searchDescLong = searchDesc // TODO

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: searchDesc,
	Long:  searchDescLong,
	Args:  cobra.MinimumNArgs(1), // 任意数量实现关键字带空格时不需要引号包裹
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		initTemplate()
		searchFlagChanged(cmd.Flags())
		return initConfig(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		keyword := strings.Join(args, " ")

		cat := config.Search.category
		searchLog.Debug("searching site: ", config.Search.Site)
		searchLog.Debug("searching keyword: ", keyword)
		searchLog.Debug("searching categories: ", cat)

		var total int
		if !config.Search.Detail {
			total, err = search(cmd.Context(), config.Search.Site, keyword, cat)
		} else {
			total, err = searchDetail(cmd.Context(), config.Search.Site, keyword, cat)
		}
		if err != nil {
			searchLog.Error("failed to search: ", err)
			return ErrHandled
		}

		searchLog.Infof("%d results in total", total)
		// TODO?: press ENTER for next page, q for quit

		return nil
	},
}

const (
	FLAG_EH   = "eh"
	FLAG_EX   = "ex"
	FLAG_SITE = "site"

	FLAG_EH_TAG         = "eh-tag"
	FLAG_CAT            = "cat"
	FLAG_DETAIL         = "detail"
	FLAG_TORRENT_DETAIL = "torrent-detail"
)

func init() {
	searchCmd.Flags().BoolP(FLAG_EH, "e", false, "use e-hentai.org")
	searchCmd.Flags().BoolP(FLAG_EX, "x", false, "use exhentai.org")
	searchCmd.Flags().String(FLAG_SITE, EHentai.EHENTAI_DOMAIN, "specify site by string (e.g. --site exhentai.org)")
	searchCmd.MarkFlagsMutuallyExclusive(FLAG_EH, FLAG_EX, FLAG_SITE)

	searchCmd.Flags().BoolP(FLAG_EH_TAG, "t", false, "enable EhTagTranslation")
	searchCmd.Flags().StringSlice(FLAG_CAT, nil, "specify categories (e.g. --cat DOUJINSHI,MANGA)")
	searchCmd.Flags().BoolP(FLAG_DETAIL, "d", false, "fetch detailed information about galleries")
	searchCmd.Flags().Bool(FLAG_TORRENT_DETAIL, false, "show torrents while flag detail is set")

	rootCmd.AddCommand(searchCmd)
}

func searchFlagChanged(searchFlags *pflag.FlagSet) {
	if searchFlags.Changed(FLAG_EH) {
		eh, _ := searchFlags.GetBool(FLAG_EH)
		if eh {
			viper.Set("search.site", EHentai.EHENTAI_DOMAIN)
		}
	}
	if searchFlags.Changed(FLAG_EX) {
		ex, _ := searchFlags.GetBool(FLAG_EX)
		if ex {
			viper.Set("search.site", EHentai.EXHENTAI_DOMAIN)
		}
	}
	if searchFlags.Changed(FLAG_SITE) {
		site, _ := searchFlags.GetString(FLAG_SITE)
		viper.Set("search.site", site)
	}
	if searchFlags.Changed(FLAG_EH_TAG) {
		ehTag, _ := searchFlags.GetBool(FLAG_EH_TAG)
		viper.Set("search.ehTagTranslation", ehTag)
	}
	if searchFlags.Changed(FLAG_CAT) {
		cat, _ := searchFlags.GetStringSlice(FLAG_CAT)
		viper.Set("search.category", cat)
	}
	if searchFlags.Changed(FLAG_DETAIL) {
		detail, _ := searchFlags.GetBool(FLAG_DETAIL)
		viper.Set("search.detail", detail)
	}
	if searchFlags.Changed(FLAG_TORRENT_DETAIL) {
		torrentDetail, _ := searchFlags.GetBool(FLAG_TORRENT_DETAIL)
		viper.Set("search.torrentDetail", torrentDetail)
	}
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
	if err := initEhTagDB(); err != nil {
		searchLog.Error("failed to init EhTagTranslation database: ", err)
	}

	for i := range results {
		// 倒序打印
		sn := len(results) - i
		result := results[sn-1]

		// 汉化 tags
		if config.Search.EhTagTranslation {
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
	if err := initEhTagDB(); err != nil {
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
		if config.Search.EhTagTranslation {
			gallery.Tags = EHentai.TranslateMulti(gallery.Tags)
		}

		// 填充模板
		if err := tmpl.ExecuteTemplate(os.Stdout, "gallery", gallery); err != nil {
			searchLog.Fatalf("failed to execute gallery template: %v", err)
		}
		url := fmt.Sprintf("https://%s/g/%d/%s/", site, gallery.GId, gallery.Token)
		// 列出所有种子
		if config.Search.TorrentDetail {
			if err := tmpl.ExecuteTemplate(os.Stdout, "torrent", gallery.Torrents); err != nil {
				searchLog.Fatalf("failed to execute torrent template: %v", err)
			}
		}
		// 详细信息中没有 url, 另外输出
		fmt.Println(utils.Hyperlink(url))
	}

	return
}
