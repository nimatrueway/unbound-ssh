package core

import (
	"context"
	"github.com/sirupsen/logrus"
	"io"
)

type ContextReadCloser struct {
	io.ReadCloser
}

func (c *ContextReadCloser) BindTo(ctx context.Context) io.ReadCloser {
	go func() {
		select {
		case <-ctx.Done():
			err := c.Close()
			if err != nil && !IsAlreadyClosed(err) {
				logrus.Warnf("Failed to close %s reader: %s", DetermineReaderName(c.ReadCloser), err.Error())
			}
		}
	}()

	return &ContextBoundReadCloser{c, ctx}
}

type ContextBoundReadCloser struct {
	*ContextReadCloser
	ctx context.Context
}

func (c *ContextBoundReadCloser) Read(p []byte) (n int, err error) {
	return c.ContextReadCloser.Read(p)
}
