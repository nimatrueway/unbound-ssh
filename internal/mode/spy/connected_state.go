package spy

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
	session *yamux.Session
	manager *service.YamuxStreamManager
}

func NewConnectedState(r core.ContextBindingReader, w stdio.Writer) ConnectedState {
	return ConnectedState{
		reader: r,
		writer: w,
	}
}

func (ym *ConnectedState) ListenAndServe(ctx context.Context, serviceManager *service.SpyServiceManager) error {
	if ym.manager != nil {
		return tracerr.New("already listening")
	}

	// create yamux server config
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
		logrus.Info("closed virtual connection of spy-mode.")
		_ = reader.Close()
		_ = writer.Close()
	}
	rwc := core.WithRwCloser(codec.WrapCodec(reader, writer), func() error {
		closer()
		return nil
	})
	defer closer()

	// create yamux server session
	session, err := yamux.Server(rwc, cfg)
	if err != nil {
		return tracerr.Wrap(err)
	}
	logrus.Info("created yamux server session.")

	// forward received yamux session to addr
	ym.manager = service.NewYamuxForwarderManager(session, writer)

	// open the accept stream
	err = acceptAndAssignControlStream(ym.manager)
	if err != nil {
		return err
	}

	err = ym.manager.AcceptYamuxAndForward(ctx, serviceManager)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

func acceptAndAssignControlStream(ym *service.YamuxStreamManager) error {
	yamuxStream, err := ym.Session.AcceptStream()
	if err != nil {
		return tracerr.Wrap(err)
	}
	logrus.Debug("accepted control stream for this yamux session")

	ym.ControlStream = service.NewControlStream(yamuxStream)

	helloResponder := func(hello mode.HelloRequest) (mode.HelloResponse, error) {
		return mode.HelloResponse{}, nil
	}
	if err := service.RpcExpectAndRespond[mode.HelloExchange](ym.ControlStream, helloResponder); err != nil {
		logrus.Errorf("failed to receive Hello or respond to it on the control stream: %s", err.Error())
		return err
	}

	return nil
}
