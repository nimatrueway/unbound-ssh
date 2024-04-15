package service

import (
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	"net"
	"os"
)

type SpyServiceManager struct {
	services []config.ServiceDescription
	servers  []Server
	addr     []net.Addr
}

func NewSpyServiceManager(services []config.ServiceDescription) (*SpyServiceManager, error) {
	instance := SpyServiceManager{services: services}
	if err := instance.launch(); err != nil {
		return nil, err
	}
	return &instance, nil
}

func (sm *SpyServiceManager) Addr(idx int) net.Addr {
	return sm.addr[idx]
}

func (sm *SpyServiceManager) launch() (err error) {
	defer func() {
		if err != nil {
			_ = sm.Close()
		}
	}()

	for i := range sm.services {
		var addr net.Addr
		var server Server
		addr, server, err = sm.justLaunch(i)
		if err != nil {
			logrus.Errorf("error launching %s server: %s", sm.services[i].Type, err.Error())
			return err
		}
		sm.addr = append(sm.addr, addr)
		sm.servers = append(sm.servers, server)
	}

	return nil
}

func (sm *SpyServiceManager) Close() error {
	errs := make([]error, 0)

	for i := range sm.servers {
		if sm.servers[i] != nil {
			err := sm.servers[i].Close()
			if err != nil {
				logrus.Warnf("error closing server#%d (%s): %s", i, sm.services[i].Type, err.Error())
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return tracerr.Errorf("failed to close service manager: %v", errs)
	} else {
		return nil
	}
}

func (sm *SpyServiceManager) justLaunch(idx int) (net.Addr, Server, error) {
	service := sm.services[idx]
	if service.Type == config.PortForward {
		return &service.Destination, nil, nil
	}

	// launch internal server
	listener, err := temporaryListenAddr()
	if err != nil {
		return nil, nil, tracerr.Wrap(err)
	}
	var server Server
	if service.Type == config.EmbeddedWebdav {
		server, err = CreateWebdavServer()
	} else if service.Type == config.EmbeddedSsh {
		server, err = CreateSshServer(service.Certificate)
	} else if service.Type == config.Echo {
		server, err = CreateEchoServer()
	}

	if err != nil {
		logrus.Errorf("error creating %s server: %s", service.Type, err.Error())
		return nil, nil, tracerr.Wrap(err)
	}
	go func() {
		err = server.Serve(listener)
		if err != nil {
			if !core.IsAlreadyClosed(err) {
				logrus.Errorf("error serving %s server: %s", service.Type, err.Error())
			}
		} else {
			logrus.Infof("%s server started on: %s", service.Type, listener.Addr())
		}
	}()

	return listener.Addr(), server, nil
}

func temporaryListenAddr() (net.Listener, error) {
	// because file walls may block listening to ports
	// listener, err := net.Listen("tcp", ":0")

	f, err := os.CreateTemp("", "unbound-ssh-service-*.sock")
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	err = f.Close()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	err = os.Remove(f.Name())
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	listener, err := net.Listen("unix", f.Name())
	return listener, err
}
