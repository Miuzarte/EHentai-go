package cmd

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/Miuzarte/EHentai-go"
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/bar"
	"github.com/Miuzarte/EHentai-go/internal/env"
	"github.com/Miuzarte/EHentai-go/internal/utils"
	"github.com/Miuzarte/SimpleLog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/vbauerster/mpb/v8"
)

var dlLog = SimpleLog.New("[Download]", true, false)

const downloadDesc = "Download gallery or pages, using slice syntax(allow negative) to specify page range"

const downloadDescLong = downloadDesc +
	"\n" + "Support multiple urls" +
	"\n" + "Help for slice syntax: " +
	"\n" + "[start:end] / [:end] / [start:] / [index]" +
	"\n" + "start is inclusive, end is exclusive" +
	"\n" + "negative index is allowed" +
	"\n" + "e.g." +
	"\n" + "\"-p [3:-1]\": start from the 3rd page and without last page"

var downloadCmd = &cobra.Command{
	Use:   "download <gallery/page url>... [-p <page range>]",
	Short: downloadDesc,
	Long:  downloadDescLong,
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		downloadFlagChanged(cmd.Flags())
		return initConfig(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// 解析切片
		pageRange, _ := cmd.Flags().GetString(FLAG_PAGES)
		var sliceSyntaxes utils.SliceSyntaxes
		if pageRange != "" {
			sliceSyntaxes, err = utils.ParseSliceSyntaxes(pageRange)
			if err != nil {
				return
			}
		}

		// 收集所有链接
		var galleries []string
		var pages []string
		for i, arg := range args {
			var u *url.URL
			u, err = url.Parse(arg)
			if err != nil {
				return fmt.Errorf("invalid url[%d]: %s", i, arg)
			}
			if u.Scheme == "" {
				u.Scheme = "https"
			}

			switch {
			case strings.Contains(arg, "/g/"):
				galleries = append(galleries, u.String())
			case strings.Contains(arg, "/s/"):
				pages = append(pages, u.String())

			default:
				dlLog.Warnf("invalid url[%d]: %s", i, arg)
				if !utils.WaitAck("continue?") {
					return ErrAborted
				}
			}
		}
		if len(galleries) == 0 && len(pages) == 0 {
			dlLog.Error("no valid url found")
			return ErrHandled
		}

		if len(sliceSyntaxes) != 0 && len(galleries) > 1 {
			dlLog.Warn("you are using page range in multiple galleries")
			if !utils.WaitAck("continue?") {
				return ErrAborted
			}
		}

		// galleries
		if len(galleries) != 0 {
			var dl *ehentaiDownload
			dl, err = galleryDownload(cmd.Context(), galleries, sliceSyntaxes)
			if err != nil {
				return
			}

			err = dl.download()
			if err != nil {
				dlLog.Error("failed to download: ", err)
				return ErrHandled
			}
		}

		// pages
		if len(pages) != 0 {
			if len(sliceSyntaxes) != 0 {
				dlLog.Warn("slice syntax will be ignored for pages download")
			}
			var dl *ehentaiDownload
			dl, err = pagesDownload(cmd.Context(), pages)
			if err != nil {
				return
			}

			err = dl.download()
			if err != nil {
				dlLog.Error("failed to download: ", err)
				return ErrHandled
			}
		}

		// 写入是异步的
		EHentai.WaitForWrite()

		dlLog.Info("download completed")
		return nil
	},
}

const (
	FLAG_THREADS         = "threads"
	FLAG_PROGRESS        = "progress"
	FLAG_RETRY           = "retry"
	FLAG_ENV_PROXY       = "env-proxy"
	FLAG_DOMAIN_FRONTING = "domain-fronting"
	FLAG_DIR             = "dir"

	FLAG_PAGES = "pages"
)

