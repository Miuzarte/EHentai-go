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
func PostGalleryMetadata(ctx context.Context, g ...GIdList) (resp *GalleryMetadataResponse, err error) {
	defer func() {
		if resp != nil && err == nil {
			// 缓存元数据
			for _, g := range resp.GMetadata {
				// 接口未返回错误时存入
				if g.Error == "" {
					metaCacheWrite(g.GId, &g, nil)
				}
			}
		}
	}()

	if len(g) == 0 {
		return nil, wrapErr(ErrNoGalleryProvided, nil)
	}

	reqBody := GalleryMetadataRequest{
		Method:    "gdata",
		GIdList:   make([][]any, 0, len(g)),
		Namespace: 1,
	}
	for _, gallery := range g {
		reqBody.GIdList = append(reqBody.GIdList, []any{gallery.GalleryId, gallery.GalleryToken})
	}

	return post[GalleryMetadataResponse](ctx, API_URL, reqBody)
}

// PostGalleryToken posts to the official API and returns gallery token
func PostGalleryToken(ctx context.Context, p ...PageList) (*GalleryTokenResponse, error) {
	if len(p) == 0 {
		return nil, wrapErr(ErrNoPageProvided, nil)
	}

	reqBody := GalleryTokenRequest{
		Method:   "gtoken",
		PageList: make([][]any, 0, len(p)),
	}
	for _, page := range p {
		reqBody.PageList = append(reqBody.PageList, []any{page.GalleryId, page.PageToken, page.PageNum})
	}

	return post[GalleryTokenResponse](ctx, API_URL, reqBody)
}
