package EHentai

import (
	"context"
	"iter"
)

type dlPage struct {
	url  string
	page PageData
	err  chan error
}

func (p *dlPage) download(ctx context.Context) error {
	data, err := downloadPage(ctx, p.url)
	if err != nil {
		return err
	}
	p.page = data
	return nil
}

func (p *dlPage) done() bool {
	return len(p.page.Data) != 0 // 未完成的一定为空
}

type dlJob struct {
	pages  []*dlPage
	ctx    context.Context
	cancel context.CancelFunc
	err    error
}

func (j *dlJob) init(pageUrls []string) {
	j.ctx, j.cancel = context.WithCancel(context.Background())

	if j.err != nil {
		return // do nothing
	}

	if len(j.pages) > 0 {
		return
	}

	j.pages = make([]*dlPage, len(pageUrls))
	for i := range pageUrls {
		j.pages[i] = &dlPage{
			url: pageUrls[i],
		}
	}
}

func (j *dlJob) startBackground() {
	if j.err != nil {
		return
	}

	for _, page := range j.pages {
		page.err = make(chan error, 1)
	}

	go func() {
		limiter := newLimiter(threads)
		for _, page := range j.pages {
			if page.done() {
				close(page.err)
				continue
			}

			limiter.acquire()
			go func(page *dlPage) {
				defer limiter.release()
				page.err <- page.download(j.ctx)
			}(page)
		}
	}()
}

func (j *dlJob) downloadIter() iter.Seq2[PageData, error] {
	return func(yield func(PageData, error) bool) {
		defer j.cancel()

		if j.err != nil {
			yield(PageData{}, j.err)
			return
		}
		for _, dlPage := range j.pages {
			err, ok := <-dlPage.err
			if !ok { // 已下载, 下载方关闭了 err
				continue
			}

			_, pToken, gId, pNum := UrlGetPTokenGIdPNum(dlPage.url)
			if !yield(PageData{PageList{pToken, gId, pNum}, dlPage.page.Data}, err) {
				return
			}
		}
	}
}

type limiter struct {
	max int
	sem chan struct{}
}

func newLimiter(max int) *limiter {
	return &limiter{
		max: max,
		sem: make(chan struct{}, max),
	}
}

func (l *limiter) acquire() {
	l.sem <- struct{}{}
}

func (l *limiter) release() {
	<-l.sem
}
