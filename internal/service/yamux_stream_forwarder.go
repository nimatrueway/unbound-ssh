package service

import (
	"context"
	"github.com/hashicorp/yamux"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	"io"
	"net"
)

type YamuxForwarder struct {
	conn   net.Conn
	stream *yamux.Stream
}

func NewYamuxForwarder(conn net.Conn, stream *yamux.Stream) YamuxForwarder {
	return YamuxForwarder{
		conn:   conn,
		stream: stream,
	}
}

func (yf *YamuxForwarder) start(ctx context.Context) {
	go func() {
		// we used io.NopCloser(_) on yf.stream to prevent closing the stream upon yamux shutdown on ?-mode side
		// otherwise it will send FIN packets to the other side, which leads to gibberish printing on console
		err := core.DuplexCopy(ctx, yf.conn, &(core.ContextReadCloser{ReadCloser: io.NopCloser(yf.stream)}), yf.stream, &(core.ContextReadCloser{ReadCloser: yf.conn}))
		if err != nil && ctx.Err() != nil && !core.IsAlreadyClosed(err) {
			logrus.Errorf("failed to forward traffic from/to yamux stream [%d]: %s", yf.stream.StreamID(), err.Error())
		}
	}()
}

func (yf *YamuxForwarder) Close() error {
	var errs []error

	if err := yf.stream.Close(); err != nil {
		errs = append(errs, tracerr.Errorf("failed to close yamux stream: %s", err.Error()))
	}

	if err := yf.conn.Close(); err != nil && !core.IsAlreadyClosed(err) {
		errs = append(errs, tracerr.Errorf("failed to close connection: %s", err.Error()))
	}

	if len(errs) > 0 {
		return tracerr.Errorf("failed to close yamux forwarder: %v", errs)
	} else {
		return nil
	}
}
