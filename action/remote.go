package action

import "net/http"

type IRemoteIPProvider interface {
	Get(req *http.Request) string
}

type _HeaderRemoteIPProvider struct {
	headers []string
}

func (p *_HeaderRemoteIPProvider) Get(req *http.Request) string {
	for _, h := range p.headers {
		if ip := req.Header.Get(h); ip != "" {
			return ip
		}
	}
	return req.RemoteAddr
}

func NewHeadersRemoteIPProvider(headers ...string) IRemoteIPProvider {
	return &_HeaderRemoteIPProvider{headers: headers}
}
