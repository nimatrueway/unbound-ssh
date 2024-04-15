package codec

import (
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"io"
)

func WrapCodec(r io.Reader, w io.Writer) io.ReadWriter {
	switch config.Config.Transfer.Codec {
	case config.Plain:
	case config.Hex:
		r = HexReader(r)
		w = HexWriter(w)
	default:
		panic("invalid codec")
	}
	return core.NewReadWriter(r, w)
}
