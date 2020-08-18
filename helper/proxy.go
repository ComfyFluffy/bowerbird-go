package helper

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/proxy"
)

func SetTransportProxy(tr *http.Transport, uris ...string) error {
	var uri string
	for _, urir := range uris {
		if urir != "" {
			uri = urir
			break
		}
	}
	if uri == "" {
		return nil
	}

	pr, err := url.Parse(uri)
	if err != nil {
		return err
	}

	switch strings.ToLower(pr.Scheme) {
	case "http":
		hp := http.ProxyURL(pr)
		tr.Proxy = hp
	case "socks5":
		var spauth *proxy.Auth
		spw, _ := pr.User.Password()
		spu := pr.User.Username()
		if spw != "" || spu != "" {
			spauth = &proxy.Auth{User: spu, Password: spw}
		}
		spd, err := proxy.SOCKS5("tcp", pr.Host, spauth, proxy.Direct)
		if err != nil {
			return err
		}
		tr.DialContext = spd.(proxy.ContextDialer).DialContext
	default:
		return errors.New("unsupported proxy protocol: " + pr.Scheme)
	}
	return nil
}
