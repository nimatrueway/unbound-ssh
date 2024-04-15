package term

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
)

type ReadInBackground struct {
	Stdin     io.Reader
	reactions map[byte]func()
	buf       []byte
}

func NewReadInBackground(stdin io.Reader) ReadInBackground {
	return ReadInBackground{Stdin: stdin, reactions: map[byte]func(){}, buf: make([]byte, 1)}
}

func (r *ReadInBackground) ReactTo(key byte, with func()) *ReadInBackground {
	r.reactions[key] = with
	return r
}

func (r *ReadInBackground) Start(ctx context.Context) {
	go func() {
		for {
			c, err := r.fetchByte(ctx)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					logrus.Warnf("error reading from stdin: %s", err.Error())
				}
				return
			}

			if reaction, ok := r.reactions[c]; ok {
				logrus.Debug(fmt.Sprintf("reacting to received key: %x [%c]", c, c))
				reaction()
			}
		}
	}()
}

func (r *ReadInBackground) fetchByte(ctx context.Context) (byte, error) {
	err := ctx.Err()
	if err == nil {
		n := 0
		for n == 0 && err == nil {
			n, err = r.Stdin.Read(r.buf)
		}
	}
	return r.buf[0], err
}
