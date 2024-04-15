package core

import (
	"context"
	"errors"
	"io"
)

type ContextBoundWriter struct {
	w             io.Writer
	ctx           context.Context
	cancelCauseFn context.CancelCauseFunc
}

func NewContextBoundWriter(w io.Writer, ctx context.Context) *ContextBoundWriter {
	newCtx, cancelCauseFn := context.WithCancelCause(ctx)
	return &ContextBoundWriter{
		w:             w,
		ctx:           newCtx,
		cancelCauseFn: cancelCauseFn,
	}
}

func (sr *ContextBoundWriter) Context() context.Context {
	return sr.ctx
}

func (sr *ContextBoundWriter) Close() error {
	sr.CancelWithCause(errors.New("use of closed network connection"))
	return nil
}

func (sr *ContextBoundWriter) Cancel() {
	sr.cancelCauseFn(context.Canceled)
}

func (sr *ContextBoundWriter) CancelWithCause(cause error) {
	sr.cancelCauseFn(cause)
}

func (sr *ContextBoundWriter) Write(p []byte) (int, error) {
	if sr.ctx.Err() != nil {
		return 0, context.Cause(sr.ctx)
	}

	return sr.w.Write(p)
}

// ----------------------------------------------------------------------------

type SimpleContextBindingWriter struct {
	w io.Writer
}

func (s *SimpleContextBindingWriter) NewContextBindingWriter(w io.Writer) *SimpleContextBindingWriter {
	return &SimpleContextBindingWriter{w: w}
}

func (s *SimpleContextBindingWriter) BindTo(ctx context.Context) *ContextBoundWriter {
	return NewContextBoundWriter(s.w, ctx)
}