func init() {
	downloadCmd.Flags().Int(FLAG_THREADS, 8, "number of threads to use for downloading")
	downloadCmd.Flags().Bool(FLAG_PROGRESS, true, "show progress bar")
	downloadCmd.Flags().IntP(FLAG_RETRY, "r", 2, "letry broken images")
	downloadCmd.Flags().Bool(FLAG_ENV_PROXY, true, "look up proxy from environment variables")
	downloadCmd.Flags().Bool(FLAG_DOMAIN_FRONTING, false, "enable domain fronting")
	downloadCmd.Flags().String(FLAG_DIR, env.XDir, "directory to save downloaded files")

	downloadCmd.Flags().StringP(FLAG_PAGES, "p", "", "specify gallery pages range (e.g. [3:-3])")
	rootCmd.AddCommand(downloadCmd)
}

func downloadFlagChanged(downloadFlags *pflag.FlagSet) {
	if downloadFlags.Changed(FLAG_THREADS) {
		threads, _ := downloadFlags.GetInt(FLAG_THREADS)
		viper.Set("download.threads", threads)
	}
	if downloadFlags.Changed(FLAG_PROGRESS) {
		progress, _ := downloadFlags.GetBool(FLAG_PROGRESS)
		viper.Set("download.progressBar", progress)
	}
	if downloadFlags.Changed(FLAG_RETRY) {
		retry, _ := downloadFlags.GetInt(FLAG_RETRY)
		viper.Set("download.retryDepth", retry)
	}
	if downloadFlags.Changed(FLAG_ENV_PROXY) {
		envProxy, _ := downloadFlags.GetBool(FLAG_ENV_PROXY)
		viper.Set("download.envProxy", envProxy)
	}
	if downloadFlags.Changed(FLAG_DOMAIN_FRONTING) {
		domainFronting, _ := downloadFlags.GetBool(FLAG_DOMAIN_FRONTING)
		viper.Set("download.domainFronting", domainFronting)
	}
	if downloadFlags.Changed(FLAG_DIR) {
		dir, _ := downloadFlags.GetString(FLAG_DIR)
		viper.Set("download.dir", dir)
	}
}

type ehentaiDownload struct {
	GIds        []int // 准备下载的画廊, 作为以下 map 的 key
	Gallerys    map[int]EHentai.Gallery
	GalleryUrls map[int]string
	PageUrls    map[int][]string // 对应画廊要下载的链接, 长度可能小于 .Totals

	ctx      context.Context
	progress *mpb.Progress
	bar      *mpb.Bar
	start    time.Time
}

func galleryDownload(ctx context.Context, galleryUrls []string, sss utils.SliceSyntaxes) (dl *ehentaiDownload, err error) {
	dl = &ehentaiDownload{
		GIds:        make([]int, len(galleryUrls)),
		Gallerys:    make(map[int]EHentai.Gallery, len(galleryUrls)),
		GalleryUrls: make(map[int]string, len(galleryUrls)),
		PageUrls:    make(map[int][]string, len(galleryUrls)),

		ctx: ctx,
	}

	// 按原顺序收集画廊 ID 和 URL
	// 画廊 ID 作为 map 的 key
	for i := range galleryUrls {
		g := EHentai.UrlToGallery(galleryUrls[i])
		dl.GIds[i] = g.GalleryId
		dl.Gallerys[g.GalleryId] = g
		dl.GalleryUrls[g.GalleryId] = galleryUrls[i]
	}

	// 获取画廊元数据
	gl := make([]EHentai.GIdList, len(dl.GIds))
	for i, gId := range dl.GIds {
		gl[i] = dl.Gallerys[gId]
	}
	resp, err := EHentai.PostGalleryMetadata(ctx, gl...)
	if err != nil {
		return nil, err
	}
	if len(resp.GMetadata) != len(dl.GIds) {
		return nil, fmt.Errorf("len(resp.GMetadata)(%d) != len(dl.GIds)(%d)", len(resp.GMetadata), len(dl.GIds))
	}

	// 获取画廊页链接 同时解析 [utils.SliceSyntaxes]
	for _, gId := range dl.GIds {
		gallery, err := EHentai.FetchGalleryDetails(ctx, dl.GalleryUrls[gId])
		if err != nil {
			return nil, err
		}
		pageUrls := gallery.PageUrls

		var indexes []int
		if len(sss) != 0 {
			indexes, err = sss.ToIndexesNoRepeat(len(pageUrls))
			if err != nil {
				return nil, err
			}
			pageUrls = utils.DoIndexes(pageUrls, indexes)
		}

		dl.PageUrls[gId] = pageUrls
	}

	// 进度条
	n := 0
	for _, urls := range dl.PageUrls {
		n += len(urls)
	}
	dl.pbInit(ctx, int64(n))

	return
}

