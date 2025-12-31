package EHentai

import (
	"context"
	"errors"
)

const API_URL = `https://api.e-hentai.org/api.php`

var (
	ErrNoGalleryProvided = errors.New("no gallery provided")
	ErrNoPageProvided    = errors.New("no page provided")
)

// PostGalleryMetadata posts to the official API and returns gallery metadata
func PostGalleryMetadata(ctx context.Context, g ...GIdList) (metadatas []GalleryMetadata, err error) {
	type request struct {
		Method    string  `json:"method"`
		GIdList   [][]any `json:"gidlist"`
		Namespace int     `json:"namespace"`
	}
	type response struct {
		GMetadata []GalleryMetadata `json:"gmetadata"`
	}

	defer func() {
		if err == nil {
			// 缓存元数据
			for _, g := range metadatas {
				// 接口未返回错误时存入
				if g.Error == "" {
					metaCacheWrite(g.GId, &g, nil)
				}
			}
		}
	}()

	if len(g) == 0 {
		err = wrapErr(ErrNoGalleryProvided, nil)
		return
	}

	reqBody := request{
		Method:    "gdata",
		GIdList:   make([][]any, 0, len(g)),
		Namespace: 1,
	}
	for _, gallery := range g {
		reqBody.GIdList = append(reqBody.GIdList, []any{gallery.GalleryId, gallery.GalleryToken})
	}

	resp, err := post[response](ctx, API_URL, reqBody)
	if err != nil {
		return
	}
	metadatas = resp.GMetadata
	return
}

// PostGalleryToken posts to the official API and returns gallery token
func PostGalleryToken(ctx context.Context, p ...PageList) (tokens []TokenList, err error) {
	type request struct {
		Method   string  `json:"method"`
		PageList [][]any `json:"pagelist"`
	}
	type response struct {
		TokenLists []TokenList `json:"tokenlist"`
	}

	if len(p) == 0 {
		err = wrapErr(ErrNoPageProvided, nil)
		return
	}

	reqBody := request{
		Method:   "gtoken",
		PageList: make([][]any, 0, len(p)),
	}
	for _, page := range p {
		reqBody.PageList = append(reqBody.PageList, []any{page.GalleryId, page.PageToken, page.PageNum})
	}

	resp, err := post[response](ctx, API_URL, reqBody)
	if err != nil {
		return
	}
	tokens = resp.TokenLists
	return
}
