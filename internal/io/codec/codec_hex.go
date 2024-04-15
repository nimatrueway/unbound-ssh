package codec

import (
	"encoding/hex"
	stdio "io"
)

type hexEncoderWriter struct {
	*genericEncoderWriter
}

func HexWriter(dst stdio.Writer) stdio.Writer {
	converter := func(p []byte) ([]byte, error) {
		return hex.AppendEncode(make([]byte, 0), p), nil
	}
	return &hexEncoderWriter{&genericEncoderWriter{Writer: dst, Name: "hex", Converter: converter}}
}

// ----------------------------------------------------------------------------------------------------------------

type hexDecoderReader struct {
	*genericDecoderReader
}

func HexReader(src stdio.Reader) stdio.Reader {
	converter := func(p []byte) ([]byte, error) {
		return hex.AppendDecode(make([]byte, 0), p)
	}
	return &hexDecoderReader{&genericDecoderReader{Reader: src, Name: "hex", ChunkSize: 2, Converter: converter}}
}
