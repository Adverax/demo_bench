package service

import (
	"context"
	"errors"
	"sync"
	"time"
)

type statistics struct {
	sync.Mutex
	keys map[string]int
}

func (stats *statistics) assign(key string, val int) {
	stats.Lock()
	defer stats.Unlock()
	stats.keys[key] = val
}

func (stats *statistics) fetch(keys []string) map[string]int {
	stats.Lock()
	defer stats.Unlock()
	var m = make(map[string]int, len(keys))
	for _, key := range keys {
		if val, has := stats.keys[key]; has {
			m[key] = val
		}
	}
	return m
}

type Config struct {
	Timeout int `json:"timeout"`
}

type Service interface {
	Execute(ctx context.Context, query string) (map[string]int, error)
}

type service struct {
	stats     statistics
	messenger Messenger
	config    *Config
}

func (service *service) Execute(
	ctx context.Context,
	query string,
) (map[string]int, error) {
	response, err := service.messenger.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	entries := parseYandexResponse(response)
	if entries.Error != nil {
		return nil, entries.Error
	}

	urls := make([]string, len(entries.Items))
	for i, entry := range entries.Items {
		urls[i] = entry.Url
	}

	return service.Resolve(ctx, urls)
}

func (service *service) Resolve(
	ctx context.Context,
	urls []string,
) (map[string]int, error) {
	has := service.stats.fetch(urls)
	want := diff(urls, has)
	if len(want) == 0 {
		return has, nil
	}
	err := service.updateAll(ctx, want)
	if err != nil {
		return nil, err
	}
	return service.fetch(urls), nil
}

func (service *service) updateAll(
	ctx context.Context,
	want []string,
) error {
	for _, url := range want {
		go service.update(ctx, url)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(service.config.Timeout) * time.Second):
		return nil
	}
}

func (service *service) update(
	ctx context.Context,
	url string,
) {
	var l, r = 1, 1

	for {
		count := service.benchmark(ctx, url, r)
		if count != r {
			break
		}
		r *= 2
	}

	for l < r {
		h := l + (r-l)/2
		count := service.benchmark(ctx, url, h)
		if count < h {
			r = h - 1
		} else {
			l = h + 1
			service.stats.assign(url, count)
		}
	}
}

func (service *service) benchmark(
	ctx context.Context,
	url string,
	n int,
) (count int) {
	var latch sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if service.messenger.Test(ctx, url) {
				latch.Lock()
				defer latch.Unlock()
				count++
			}
		}()
	}
	wg.Wait()
	return
}

func (service *service) fetch(keys []string) map[string]int {
	res := service.stats.fetch(keys)
	for _, key := range keys {
		if _, has := res[key]; !has {
			res[key] = 1
		}
	}
	return res
}

var ErrInvalidResponse = errors.New("invalid http response")

func New(
	messenger Messenger,
	config *Config,
) Service {
	return &service{
		messenger: messenger,
		config:    config,
		stats: statistics{
			keys: make(map[string]int, 32768),
		},
	}
}

func diff(urls []string, has map[string]int) (res []string) {
	for _, u := range urls {
		if _, ok := has[u]; !ok {
			res = append(res, u)
		}
	}
	return
}
