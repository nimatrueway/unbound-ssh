package service

import (
	"bufio"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	"io"
	"net"
)

type EchoServer struct {
	l           net.Listener
	connections []net.Conn
}

const EchoPattern = "received: %s"

func CreateEchoServer() (Server, error) {
	return &EchoServer{}, nil
}

func (f *EchoServer) Serve(l net.Listener) error {
	for {
		f.l = l
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		f.connections = append(f.connections, conn)

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logrus.Errorf("echo server failed to close connection, err: %v", err)
		}
	}(conn)
	reader := bufio.NewReader(conn)
	for {
		// read client request data
		line, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			if err != io.EOF {
				logrus.Warnf("echo server failed to read data, err: %s", err.Error())
			}
			return
		}
		logrus.Tracef("echo server received request: %#v", line)

		response := fmt.Sprintf(EchoPattern, string(line))
		logrus.Tracef("echo server sent response: %#v", response)
		_, err = conn.Write([]byte(response))
		if err != nil {
			logrus.Warnf("echo server failed to write data, err: %s", err.Error())
			return
		}
	}
}

func (f *EchoServer) Close() (err error) {
	errs := make([]error, 0)

	if err = f.l.Close(); err != nil && !core.IsAlreadyClosed(err) {
		errs = append(errs, err)
	}

	for _, conn := range f.connections {
		err := conn.Close()
		if err != nil && !core.IsAlreadyClosed(err) {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return tracerr.Errorf("failed to close echo server: %v", errs)
	} else {
		return nil
	}
}
