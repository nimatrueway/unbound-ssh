package core

import (
	"context"
	"io"
)

type ContextBoundReader struct {
	r             *ContextReader
	ctx           context.Context
	cancelCauseFn context.CancelCauseFunc
}

func (sr *ContextBoundReader) Context() context.Context {
	return sr.ctx
}

func (sr *ContextBoundReader) Close() error {
	sr.CancelWithCause(io.EOF)
	return nil
}

func (sr *ContextBoundReader) Cancel() {
	sr.cancelCauseFn(context.Canceled)
}

func (sr *ContextBoundReader) CancelWithCause(cause error) {
	sr.cancelCauseFn(cause)
}

func (sr *ContextBoundReader) Read(p []byte) (int, error) {
	return sr.r.Read(sr.ctx, p)
}

func (sr *ContextBoundReader) UnreadBytes(bytes []byte) {
	sr.r.UnreadBytes(bytes)
}
