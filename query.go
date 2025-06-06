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

const (
	EHENTAI_URL  = `https://e-hentai.org`
	EXHENTAI_URL = `https://exhentai.org`
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
)

// Found about 192,819 results.
// Found 2 results.
// Found 1 result.
var foundReg = regexp.MustCompile(`Found(?: about)? ([\d,]+) results?`)

func searchDetail(ctx context.Context, url, keyword string, categories ...Category) (total int, galleries GalleryMetadatas, err error) {
	total, results, err := queryFSearch(ctx, url, keyword, categories...)
	if err != nil {
		return
	}
	list := make([]Gallery, len(results))
	for i := range results {
		list[i] = Gallery{results[i].Gid, results[i].Token}
	}
	resp, err := PostGalleryMetadata(ctx, list...)
	if err != nil {
		return 0, nil, err
	}
	return total, resp.GMetadata, nil
}

// total != len(results) 即不止一页
func queryFSearch(ctx context.Context, url, keyword string, categories ...Category) (total int, results FSearchResults, err error) {
	u, err := netUrl.Parse(url)
	if err != nil {
		return 0, nil, err
	}
	querys := make(netUrl.Values)
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
		// cat   "body > div.ido > div:nth-child(2) > table > tbody > tr:nth-child(2) > td.gl1c.glcat > div"
		// cover "body > div.ido > div:nth-child(2) > table > tbody > tr:nth-child(2) > td.gl2c > div > div > img"
		// stars "body > div.ido > div:nth-child(2) > table > tbody > tr:nth-child(2) > td.gl2c > div > div.ir"
		// url   "body > div.ido > div:nth-child(2) > table > tbody > tr:nth-child(2) > td.gl3c.glname > a"
		// tag1  "body > div.ido > div:nth-child(2) > table > tbody > tr:nth-child(3) > td.gl3c.glname > a > div:nth-child(2) > div:nth-child(1)"
		// tag2  "body > div.ido > div:nth-child(2) > table > tbody > tr:nth-child(3) > td.gl3c.glname > a > div:nth-child(2) > div:nth-child(2)"
		// title "body > div.ido > div:nth-child(2) > table > tbody > tr:nth-child(2) > td.gl3c.glname > a > div.glink"
		// pages "body > div.ido > div:nth-child(2) > table > tbody > tr:nth-child(2) > td.gl4c.glhide > div:nth-child(2)"
		cat := s.Find("td.gl1c.glcat > div").Text()
		cover, ok := s.Find("td.gl2c > div > div > img").Attr("data-src")
		if !ok {
			cover, _ = s.Find("td.gl2c > div > div > img").Attr("src")
		}
		stars, _ := s.Find("td.gl2c > div > div.ir").Attr("style")
		url, _ := s.Find("td.gl3c.glname > a").Attr("href")
		var tags []string
		s.Find("td.gl3c.glname > a > div > div.gt").
			Each(func(i int, s *goquery.Selection) {
				tags = append(tags, s.AttrOr("title", s.Text()))
			})
		title := s.Find("td.gl3c.glname > a > div.glink").Text()
		pages := s.Find("td.gl4c.glhide > div:nth-child(2)").Text()
		domain, gId, gToken := UrlGetGIdGToken(url)
		if gId, _ := atoi(gId); gId != 0 && gToken != "" {
			results = append(results, FSearchResult{domain, gId, gToken, cat, cover, parseStars(stars), url, tags, title, pages})
		}
	})
	return
}

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

func parseStars(stars string) (rating string) {
	matches := starsReg.FindStringSubmatch(stars)
	if len(matches) == 0 {
		return ""
	}
	x, err := atoi(matches[1])
	if err != nil {
		return ""
	}
	y, err := atoi(matches[2])
	if err != nil {
		return ""
	}
	units := 5
	decimal := 0
	if y < -21 {
		units -= 1
		decimal = 5
	}
	units -= (-x / 16)
	return itoa(units) + "." + itoa(decimal)
}

// Showing 1 - 20 of 65 images
// Showing 1 - 5 of 5 images
// Showing 1 - 1 of 1 image (?
var numReg = regexp.MustCompile(`Showing 1 - (\d+) of (\d+) images?`)

// fetchGalleryPages 遍历获取所有页链接
func fetchGalleryPages(ctx context.Context, galleryUrl string) (pageUrls []string, err error) {
	defer func() {
		if err == nil {
			// 缓存页链接
			g := UrlToGallery(galleryUrl)
			metaCacheWrite(g.GalleryId, nil, pageUrls)
		}
	}()

	doc, err := httpGetDoc(ctx, galleryUrl)
	if err != nil {
		return nil, err
	}

	// body > div:nth-child(*) > p
	// <p class="gpc">Showing 1 - 5 of 5 images</p>
	numImages := doc.Find(".gpc").Text()
	matches := numReg.FindStringSubmatch(numImages)
	if len(matches) == 0 {
		return nil, wrapErr(ErrRegMismatch, numImages)
	}
	if matches[1] == "" || matches[2] == "" {
		return nil, wrapErr(ErrRegEmptyMatch, numImages)
	}
	end, _ := atoi(matches[1])
	total, _ := atoi(matches[2])
	if end == 0 || total == 0 {
		return nil, wrapErr(ErrNoImage, numImages)
	}
	pages := 0
	if end == total {
		pages = 1
	} else if end > total {
		return nil, wrapErr(ErrEndGreaterThanTotal, numImages)
	} else {
		pages = total / end
		if total%end != 0 {
			pages++
		}
	}

	pageUrls = make([]string, total)
	errs := make(chan error, pages)

	wg := sync.WaitGroup{}
	wg.Add(pages)

	limiter := newLimiter()
	defer limiter.close()

	for page := range pages {
		select {
		case <-ctx.Done():
			return
		case limiter.acquire() <- struct{}{}:
		}

		go func(page int) {
			defer func() {
				limiter.release()
				wg.Done()
			}()

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
		}(page)
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	for err := range errs {
		if err != nil {
			return nil, err
		}
	}

	for i := range pageUrls {
		if pageUrls[i] == "" {
			return nil, wrapErr(ErrFoundEmptyPageUrl, i)
		}
	}

	return pageUrls, nil
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
