package listen

import (
	"context"
	"github.com/hashicorp/yamux"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/codec"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/internal/mode"
	"github.com/nimatrueway/unbound-ssh/internal/service"
	"github.com/nimatrueway/unbound-ssh/internal/view"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	stdio "io"
)

type ConnectedState struct {
	reader  core.ContextBindingReader
	writer  stdio.Writer
	manager *service.YamuxStreamManager
}

const EndOfText byte = 3 // Ctrl+C in ascii

func CreateConnectedState(r core.ContextBindingReader, w stdio.Writer) ConnectedState {
	return ConnectedState{
		reader: r,
		writer: w,
	}
}

func (ym *ConnectedState) ListenAndServe(ctx context.Context, manager *service.ListenServiceManager) error {
	if ym.manager != nil {
		return tracerr.New("already listening")
	}

	// create a yamux client config
	cfg := yamux.DefaultConfig()
	cfg.StreamOpenTimeout = config.Config.Transfer.ConnectionTimeout
	cfg.StreamCloseTimeout = config.Config.Transfer.ConnectionTimeout
	cfg.ConnectionWriteTimeout = config.Config.Transfer.ConnectionTimeout
	cfg.KeepAliveInterval = config.Config.Transfer.ConnectionTimeout
	cfg.LogOutput = view.LogFile

	// assign rwc
	reader := ym.reader.BindTo(context.Background())
	writer := core.NewContextBoundWriter(ym.writer, context.Background())
	closer := func() {
		logrus.Info("closing virtual connection of listen-mode.")
		_ = reader.Close()
		_ = writer.Close()
	}
	rwc := core.WithRwCloser(codec.WrapCodec(reader, writer), func() error {
		closer()
		return nil
	})
	defer closer()

	// create a yamux client session
	session, err := yamux.Client(rwc, cfg)
	if err != nil {
		return tracerr.Wrap(err)
	}
	logrus.Info("created yamux client session.")

	// forward traffic of the incoming connections to the yamux session
	ym.manager = service.NewYamuxForwarderManager(session, writer)

	// open the control stream
	err = openAndAssignControlStream(ym.manager)
	if err != nil {
		return err
	}

	err = ym.manager.ReceiveAndOpenYamux(ctx, manager)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

func openAndAssignControlStream(ym *service.YamuxStreamManager) error {
	yamuxStream, err := ym.Session.OpenStream()
	if err != nil {
		return tracerr.Wrap(err)
	}
	logrus.Debug("established control stream for this yamux session")

	// send the first command as HelloRequest
	ym.ControlStream = service.NewControlStream(yamuxStream)

	sendHello := service.RpcCreateInvoker[mode.HelloExchange](ym.ControlStream)
	_, err = sendHello(mode.HelloRequest{})
	if err != nil {
		logrus.Errorf("failed to complete HelloExchange through control stream: %s", err.Error())
		return err
	}

	logrus.Info("successfully completed HelloExchange through control stream.")
	return nil
}
