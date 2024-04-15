package core

import (
	"context"
	"io"
)

type ContextBindingReader interface {
	BindTo(ctx context.Context) io.ReadCloser
}

type ContextBindingWriter interface {
	BindTo(ctx context.Context) io.WriteCloser
}
