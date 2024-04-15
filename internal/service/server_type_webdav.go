package service

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/net/webdav"
	"net"
	"net/http"
)

type WebdavServer struct {
	server *http.Server
	l      net.Listener
}

const AccessPath = "/"

func CreateWebdavServer() (Server, error) {
	handler := webdav.Handler{
		FileSystem: webdav.Dir(AccessPath),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				logrus.Warnf("WEBDAV [%s]: %s, ERROR: %s\n", r.Method, r.URL, err)
			} else {
				logrus.Tracef("WEBDAV [%s]: %s \n", r.Method, r.URL)
			}
		},
	}
	server := http.Server{
		Handler: &handler,
	}
	return &WebdavServer{server: &server}, nil
}

func (f *WebdavServer) Serve(l net.Listener) error {
	f.l = l
	if err := f.server.Serve(l); err != nil {
		return err
	}
	return nil
}

func (f *WebdavServer) Close() (err error) {
	errs := make([]error, 0)

	if err = f.l.Close(); err != nil {
		errs = append(errs, err)
	}

	if err = f.server.Close(); err != nil {
		errs = append(errs, err)
	}

	return nil
}
