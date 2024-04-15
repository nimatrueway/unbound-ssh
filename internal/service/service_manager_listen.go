package service

import (
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	"io"
	"net"
	"sync"
)

type ListenServiceManager struct {
	services []config.ServiceDescription
	listener []net.Listener
	accepted chan lo.Tuple2[net.Conn, int]
}

func NewListenServiceManager(services []config.ServiceDescription) (*ListenServiceManager, error) {
	cloned := make([]config.ServiceDescription, len(services))
	copy(cloned, services)

	instance := ListenServiceManager{
		services: services,
		accepted: make(chan lo.Tuple2[net.Conn, int]),
	}
	err := instance.launch()
	if err != nil {
		return nil, err
	}
	instance.acceptLoop()
	return &instance, nil
}

func (lsm *ListenServiceManager) Accept() (net.Conn, int, error) {
	for {
		tuple, ok := <-lsm.accepted
		if ok {
			return tuple.A, tuple.B, nil
		} else {
			return nil, -1, io.EOF
		}
	}
}

func (lsm *ListenServiceManager) acceptLoop() {
	if lsm.listener == nil {
		// as a consequence Accept() will not return until the listener is closed
		return
	}

	group := sync.WaitGroup{}
	group.Add(len(lsm.services))
	for i, listener := range lsm.listener {
		go func(i int, listener net.Listener) {
			defer group.Done()
			for {
				conn, err := listener.Accept()
				if err != nil {
					if !core.IsAlreadyClosed(err) {
						logrus.Warnf("error accepting connection on listener#%d (%s://%s): %s", i, lsm.services[i].Bind.Network(), lsm.services[i].Bind.String(), err.Error())
					}
					break
				}
				lsm.accepted <- lo.Tuple2[net.Conn, int]{A: conn, B: i}
			}
		}(i, listener)
	}
	go func() {
		group.Wait()
		close(lsm.accepted)
	}()
}

func (lsm *ListenServiceManager) launch() (err error) {
	defer func() {
		if err != nil {
			_ = lsm.Close()
		}
	}()

	for _, service := range lsm.services {
		var listener net.Listener
		listener, err = net.Listen(service.Bind.Network(), service.Bind.String())
		if err != nil {
			return tracerr.Wrap(err)
		}
		lsm.listener = append(lsm.listener, listener)
	}
	return nil
}

func (lsm *ListenServiceManager) Close() error {
	errs := make([]error, 0)

	for i, listener := range lsm.listener {
		err := listener.Close()
		if err != nil && !core.IsAlreadyClosed(err) {
			logrus.Warnf("error closing listener#%d (%s://%s): %s", i, lsm.services[i].Bind.Network(), lsm.services[i].Bind.String(), err.Error())
			errs = append(errs, err)
		}
	}

	if lsm.listener == nil {
		// release Accept() if there was no listener to begin with
		close(lsm.accepted)
	}

	if len(errs) > 0 {
		return tracerr.Errorf("failed to close service manager: %v", errs)
	} else {
		return nil
	}
}
