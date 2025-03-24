package EHentai

import (
	"context"
)

type dlPage struct {
	url  string
	err  chan error
	data []byte
}

func (p *dlPage) download(ctx context.Context) error {
	data, err := downloadPage(ctx, p.url)
	if err != nil {
		return err
	}
	p.data = data
	return nil
}

func (p *dlPage) done() bool {
	return len(p.data) != 0 // 未完成的一定为空
}

type dlJob struct {
	pages  []*dlPage
	ctx    context.Context
	cancel context.CancelFunc
}

func (j *dlJob) init(pageUrls []string) {
	j.ctx, j.cancel = context.WithCancel(context.Background())

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
