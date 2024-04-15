package codec

import (
	"bytes"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
	stdio "io"
	"testing"
	"time"
)

type Codec lo.Tuple3[string, func(reader stdio.Reader) stdio.Reader, func(reader stdio.Writer) stdio.Writer]

func TestBasicCodec(t *testing.T) {
	for _, codec := range []Codec{{"hex", HexReader, HexWriter}} {
		codecR := codec.B
		codecW := codec.C

		for i := 0; i < 10000; i++ {
			channelReader := core.NewChannelReader()

			str := randomString(false)
			strSlices := core.RandomlySlice(str)
			for _, slice := range strSlices {
				channelReader.WriteString(slice)
			}
			channelReader.Fail(stdio.EOF)

			// encode channelReader content chunk by chunk
			encoded := bytes.NewBufferString("")
			buf := make([]byte, 1024)
			for {
				n, err := channelReader.Read(buf)
				if err == stdio.EOF {
					break
				}
				encodeW := codecW(encoded)
				_, err = encodeW.Write(buf[:n])
				require.NoError(t, err)
			}

			// random resize chunks
			rechunked := core.NewChannelReader()
			encSlices := core.RandomlySlice(encoded.String())
			for _, slice := range encSlices {
				rechunked.WriteString(slice)
			}
			rechunked.Fail(stdio.EOF)

			// read and decode channelReader content chunk by chunk
			reader := codecR(rechunked)
			output, err := stdio.ReadAll(reader)
			require.Nil(t, err)
			outputStr := string(output)
			if str != outputStr {
				require.FailNowf(t, "", "codec %s failed to encode %#v string that is strSlices into %#v\n\nor decode after the encoded sequence %#v got resliced to %#v\n\nresult was %#v", codec.A, str, strSlices, encoded.String(), encSlices, outputStr)
			}
		}
	}
}

func randomString(onlyAlphanumeric bool) string {
	random := rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
	n := random.Intn(50) + 1
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		var c int
		if onlyAlphanumeric {
			possibility := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
			c = int(possibility[rand.Intn(len(possibility))])
		} else {
			c = rand.Intn(255)
		}

		buf[i] = byte(c)
	}
	return string(buf)
}
