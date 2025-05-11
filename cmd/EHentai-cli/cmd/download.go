package cmd

import (
	"context"
	"fmt"
	"iter"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/Miuzarte/EHentai-go"
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/bar"
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/config"
	"github.com/Miuzarte/EHentai-go/cmd/EHentai-cli/internal/errors"
	"github.com/Miuzarte/EHentai-go/internal/utils"
	"github.com/Miuzarte/SimpleLog"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
)

var dlLog = SimpleLog.New("[Download]", true, false)

const downloadDesc = "Download gallery or pages, using slice syntax(allow negative) to specify page range"

const downloadDescLong = downloadDesc +
	"\n" + "Help for slice syntax: " +
	"\n" + "[start:end] / [:end] / [start:] / [index]" +
	"\n" + "start is inclusive, end is exclusive" +
	"\n" + "negative index is allowed" +
	"\n" + "e.g." +
	"\n" + "\"[3:-1]\": start from 3rd page without last page"

var pageRange string

var downloadCmd = &cobra.Command{
	Use:   "download <gallery/page url>... [-p <page range>]",
	Short: downloadDesc,
	Long:  downloadDescLong,
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(_ *cobra.Command, _ []string) (err error) {
		return config.InitConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// 解析切片
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
			u, err := url.Parse(arg)
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
				return fmt.Errorf("invalid url[%d]: %s", i, arg)
			}
		}

		if len(sliceSyntaxes) != 0 && len(args) > 1 {
			dlLog.Warn("you are using page range in multiple urls")
			var input string
			fmt.Print("continue? (y/n): ")
			fmt.Scanln(&input)
			if input != "y" && input != "Y" {
				err = errors.Aborted
				return
			}
		}

		var dl *ehentaiDownload
		dl, err = galleryDownload(cmd.Context(), galleries, sliceSyntaxes)
		if err != nil {
			return
		}

		// galleries
		for _, gId := range dl.GIds {
			dlLog.Info("downloading: ", dl.GalleryUrls[gId])
			for page, dlErr := range dl.downloadIter(cmd.Context(), gId) {
				if dlErr != nil {
					dlLog.Errorf("failed to download gallery %d page %d: %v", gId, page.PageNum, dlErr)
					continue
				}
			}
		}

		// pages
		if len(pages) != 0 {
			if len(sliceSyntaxes) != 0 {
				dlLog.Warn("slice syntax will be ignored for pages download")
			}
			dl, err = pagesDownload(cmd.Context(), pages)
			if err != nil {
				return
			}

			for _, gId := range dl.GIds {
				dlLog.Info("downloading: ", dl.GalleryUrls[gId])
				for page, dlErr := range dl.downloadIter(cmd.Context(), gId) {
					if dlErr != nil {
						dlLog.Errorf("failed to download gallery %d page %d: %v", gId, page.PageNum, dlErr)
						continue
					}
				}
			}
		}

		// 写入是异步的
		EHentai.WaitForWrite()

		dlLog.Info("download completed")
		return nil
	},
}

func init() {
	downloadCmd.Flags().StringVarP(&pageRange, "pages", "p", "", "Specify page range (e.g. [3:-3])")
	rootCmd.AddCommand(downloadCmd)
}

type ehentaiDownload struct {
	GIds        []int // 准备下载的画廊, 作为以下 map 的 key
	Gallerys    map[int]EHentai.Gallery
	GalleryUrls map[int]string
	PageUrls    map[int][]string // 对应画廊要下载的链接, 长度可能小于 .Totals

	progress *mpb.Progress
	bar      *mpb.Bar
	start    time.Time
}

func (ep *ehentaiDownload) initProgressBar(ctx context.Context, total int64) {
	if !config.Download.ProgressBar {
		return
	}
	ep.progress = mpb.NewWithContext(ctx, bar.RefreshRate)
	ep.bar = ep.progress.New(total,
		bar.BarStyleMain,
		mpb.PrependDecorators(
			bar.Spinner,
			bar.ETA,
		),
		mpb.BarRemoveOnComplete(),
	)
	ep.bar.SetPriority(bar.Priority())
}

func (ep *ehentaiDownload) pbIncr(n int64) {
	if !config.Download.ProgressBar {
		return
	}
	ep.bar.EwmaIncrInt64(n, time.Since(ep.start))
}

