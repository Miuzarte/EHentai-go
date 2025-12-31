package EHentai

import (
	"context"
	"errors"
	netUrl "net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type Category uint

const ( // 实际 query 时要用 1023^CATEGORY_XXX
	CATEGORY_MISC Category = 1 << iota
	CATEGORY_DOUJINSHI
	CATEGORY_MANGA
	CATEGORY_ARTIST_CG
	CATEGORY_GAME_CG
	CATEGORY_IMAGE_SET
	CATEGORY_COSPLAY
	CATEGORY_ASIAN_PORN
	CATEGORY_NON_H
	CATEGORY_WESTERN
)

const CATEGORY_COUNT = 10

func (c Category) Str() string {
	switch c {
	case CATEGORY_MISC:
		return "Miscellaneous"
	case CATEGORY_DOUJINSHI:
		return "Doujinshi"
	case CATEGORY_MANGA:
		return "Manga"
	case CATEGORY_ARTIST_CG:
		return "Artist CG"
	case CATEGORY_GAME_CG:
		return "Game CG"
	case CATEGORY_IMAGE_SET:
		return "Image Set"
	case CATEGORY_COSPLAY:
		return "Cosplay"
	case CATEGORY_ASIAN_PORN:
		return "Asian Porn"
	case CATEGORY_NON_H:
		return "Non-H"
	case CATEGORY_WESTERN:
		return "Western"
	}
	return "unknown"
}

func (c Category) String() string {
	if c == 0 {
		return "All"
	}
	var cats []string
	for i := range CATEGORY_COUNT {
		if c&(1<<i) != 0 {
			cats = append(cats, Category(1<<i).Str())
		}
	}
	return strings.Join(cats, " | ")
}

func (c Category) Format() string {
	return strconv.FormatUint(uint64(1023^c), 10)
}

var ErrUnknownCategory = errors.New("unknown category")

func ParseCategory(ss ...string) (cat Category, err error) {
	if len(ss) == 0 {
		return 0, nil
	}
	replacer := strings.NewReplacer(
		"-", "",
		"_", "",
		" ", "",
	)
	for _, s := range ss {
		s = replacer.Replace(s)
		s = strings.ToUpper(s)
		switch s {
		case "MISC":
			cat |= CATEGORY_MISC
		case "DOUJINSHI":
			cat |= CATEGORY_DOUJINSHI
		case "MANGA":
			cat |= CATEGORY_MANGA
		case "ARTISTCG":
			cat |= CATEGORY_ARTIST_CG
		case "GAMECG":
			cat |= CATEGORY_GAME_CG
		case "IMAGESET":
			cat |= CATEGORY_IMAGE_SET
		case "COSPLAY":
			cat |= CATEGORY_COSPLAY
		case "ASIANPORN":
			cat |= CATEGORY_ASIAN_PORN
		case "NONH":
			cat |= CATEGORY_NON_H
		case "WESTERN":
			cat |= CATEGORY_WESTERN
		default:
			return 0, wrapErr(ErrUnknownCategory, s)
		}
	}
	return
}

var (
	ErrSadPanda            = errors.New("sad panda")
	ErrIpBanned            = errors.New("ip banned")
	ErrNoHitsFound         = errors.New("no hits found")
	ErrRegMismatch         = errors.New("regexp mismatch")
	ErrRegEmptyMatch       = errors.New("regexp empty match")
	ErrNoResult            = errors.New("no result")
	ErrEmptyTable          = errors.New("empty table")
	ErrNoImage             = errors.New("no image")
	ErrEndGreaterThanTotal = errors.New("end > total")
	ErrFoundEmptyPageUrl   = errors.New("found empty page url")
	ErrNotANumber          = errors.New("not a number")
)

func searchDetail(ctx context.Context, url Url, keyword string, categories ...Category) (total int, galleries GalleryMetadatas, err error) {
	total, results, err := queryFSearch(ctx, url, keyword, categories...)
	if err != nil {
		return 0, nil, err
	}
	list := make([]Gallery, len(results))
	for i := range results {
		list[i] = results[i].Gallery
	}
	metadatas, err := PostGalleryMetadata(ctx, list...)
	if err != nil {
		return 0, nil, err
	}
	return total, metadatas, nil
}

// Found about 192,819 results.
// Found 1,000+ results.
// Found 2 results.
// Found 1 result.
var foundReg = regexp.MustCompile(`Found(?: about)? ([\d,]+)\+? results?`)

// 5   background-position:0px -1px;opacity:1
// 4.5 background-position:0px -21px;opacity:1
// 4   background-position:-16px -1px;opacity:1
// 3.5 background-position:-16px -21px;opacity:1
// 3   background-position:-32px -1px;opacity:1
// 2.5 background-position:-32px -21px;opacity:1
// 2   background-position:-48px -1px;opacity:1
// 1.5 background-position:-48px -21px;opacity:1
// 1   background-position:-64px -1px;opacity:1
// 0.5 background-position:-64px -21px;opacity:1
// 0   background-position:-80px -1px;opacity:1
var starsReg = regexp.MustCompile(`background-position:(-?\d+)px (-\d+)px`)

func parseStars(stars string) (rating float64) {
	matches := starsReg.FindStringSubmatch(stars)
	if len(matches) == 0 {
		return 0
	}
	x, err := atoi(matches[1])
	if err != nil {
		return 0
	}
	y, err := atoi(matches[2])
	if err != nil {
		return 0
	}

	rating = 5.0 - float64(-x/16)
	switch y {
	case -1:
	case -21:
		rating -= 0.5
	default:
		rating -= float64(y+1) / 20.0
	}
	return
}

// total != len(results) 即不止一页
func queryFSearch(ctx context.Context, url Url, keyword string, categories ...Category) (total int, results FSearchResults, err error) {
	u, err := netUrl.Parse(url)
	if err != nil {
		return 0, nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	querys := netUrl.Values{}
	if len(categories) != 0 {
		var cate Category
		for _, c := range categories {
			cate |= c
		}
		querys.Set("f_cats", cate.Format())
	}
	if keyword != "" {
		querys.Set("f_search", keyword)
	}
	u.RawQuery = querys.Encode()

	doc, err := httpGetDoc(ctx, u.String())
	if err != nil {
		return 0, nil, err
	}

	// body > div.ido > div:nth-child(2) > p
	noHitsFound := doc.Find("body > div.ido > div:nth-child(2) > p").Text()
	if noHitsFound != "" {
		return 0, nil, wrapErr(ErrNoHitsFound, noHitsFound)
	}

	// body > div.ido > div:nth-child(2) > div.searchtext > p
	foundResults := doc.Find("body > div.ido > div:nth-child(2) > div.searchtext > p").Text()
	matches := foundReg.FindStringSubmatch(foundResults)
	if len(matches) == 0 {
		return 0, nil, wrapErr(ErrRegMismatch, foundResults)
	}
	if matches[1] == "" {
		return 0, nil, wrapErr(ErrRegEmptyMatch, foundResults)
	}
	total, _ = atoi(strings.ReplaceAll(matches[1], ",", ""))
	if total == 0 {
		return 0, nil, wrapErr(ErrNoResult, foundResults)
	}

	// body > div.ido > div:nth-child(2) > table > tbody > tr:nth-child(*)
	table := doc.Find("body > div.ido > div:nth-child(2) > table > tbody > tr")
	tableLen := table.Length()
	if tableLen == 0 {
		return 0, nil, wrapErr(ErrEmptyTable, foundResults)
	}
	results = make(FSearchResults, 0, tableLen-1)
	table.Each(func(i int, s *goquery.Selection) {
		if i == 0 { // 表头
			return
		}

		// cat      "td.gl1c.glcat > div"

		// cover    "td.gl2c > div.glthumb > div:nth-child(1) > img"
		// time     "td.gl2c > div:nth-child(3) > div:nth-child(1)"
		// stars    "td.gl2c > div:nth-child(3) > div.ir"

		// url   "td.gl3c.glname > a" // href
		// title "td.gl3c.glname > a > div.glink"
		// tags  "td.gl3c.glname > a > div:nth-child(2)"
		// tag1  "td.gl3c.glname > a > div:nth-child(2) > div:nth-child(1)"
		// tag2  "td.gl3c.glname > a > div:nth-child(2) > div:nth-child(2)"

		// uploader "td.gl4c.glhide > div:nth-child(1) > a"
		// pages    "td.gl4c.glhide > div:nth-child(2)"

		// 分类
		cat := s.Find("td.gl1c.glcat > div").Text()

		gl2c := s.Find("td.gl2c > div")
		// 封面
		cover, ok := gl2c.Find("div > img").Attr("data-src")
		if !ok {
			cover, _ = gl2c.Find("div > img").Attr("src")
		}
		// 上传时间
		upTime := gl2c.Find("div:nth-child(3) > div:nth-child(1)").Text()
		// 评分
		stars, _ := gl2c.Find("div.ir").Attr("style")

		gl3c := s.Find("td.gl3c.glname > a")
		// 链接
		url, _ := gl3c.Attr("href")
		// 标题
		title := gl3c.Find("div.glink").Text()
		var tags []string
		// 标签
		gl3c.Find("div:nth-child(2) > div").Each(func(i int, s *goquery.Selection) {
			tags = append(tags, s.AttrOr("title", s.Text()))
		})

		gl4c := s.Find("td.gl4c.glhide")
		// 上传者
		uploader := gl4c.Find("div:nth-child(1) > a").Text()
		// 页数
		pages := gl4c.Find("div:nth-child(2)").Text()
		pages = strings.TrimSuffix(pages, " pages") // "65 pages"
		var pagesNum int
		pagesNum, err = atoi(pages)
		if err != nil {
			err = wrapErr(ErrNotANumber, pages)
			return
		}

		domain, gId, gToken := UrlGetGIdGToken(url)
		gIdNum, _ := atoi(gId)
		results = append(results, FSearchResult{
			Domain: domain,
			Gallery: Gallery{
				GalleryId:    gIdNum,
				GalleryToken: gToken,
			},

			Cat: cat,

			Cover:  cover,
			Posted: upTime,
			Rating: parseStars(stars),

			Url:   url,
			Title: title,
			Tags:  tags,

			Uploader: uploader,
			Pages:    pagesNum,
		})
	})
	return
}

var (
	coverUrlReg = regexp.MustCompile(`url\(([^)]+)\)`)
	// Showing 1 - 20 of 2,000 images
	// Showing 1 - 20 of 65 images
	// Showing 1 - 5 of 5 images
	// Showing 1 - 1 of 1 image (?
	numReg = regexp.MustCompile(`Showing 1 - (\d+) of ([\d,]+) images?`)
)

var ErrInvalidUrl = errors.New("invalid url")

func fetchGalleryDetailsTryCache(ctx context.Context, galleryUrl string) (GalleryDetails, error) {
	g := UrlToGallery(galleryUrl)
	if g.GalleryId == 0 {
		return GalleryDetails{}, wrapErr(ErrInvalidUrl, galleryUrl)
	}

	if gallery := DetailsCacheRead(g.GalleryId); gallery != nil {
		return *gallery, nil
	}

	// inline optimized
	return fetchGalleryDetails(ctx, galleryUrl)
}

func fetchGalleryDetails(ctx context.Context, galleryUrl string) (gallery GalleryDetails, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer func() {
		if err == nil {
			// 缓存画廊详情与页链接
			g := UrlToGallery(galleryUrl)
			detailsCacheWrite(g.GalleryId, gallery)
			metaCacheWrite(g.GalleryId, nil, gallery.PageUrls)
		}
	}()

	doc, err := httpGetDoc(ctx, galleryUrl)
	if err != nil {
		return
	}

	// cover    "#gd1 > div"
	// title    "#gn"
	// titleJpn "#gj"

	var cover string
	doc.Find("#gd1 > div").Each(func(i int, sel *goquery.Selection) {
		style, exists := sel.Attr("style")
		if exists {
			matches := coverUrlReg.FindStringSubmatch(style)
			if len(matches) > 1 {
				cover = matches[1]
			}
		}
	})

	title := doc.Find("#gn").Text()
	titleJpn := doc.Find("#gj").Text()

	// cat      "#gdc > div"
	// uploader "#gdn > a:nth-child(1)"

	cat := doc.Find("#gdc > div").Text()
	uploader := doc.Find("#gdn > a:nth-child(1)").Text()

	// posted     "#gdd > table > tbody > tr:nth-child(1) > td.gdt2"
	// parent     "#gdd > table > tbody > tr:nth-child(2) > td.gdt2"
	//            "#gdd > table > tbody > tr:nth-child(2) > td.gdt2 > a"    // galleryId
	// visible    "#gdd > table > tbody > tr:nth-child(3) > td.gdt2"
	// language   "#gdd > table > tbody > tr:nth-child(4) > td.gdt2"        // "Chinese &nbsp;"
	// translated "#gdd > table > tbody > tr:nth-child(4) > td.gdt2 > span" // 译本存在 "TR" 标
	// fileSize   "#gdd > table > tbody > tr:nth-child(5) > td.gdt2"
	// length     "#gdd > table > tbody > tr:nth-child(6) > td.gdt2"
	// favorited  "#gdd > table > tbody > tr:nth-child(7) > td.gdt2"        // "3745 times"
	//            "#favcount"

	gdd := doc.Find("#gdd > table > tbody")
	posted := gdd.Find("tr:nth-child(1) > td.gdt2").Text()
	parent := gdd.Find("tr:nth-child(2) > td.gdt2 > a").Text()
	parentId, _ := atoi(parent)
	visible := gdd.Find("tr:nth-child(3) > td.gdt2").Text()
	langSel := gdd.Find("tr:nth-child(4) > td.gdt2").Clone()
	langSel.Find("span").Remove()
	language := strings.TrimSpace(langSel.Text()) // "Chinese &nbsp;"
	translated := gdd.Find("tr:nth-child(4) > td.gdt2 > span").Text()
	fileSize := gdd.Find("tr:nth-child(5) > td.gdt2").Text()
	length := gdd.Find("tr:nth-child(6) > td.gdt2").Text()
	length = strings.TrimSuffix(length, " pages") // "65 pages"
	lengthNum, _ := atoi(length)
	favorited := gdd.Find("tr:nth-child(7) > td.gdt2").Text()
	favorited = strings.TrimSuffix(favorited, " times") // "3745 times"
	favoritedNum, _ := atoi(favorited)

	// ratingCount  "#rating_count"
	// rating       "#rating_label" // "Average: 4.86"

	ratingCount := doc.Find("#rating_count").Text()
	ratingCountNum, _ := atoi(ratingCount)
	rating := doc.Find("#rating_label").Text()
	rating = strings.TrimPrefix(rating, "Average: ") // "Average: 4.86"
	rating = strings.TrimSpace(rating)
	ratingNum, err := strconv.ParseFloat(rating, 64)
	if err != nil {
		err = wrapErr(ErrNotANumber, rating)
		return
	}

	// tags         "#taglist > table > tbody"
	// tagNamespace "#taglist > table > tbody > tr:nth-child(1) > td:nth-child(1)"                        // "language:"
	// tag          "#taglist > table > tbody > tr:nth-child(x) > td:nth-child(2) > div:nth-child(1) > a" // "chinese"
	//              "#taglist > table > tbody > tr:nth-child(x) > td:nth-child(2) > div:nth-child(2) > a" // "translated"

	var tags []Tag
	taglist := doc.Find("#taglist > table > tbody")
	taglist.Find("tr").Each(func(i int, s *goquery.Selection) {
		namespace := s.Find("td:nth-child(1)").Text()
		if namespace == "" {
			return
		}
		namespace = strings.TrimSuffix(namespace, ":") // "language:"
		s.Find("td:nth-child(2) > div").Each(func(i int, s *goquery.Selection) {
			tag := s.Find("a").Text()
			if tag != "" {
				tags = append(tags, Tag{Namespace: namespace, Name: tag})
			}
		})
	})

	// body > div:nth-child(*) > p
	// <p class="gpc">Showing 1 - 5 of 5 images</p>
	numImages := doc.Find(".gpc").Text()
	matches := numReg.FindStringSubmatch(numImages)
	if len(matches) == 0 {
		err = wrapErr(ErrRegMismatch, numImages)
		return
	}
	if matches[1] == "" || matches[2] == "" {
		err = wrapErr(ErrRegEmptyMatch, numImages)
		return
	}
	matches[2] = strings.ReplaceAll(matches[2], ",", "")
	end, _ := atoi(matches[1])
	total, _ := atoi(matches[2])
	if end == 0 || total == 0 {
		err = wrapErr(ErrNoImage, numImages)
		return
	}
	pages := 0
	if end == total {
		pages = 1
	} else if end > total {
		err = wrapErr(ErrEndGreaterThanTotal, numImages)
		return
	} else {
		pages = total / end
		if total%end != 0 {
			pages++
		}
	}

	pageUrls := make([]string, total)
	errs := make(chan error, pages)

	wg := sync.WaitGroup{}
	limiter := newLimiter()
	defer limiter.close()

	for page := range pages {
		select {
		case <-ctx.Done():
			return
		case limiter.acquire() <- struct{}{}:
		}

		wg.Go(func() {
			defer limiter.release()

			var pageDoc *goquery.Document
			if page == 0 { // 起始页不需要重新加载
				pageDoc = doc
			} else {
				u, err := netUrl.Parse(galleryUrl)
				if err != nil {
					errs <- err
					return
				}
				u.RawQuery = "p=" + itoa(page)
				pageDoc, err = httpGetDoc(ctx, u.String())
				if err != nil {
					errs <- err
					return
				}
			}
			// #gdt > a:nth-child(*)
			pageDoc.Find("#gdt > a").Each(func(i int, s *goquery.Selection) {
				index := page*end + i
				url := s.AttrOr("href", "")
				pageUrls[index] = url
			})
		})

	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	for e := range errs {
		if e != nil {
			err = e
			return
		}
	}

	for i := range pageUrls {
		if pageUrls[i] == "" {
			err = wrapErr(ErrFoundEmptyPageUrl, i)
			return
		}
	}

	domain, gId, gToken := UrlGetGIdGToken(galleryUrl)
	gIdNum, _ := atoi(gId)
	gallery = GalleryDetails{
		Domain: domain,
		Gallery: Gallery{
			GalleryId:    gIdNum,
			GalleryToken: gToken,
		},

		Cover:    cover,
		Title:    title,
		TitleJpn: titleJpn,

		Cat:      cat,
		Uploader: uploader,

		Posted:     posted,
		Parent:     parentId,
		Visible:    visible,
		Language:   language,
		Translated: translated,
		FileSize:   fileSize,
		Length:     lengthNum,
		Favorited:  favoritedNum,

		RatingCount: ratingCountNum,
		Rating:      ratingNum,

		Tags: tags,

		PageUrls: pageUrls,
	}
	return
}

// fetchPageImageUrl 获取画廊某页的图直链与页备链
func fetchPageImageUrl(ctx context.Context, pageUrl string) (imgUrl string, bakPage string, err error) {
	doc, err := httpGetDoc(ctx, pageUrl)
	if err != nil {
		return "", "", err
	}
	// #img
	img, _ := doc.Find("#img").Attr("src")
	// <a href="#" id="loadfail" onclick="return nl('SZF-483294')">Reload broken image</a>
	onclick, _ := doc.Find("#loadfail").Attr("onclick")
	return img, nl(pageUrl, onclick), nil
}

var nlReg = regexp.MustCompile(`nl\('(.+?)'\)`)

func nl(url, onclick string) string {
	u, err := netUrl.Parse(url)
	if err != nil {
		return ""
	}
	matches := nlReg.FindStringSubmatch(onclick)
	if len(matches) == 0 {
		return ""
	}
	nl := matches[1]
	if u.RawQuery != "" {
		u.RawQuery += "&nl=" + nl
	} else {
		u.RawQuery = "nl=" + nl
	}
	return u.String()
}
