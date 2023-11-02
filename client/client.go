package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

func NewClient(proxyUrl, proxyAuth string) *http.Client {
	return &http.Client{
		Transport: &transport{
			proxyUrl:  proxyUrl,
			proxyAuth: proxyAuth,
			Transport: http.DefaultTransport,
		},
	}
}

type transport struct {
	proxyUrl  string
	proxyAuth string
	Transport http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	pUrl, err := url.Parse(t.proxyUrl)
	if err != nil {
		return nil, errors.New("parse proxy target error")
	}

	origin := fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)

	req.Host = pUrl.Host
	req.URL.Host = pUrl.Host
	req.Header.Add("X-Proxy-Target", origin)
	req.Header.Add("X-Proxy-Auth", t.proxyAuth)
	return t.Transport.RoundTrip(req)
}
