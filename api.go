package EHentai

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const API_URL = `https://api.e-hentai.org/api.php`

var (
	ErrNoGalleryProvided = errors.New("no gallery provided")
	ErrNoPageProvided    = errors.New("no page provided")
)

// PostGalleryMetadata posts to the official API and returns gallery metadata
func PostGalleryMetadata(g ...GIdList) (resp *GalleryMetadataResponse, err error) {
	defer func() {
		if resp != nil && err == nil {
			// 缓存元数据
			for _, g := range resp.GMetadata {
				if g.Error != "" {
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

	return post[GalleryMetadataResponse](API_URL, reqBody)
}

// PostGalleryToken posts to the official API and returns gallery token
func PostGalleryToken(p ...PageList) (*GalleryTokenResponse, error) {
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

	return post[GalleryTokenResponse](API_URL, reqBody)
}

func post[T any](url string, body any) (*T, error) {
	r, w := io.Pipe()
	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, r)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respBody T
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}
