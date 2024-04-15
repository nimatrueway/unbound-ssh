package test

import (
	"context"
	"github.com/hashicorp/yamux"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/test/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"io"
	"sync"
	"testing"
)

func TestYamuxCloseConnection(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)
	serverConn, clientConn := utils.NewTappedConnectionPair(t, "")

	group := sync.WaitGroup{}
	group.Add(2)

	// mimic listen-mode
	go func() {
		defer group.Done()

		conn := DoNotCloseConnection(clientConn)
		session, err := yamux.Client(conn, yamux.DefaultConfig())
		require.NoError(t, err)

		stream, err := session.OpenStream()
		require.NoError(t, err)

		_, err = stream.Write([]byte("hello"))
		require.NoError(t, err)

		err = stream.Close()
		require.NoError(t, err)

		err = session.Close()
		require.NoError(t, err)

		_, err = io.ReadAll(conn)
		require.NoError(t, err)
	}()

	// mimic spy-mode
	go func() {
		defer group.Done()

		conn := DoNotCloseConnection(serverConn)
		session, err := yamux.Server(conn, yamux.DefaultConfig())
		require.NoError(t, err)

		stream, err := session.AcceptStream()
		require.NoError(t, err)

		buf := make([]byte, 5)
		_, err = io.ReadAtLeast(stream, buf, 5)
		require.NoError(t, err)
		require.Equal(t, "hello", string(buf))

		_, err = stream.Read(buf)
		require.Equal(t, io.EOF, err)

		err = stream.Close()
		require.NoError(t, err)

		err = session.Close()
		require.NoError(t, err)
	}()

	group.Wait()
}

// mimic how we create a virtual connection on an interactive shell text stream
// where we do not actually close the connection, but just interrupt the underlying reader and writer
func DoNotCloseConnection(rwc io.ReadWriteCloser) io.ReadWriteCloser {
	r := core.NewContextReader(rwc).BindTo(context.Background())
	w := core.NewContextBoundWriter(rwc, context.Background())
	return core.WithRwCloser(core.NewReadWriter(r, w), func() error {
		err1 := r.Close()
		err2 := w.Close()
		if err1 != nil {
			return err1
		} else if err2 != nil {
			return err2
		} else {
			return nil
		}
	})
}
