package core

import (
	"io"
	"strings"
)

func NewReadWriter(r io.Reader, w io.Writer) io.ReadWriter {
	return &readWriter{r, w}
}

type readWriter struct {
	io.Reader
	io.Writer
}

// ------------------------------------------------------------------------

func NopeRwCloser(rw io.ReadWriter) io.ReadWriteCloser {
	return WithRwCloser(rw, func() error { return nil })
}

func WithRwCloser(rw io.ReadWriter, f func() error) io.ReadWriteCloser {
	return &withRwCloser{rw, f}
}

type withRwCloser struct {
	io.ReadWriter
	f func() error
}

func (c *withRwCloser) Close() error {
	return c.f()
}

// ------------------------------------------------------------------------

func WithRCloser(r io.Reader, f func() error) io.ReadCloser {
	return &withRCloser{r, f}
}

type withRCloser struct {
	io.Reader
	f func() error
}

func (c *withRCloser) Close() error {
	return c.f()
}

// ------------------------------------------------------------------------

func IsAlreadyClosed(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "use of closed") || strings.Contains(err.Error(), "already closed"))
}