func galleryDownload(ctx context.Context, galleryUrls []string, sss utils.SliceSyntaxes) (ep *ehentaiDownload, err error) {
	ep = &ehentaiDownload{
		GIds:        make([]int, len(galleryUrls)),
		Gallerys:    make(map[int]EHentai.Gallery, len(galleryUrls)),
		GalleryUrls: make(map[int]string, len(galleryUrls)),
		PageUrls:    make(map[int][]string, len(galleryUrls)),
	}

	// 按原顺序收集画廊 ID 和 URL
	// 画廊 ID 作为 map 的 key
	for i := range galleryUrls {
		g := EHentai.UrlToGallery(galleryUrls[i])
		ep.GIds[i] = g.GalleryId
		ep.Gallerys[g.GalleryId] = g
		ep.GalleryUrls[g.GalleryId] = galleryUrls[i]
	}

	// 获取画廊元数据
	gl := make([]EHentai.GIdList, len(ep.GIds))
	for i, gId := range ep.GIds {
		gl[i] = ep.Gallerys[gId]
	}
	resp, err := EHentai.PostGalleryMetadata(ctx, gl...)
	if err != nil {
		return nil, err
	}
	if len(resp.GMetadata) != len(ep.GIds) {
		return nil, fmt.Errorf("len(resp.GMetadata)(%d) != len(ep.GIds)(%d)", len(resp.GMetadata), len(ep.GIds))
	}

	// 获取画廊页链接 同时解析 [utils.SliceSyntaxes]
	for _, gId := range ep.GIds {
		pageUrls, err := EHentai.FetchGalleryPageUrls(ctx, ep.GalleryUrls[gId])
		if err != nil {
			return nil, err
		}

		var indexes []int
		if len(sss) != 0 {
			indexes, err = sss.ToIndexesNoRepeat(len(pageUrls))
			if err != nil {
				return nil, err
			}
			pageUrls = utils.DoIndexes(pageUrls, indexes)
		}

		ep.PageUrls[gId] = pageUrls
	}

	// 进度条
	n := 0
	for _, urls := range ep.PageUrls {
		n += len(urls)
	}
	ep.initProgressBar(ctx, int64(n))

	return
}

func pagesDownload(ctx context.Context, pageUrls []string) (ep *ehentaiDownload, err error) {
	// 从 pageUrls 中整理出画廊
	gPageUrls := make(map[int][]string)
	for i := range pageUrls {
		g := EHentai.UrlToPage(pageUrls[i])
		gPageUrls[g.GalleryId] = append(gPageUrls[g.GalleryId], pageUrls[i])
	}

	ep = &ehentaiDownload{
		GIds:        make([]int, 0, len(gPageUrls)),
		Gallerys:    make(map[int]EHentai.Gallery, len(gPageUrls)),
		GalleryUrls: make(map[int]string, len(gPageUrls)),
		PageUrls:    gPageUrls,
	}

	// 排序画廊 ID
	// 画廊 ID 作为 map 的 key
	for gId := range ep.PageUrls {
		ep.GIds = append(ep.GIds, gId)
	}
	slices.Sort(ep.GIds)

	// 从每个画廊中取一个 P
	// 获取画廊 token
	pageList := make([]EHentai.PageList, len(ep.GIds))
	for i, gId := range ep.GIds {
		pageList[i] = EHentai.UrlToPage(ep.PageUrls[gId][0])
	}
	resp1, err := EHentai.PostGalleryToken(ctx, pageList...)
	if err != nil {
		return nil, err
	}
	if len(resp1.TokenLists) != len(ep.GIds) {
		return nil, fmt.Errorf("len(resp1.TokenLists)(%d) != len(ep.GIds)(%d)", len(resp1.TokenLists), len(ep.GIds))
	}

	// 获取画廊元数据
	gl := make([]EHentai.GIdList, len(ep.GIds))
	for i := range ep.GIds {
		gl[i] = resp1.TokenLists[i].ToGallery()
	}
	resp2, err := EHentai.PostGalleryMetadata(ctx, gl...)
	if err != nil {
		return nil, err
	}
	if len(resp2.GMetadata) != len(ep.GIds) {
		return nil, fmt.Errorf("len(resp2.GMetadata)(%d) != len(ep.GIds)(%d)", len(resp2.GMetadata), len(ep.GIds))
	}

	// 进度条
	n := 0
	for _, urls := range ep.PageUrls {
		n += len(urls)
	}
	ep.initProgressBar(ctx, int64(n))

	return
}

func (ep *ehentaiDownload) downloadIter(ctx context.Context, gId int) iter.Seq2[EHentai.PageData, error] {
	return func(yield func(EHentai.PageData, error) bool) {
		// 手动为画廊创建缓存
		if EHentai.GetCache(ep.Gallerys[gId].GalleryId) == nil {
			_, err := EHentai.CreateCacheFromUrl(ep.GalleryUrls[gId])
			if err != nil {
				yield(EHentai.PageData{}, err)
				return
			}
		}
		for page, err := range EHentai.DownloadPagesIter(ctx, ep.PageUrls[gId]...) {
			ep.pbIncr(1)
			if yield(page, err) {
				continue
			}
		}
	}
}
