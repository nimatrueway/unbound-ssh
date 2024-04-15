package service

import "net"

type Server interface {
	Serve(l net.Listener) error
	Close() error
}
