package test

import (
	"bufio"
	"context"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/internal/mode/listen"
	"github.com/nimatrueway/unbound-ssh/internal/mode/spy"
	"github.com/nimatrueway/unbound-ssh/internal/service"
	"github.com/nimatrueway/unbound-ssh/test/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestYamuxManagerEchoService(t *testing.T) {
	logrus.SetLevel(logrus.WarnLevel)

	for i := 0; i < 5; i++ {
		echoServiceAddr := config.NewAddress("unix", utils.NewSocket(t, fmt.Sprintf("iter%d", i), ""))
		echoService := config.ServiceDescription{
			Type: config.Echo,
			Bind: echoServiceAddr,
		}
		services := []config.ServiceDescription{echoService}

		serverConn, clientConn := utils.NewTappedConnectionPair(t, "")
		clientConn = DoNotCloseConnection(clientConn)
		serverConn = DoNotCloseConnection(serverConn)

		group := errgroup.Group{}
		listenCtx, stopper := context.WithCancel(context.Background())

		// mimic listen-mode
		group.Go(func() error {
			serviceManager, err := service.NewListenServiceManager(services)
			require.NoError(t, err)

			ctxReader := core.NewContextReader(clientConn)
			mode := listen.CreateConnectedState(ctxReader, clientConn)
			err = mode.ListenAndServe(listenCtx, serviceManager)
			require.NoError(t, err)

			err = clientConn.Close() // to be able to drain it

			require.NoError(t, err)
			DrainExpectEmpty(t, ctxReader, utils.AssertionTimeout)

			return nil
		})

		// mimic spy-mode
		group.Go(func() error {
			serviceManager, err := service.NewSpyServiceManager(services)
			require.NoError(t, err)

			mode := spy.NewConnectedState(core.NewContextReader(serverConn), serverConn)
			err = mode.ListenAndServe(context.Background(), serviceManager)
			require.NoError(t, err)

			return nil
		})

		conn, err := utils.KeepTrying(func() (net.Conn, error) {
			return net.Dial("unix", echoServiceAddr.String())
		})
		require.NoError(t, err)

		_, err = conn.Write([]byte("hello\n"))
		require.NoError(t, err)

		connReader := bufio.NewReader(conn)
		line, err := connReader.ReadSlice('\n')
		require.NoError(t, err)
		require.Equal(t, "received: hello\n", string(line))

		go func() {
			time.Sleep(100 * time.Millisecond) // wait for spy to spit out connection close stuff
			stopper()
		}()

		err = group.Wait()
		require.NoError(t, err)
	}
}

func DrainExpectEmpty(t *testing.T, reader core.ContextBindingReader, timeout time.Duration) {
	drainContext, drainCancel := context.WithCancel(context.Background())
	drained := new(strings.Builder)
	go func() {
		time.Sleep(timeout)
		drainCancel()
	}()

	_, err := io.Copy(drained, reader.BindTo(drainContext))
	if !core.IsAlreadyClosed(err) {
		require.NoError(t, err)
	}
	require.Equal(t, "", drained.String())
}
