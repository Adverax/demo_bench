package service

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Messenger interface {
	Query(ctx context.Context, query string) ([]byte, error)
	Test(ctx context.Context, url string) bool
}

type messenger struct {
}

func (messenger *messenger) Query(ctx context.Context, query string) ([]byte, error) {
	const baseYandexURL = "https://yandex.ru/search/touch/?service=www.yandex&ui=webmobileapp.yandex&numdoc=50&lr=213&p=0&"

	params := url.Values{}
	params.Add("text", query)
	u := baseYandexURL + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Charset", "utf-8")
	req.Header.Set("Accept-Charset", "utf-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrInvalidResponse
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return body, nil
}

func (messenger *messenger) Test(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Charset", "utf-8")
	req.Header.Set("Accept-Charset", "utf-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	if resp.StatusCode != http.StatusOK {
		return false
	}

	return true
}

func NewMessenger() Messenger {
	return &messenger{}
}
