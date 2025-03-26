package EHentai

import (
	"context"
	"errors"
	"net/http"
	netUrl "net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Domain = string

const (
	EHENTAI_DOMAIN  Domain = "e-hentai.org"
	EXHENTAI_DOMAIN Domain = "exhentai.org"
)

func checkDomain(u ...string) error {
	if !domainCheck {
		return nil
	}
	for _, u := range u {
		domain := urlGetDomain(u)
		switch domain {
		case EHENTAI_DOMAIN:
		case EXHENTAI_DOMAIN:
			if !cookie.Ok() {
				return ErrCookieNotSet
			}
		default:
			return errors.New("invalid url")
		}
	}
	return nil
}

func urlGetDomain(u string) Domain {
	switch {
	case strings.Contains(u, EHENTAI_DOMAIN):
		return EHENTAI_DOMAIN
	case strings.Contains(u, EXHENTAI_DOMAIN):
		return EXHENTAI_DOMAIN
	}
	return ""
}

func UrlGetGIdGToken(u string) (domain Domain, gId int, gToken string) {
	// https://e-hentai.org/g/{gallery_id}/{gallery_token}
	// https://e-hentai.org/g/3138775/30b0285f9b
	u = strings.TrimRight(u, "/")
	splits := strings.Split(u, "/")
	for i, s := range splits {
		if s == "g" {
			// i   g
			// i+1 3138775
			// i+2 30b0285f9b
			if len(splits) < i+3 {
				break
			}
			gId, err := strconv.Atoi(splits[i+1])
			if err != nil {
				break
			}
			switch splits[i-1] {
			case EHENTAI_DOMAIN:
				return splits[i-1], gId, splits[i+2]
			case EXHENTAI_DOMAIN:
				return splits[i-1], gId, splits[i+2]
			}
			break
		}
	}
	return "", 0, ""
}

func UrlGetPTokenGIdPNum(u string) (domain Domain, pToken string, gId int, pNum int) {
	// https://e-hentai.org/s/{page_token}/{gallery_id}-{pagenumber}
	// https://e-hentai.org/s/0b2127ea05/3138775-8
	u = strings.TrimRight(u, "/")
	splits := strings.Split(u, "/")
	for i, s := range splits {
		if s == "s" {
			// i   s
			// i+1 0b2127ea05
			// i+2 3138775-8
			if len(splits) < i+3 {
				break
			}
			tail := strings.Split(splits[i+2], "-")
			if len(tail) < 2 {
				break
			}
			gId, err := strconv.Atoi(tail[0])
			if err != nil {
				break
			}
			pIndex, err := strconv.Atoi(tail[1])
			if err != nil {
				break
			}
			switch splits[i-1] {
			case EHENTAI_DOMAIN:
				return splits[i-1], splits[i+1], gId, pIndex
			case EXHENTAI_DOMAIN:
				return splits[i-1], splits[i+1], gId, pIndex
			}
			break
		}
	}
	return "", "", 0, 0
}

func httpGet(ctx context.Context, url *netUrl.URL) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	if cookie.Ok() {
		req.Header.Set("Cookie", cookie.String())
	}
	return http.DefaultClient.Do(req)
}

func httpGetDoc(ctx context.Context, url *netUrl.URL) (*goquery.Document, error) {
	resp, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	if sadPandaCheck(doc) {
		return nil, ErrSadPanda
	}
	return doc, nil
}

func sadPandaCheck(doc *goquery.Document) bool {
	return doc.Find("head").Length() == 0 && doc.Find("body").Length() == 0
}
