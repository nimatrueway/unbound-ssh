package core

import (
	"context"
	"github.com/samber/mo"
	"io"
)

// ContextReader is a reader that instead of Read([]byte) method
// has a Read(ctx context.Context, p []byte) (int, error) method
// that can be interrupted by the given context.
type ContextReader struct {
	// the underlying Reader to read from
	r io.Reader

	// prepend a single buffered channel that holds the data received through UnreadBytes,
	// or excess data from buffer that could not be fully flushed during the last Read() call
	prepend chan []byte

	// buffer an unbuffered channel that holds the result of the last Read() of the underlying Reader
	// before they are consumed
	buffer chan mo.Result[[]byte]

	// fill an unbuffered channel is used to request buffer to be populated upto the given size
	fill chan int
}

func NewContextReader(r io.Reader) *ContextReader {
	core := ContextReader{
		r:       r,
		prepend: make(chan []byte, 1),
		buffer:  make(chan mo.Result[[]byte]),
		fill:    make(chan int),
	}
	go core.startFiller()
	return &core
}

func (sr *ContextReader) BindToConcrete(ctx context.Context) *ContextBoundReader {
	newCtx, cancelCauseFn := context.WithCancelCause(ctx)
	return &ContextBoundReader{
		r:             sr,
		ctx:           newCtx,
		cancelCauseFn: cancelCauseFn,
	}
}

func (sr *ContextReader) BindTo(ctx context.Context) io.ReadCloser {
	return sr.BindToConcrete(ctx)
}

func (sr *ContextReader) startFiller() {
	for {
		size, ok := <-sr.fill
		if !ok {
			return
		}

		buf := make([]byte, size)
		sr.buffer <- mo.Try(func() ([]byte, error) {
			n, err := sr.r.Read(buf)
			if err != nil {
				return nil, err
			} else {
				return buf[:n], err
			}
		})
	}
}

func (sr *ContextReader) UnreadBytes(p []byte) {
	if len(p) > 0 {
		select {
		case old := <-sr.prepend:
			sr.prepend <- append(p, old...)
		case sr.prepend <- p:
		}
	}
}

func (sr *ContextReader) Read(ctx context.Context, p []byte) (int, error) {
	select {
	case <-ctx.Done():
		return 0, context.Cause(ctx)
	case buf := <-sr.prepend:
		if len(buf) > len(p) {
			sr.UnreadBytes(buf[len(p):])
		}
		return copy(p, buf), nil
	case r := <-sr.buffer:
		if buf, err := r.Get(); err != nil {
			return 0, err
		} else {
			if len(buf) > len(p) {
				sr.UnreadBytes(buf[len(p):])
			}
			return copy(p, buf), nil
		}
	case sr.fill <- len(p):
		return sr.Read(ctx, p)
	}
}
