package codec

import (
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/sirupsen/logrus"
	stdio "io"
)

type genericEncoderWriter struct {
	stdio.Writer
	Converter func([]byte) ([]byte, error)
	Name      string
}

func (w *genericEncoderWriter) Write(p []byte) (int, error) {
	logrus.Tracef("request write to \"%s\" in %s (raw): %#v", core.DetermineWriterName(w.Writer), w.Name, string(p))
	converted, err := w.Converter(p)
	if err != nil {
		return 0, err
	}
	n, err := w.Writer.Write(converted)
	if err != nil {
		return 0, err
	}
	logrus.Tracef("wrote to \"%s\" in %s codec: %#v", core.DetermineWriterName(w.Writer), w.Name, string(converted))

	return n, err
}

// ----------------------------------------------------------------------------------------------------------------

type genericDecoderReader struct {
	stdio.Reader
	Name      string
	ChunkSize int
	Converter func([]byte) ([]byte, error)
	buf       [10]byte // number needs to be ChunkSize - 1
	bufLen    int
}

// p has to be at least ChunkSize bytes long
func (r *genericDecoderReader) Read(p []byte) (n int, err error) {
	if r.bufLen > 0 {
		n, err = r.Reader.Read(p[(r.bufLen):])
	} else {
		n, err = r.Reader.Read(p)
	}

	if err != nil {
		return 0, err
	}

	if r.bufLen > 0 {
		copy(p, r.buf[:r.bufLen])
		n += r.bufLen
		r.bufLen = 0
	}

	if n > 0 {
		extra := n % r.ChunkSize
		n -= extra
		if extra > 0 {
			copy(r.buf[:], p[n:n+extra])
			r.bufLen = extra
		}
	}

	if n > 0 {
		logrus.Tracef("request read from \"%s\" in %s (raw): %#v", core.DetermineReaderName(r.Reader), r.Name, string(p[:n]))
		decoded, err := r.Converter(p[:n])
		if err != nil {
			return 0, err
		}
		logrus.Tracef("read from \"%s\" in %s codec: %#v", core.DetermineReaderName(r.Reader), r.Name, string(decoded))
		return copy(p, decoded), nil
	} else {
		return r.Read(p)
	}
}