func pagesDownload(ctx context.Context, pageUrls []string) (dl *ehentaiDownload, err error) {
	// 从 pageUrls 中整理出画廊
	gPageUrls := map[int][]string{}
	for i := range pageUrls {
		g := EHentai.UrlToPage(pageUrls[i])
		gPageUrls[g.GalleryId] = append(gPageUrls[g.GalleryId], pageUrls[i])
	}

	dl = &ehentaiDownload{
		GIds:        make([]int, 0, len(gPageUrls)),
		Gallerys:    make(map[int]EHentai.Gallery, len(gPageUrls)),
		GalleryUrls: make(map[int]string, len(gPageUrls)),
		PageUrls:    gPageUrls,

		ctx: ctx,
	}

	// 排序画廊 ID
	// 画廊 ID 作为 map 的 key
	for gId := range dl.PageUrls {
		dl.GIds = append(dl.GIds, gId)
	}
	slices.Sort(dl.GIds)

	// 从每个画廊中取一个 P
	// 获取画廊 token
	pageList := make([]EHentai.PageList, len(dl.GIds))
	for i, gId := range dl.GIds {
		pageList[i] = EHentai.UrlToPage(dl.PageUrls[gId][0])
	}
	resp1, err := EHentai.PostGalleryToken(ctx, pageList...)
	if err != nil {
		return nil, err
	}
	if len(resp1.TokenLists) != len(dl.GIds) {
		return nil, fmt.Errorf("len(resp1.TokenLists)(%d) != len(dl.GIds)(%d)", len(resp1.TokenLists), len(dl.GIds))
	}

	// 获取画廊元数据
	gl := make([]EHentai.GIdList, len(dl.GIds))
	for i := range dl.GIds {
		gl[i] = resp1.TokenLists[i].ToGallery()
	}
	resp2, err := EHentai.PostGalleryMetadata(ctx, gl...)
	if err != nil {
		return nil, err
	}
	if len(resp2.GMetadata) != len(dl.GIds) {
		return nil, fmt.Errorf("len(resp2.GMetadata)(%d) != len(dl.GIds)(%d)", len(resp2.GMetadata), len(dl.GIds))
	}

	// 进度条
	n := 0
	for _, urls := range dl.PageUrls {
		n += len(urls)
	}
	dl.pbInit(ctx, int64(n))

	return
}

func (dl *ehentaiDownload) download() (err error) {
	for _, gId := range dl.GIds {
		dlLog.Info("downloading: ", dl.GalleryUrls[gId])

		// 手动为画廊创建缓存
		if EHentai.GetCache(dl.Gallerys[gId].GalleryId) == nil {
			_, err = EHentai.CreateCacheFromUrl(dl.ctx, dl.GalleryUrls[gId])
			if err != nil {
				dlLog.Error("failed to create: ", err)
				return ErrHandled
			}
		}

		for page, dlErr := range EHentai.DownloadPagesIter(dl.ctx, dl.PageUrls[gId]...) {
			dl.pbIncr(1)
			if dlErr != nil {
				err = dlErr
				dlLog.Errorf("failed to download gallery %d page %d: %v", gId, page.PageNum, dlErr)
				if !utils.WaitAck("continue?") {
					return ErrAborted
				}
				continue
			}
		}
	}
	return nil
}

func (dl *ehentaiDownload) pbInit(ctx context.Context, total int64) {
	if !config.Download.ProgressBar {
		return
	}
	dl.progress = mpb.NewWithContext(ctx, bar.RefreshRate)
	dl.bar = dl.progress.New(total,
		bar.BarStyleMain,
		mpb.PrependDecorators(
			bar.Spinner,
			bar.ETA,
		),
		mpb.BarRemoveOnComplete(),
	)
	dl.bar.SetPriority(bar.Priority())
}

func (dl *ehentaiDownload) pbIncr(n int64) {
	if !config.Download.ProgressBar {
		return
	}
	dl.bar.EwmaIncrInt64(n, time.Since(dl.start))
}
