package main

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	if len(via) >= 2 {
		return fmt.Errorf("attempted redirect to %s", req.URL)
	}
	return nil
}

func (app *application) client() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          25,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: redirectPolicyFunc,
	}
}
