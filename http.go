package cube

import (
	"net/http"
	"net/url"
)

func NewHttpClientWithProxy(proxyurl *url.URL) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyurl)

	cli := new(http.Client)
	*cli = *http.DefaultClient
	cli.Transport = transport
	return cli
}
