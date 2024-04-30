package service

import (
	"github.com/elazarl/goproxy"
	"github.com/ztrue/tracerr"
	"net"
	"net/http"
)

type httpServer struct {
	server *http.Server
	l      net.Listener
}

func CreateHttpServer() (Server, error) {
	proxy := httpServer{
		server: &http.Server{
			Handler: goproxy.NewProxyHttpServer(),
		},
	}
	return &proxy, nil
}

func (f *httpServer) Serve(l net.Listener) error {
	f.l = l
	if err := f.server.Serve(l); err != nil {
		return err
	}
	return nil
}

func (f *httpServer) Close() (err error) {
	errs := make([]error, 0)

	if err = f.l.Close(); err != nil {
		errs = append(errs, err)
	}

	if err = f.server.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return tracerr.Errorf("failed to close http proxy server: %v", errs)
	} else {
		return nil
	}
}
