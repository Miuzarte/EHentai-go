package EHentai

import (
	"bytes"
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

type Torrent struct {
	Hash  string `json:"hash"`
	Added string `json:"added"`
	Name  string `json:"name"`
	TSize string `json:"tsize"`
	FSize string `json:"fsize"`
}

type GalleryMetadata struct {
	GId          int       `json:"gid"`
	Token        string    `json:"token"`
	ArchiverKey  string    `json:"archiver_key"`
	Title        string    `json:"title"`
	TitleJpn     string    `json:"title_jpn"`
	Category     string    `json:"category"`
	Thumb        string    `json:"thumb"`
	Uploader     string    `json:"uploader"`
	Posted       string    `json:"posted"`
	FileCount    string    `json:"filecount"`
	FileSize     int       `json:"filesize"`
	Expunged     bool      `json:"expunged"`
	Rating       string    `json:"rating"`
	TorrentCount string    `json:"torrentcount"`
	Torrents     []Torrent `json:"torrents"`
	Tags         []string  `json:"tags"`
	ParentGId    string    `json:"parent_gid"`
	ParentKey    string    `json:"parent_key"`
	FirstGId     string    `json:"first_gid"`
	FirstKey     string    `json:"first_key"`
	Error        string    `json:"error"`
}

type GalleryMetadataResponse struct {
	GMetadata []GalleryMetadata `json:"gmetadata"`
}

type GalleryMetadataRequest struct {
	Method    string  `json:"method"`
	GIdList   [][]any `json:"gidlist"`
	Namespace int     `json:"namespace"`
}

// https://e-hentai.org/g/{gallery_id}/{gallery_token}/
type GIdList struct {
	Id    int
	Token string
}

// PostGalleryMetadata returns metadata of the provided galleries
func PostGalleryMetadata(g ...GIdList) (*GalleryMetadataResponse, error) {
	if len(g) == 0 {
		return nil, ErrNoGalleryProvided
	}

	reqBody := GalleryMetadataRequest{
		Method:    "gdata",
		Namespace: 1,
	}
	for _, gallery := range g {
		reqBody.GIdList = append(reqBody.GIdList, []any{gallery.Id, gallery.Token})
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, API_URL, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var metadataResponse GalleryMetadataResponse
	err = json.Unmarshal(respBytes, &metadataResponse)
	if err != nil {
		return nil, err
	}
	return &metadataResponse, nil
}

type TokenList struct {
	GId   int    `json:"gid"`
	Token string `json:"token"`
}

type GalleryTokenResponse struct {
	TokenLists []TokenList `json:"tokenlist"`
}

type GalleryTokenRequest struct {
	Method   string  `json:"method"`
	PageList [][]any `json:"pagelist"`
}

// https://e-hentai.org/s/{page_token}/{gallery_id}-{pagenumber}
type PageList struct {
	PToken string
	GId    int
	PIndex int
}

// PostGalleryToken returns token of the provided pages
func PostGalleryToken(p ...PageList) (*GalleryTokenResponse, error) {
	if len(p) == 0 {
		return nil, ErrNoPageProvided
	}

	reqBody := GalleryTokenRequest{
		Method: "gtoken",
	}
	for _, page := range p {
		reqBody.PageList = append(reqBody.PageList, []any{page.GId, page.PToken, page.PIndex})
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, API_URL, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tokenResponse GalleryTokenResponse
	err = json.Unmarshal(respBytes, &tokenResponse)
	if err != nil {
		return nil, err
	}
	return &tokenResponse, nil
}
