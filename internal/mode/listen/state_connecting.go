package listen

import (
	"context"
	"errors"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/nimatrueway/unbound-ssh/internal/io/term"
	"github.com/nimatrueway/unbound-ssh/internal/service"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

type ConnectingState struct {
	baseState *BaseState
}

func NewConnectingState(baseState *BaseState) ConnectingState {
	return ConnectingState{baseState: baseState}
}

func (pym *ConnectingState) Connect(ctx context.Context) error {
	// complete the handshake
	_, err := fmt.Fprintf(pym.baseState.Pty, signature.ListenConnectFmt)
	if err != nil {
		fmt.Print("\r\nfailed to connect.\r\n")
		return tracerr.Wrap(err)
	} else {
		fmt.Print(signature.ListenConnectFmt)
	}
	logrus.Info("sent hello back to complete handshake.")

	// launch the listener
	serviceManager, err := service.NewListenServiceManager(config.Config.Service)
	if err != nil {
		return err
	}

	// create connected state
	connectedState := CreateConnectedState(pym.baseState.PtyStdout, pym.baseState.Pty)

	// exchange context
	connectedStateCtx, connectedStateCloser := context.WithCancel(ctx)

	// allow user to kill the process with ctrl+c or ctrl+d
	reactToClose := term.NewReadInBackground(pym.baseState.Stdin.BindTo(connectedStateCtx))
	stop := func() {
		logrus.Info("received interrupt signal, shutting down connected state.")
		connectedStateCloser()
	}
	reactToClose.ReactTo(EndOfText, stop).Start(connectedStateCtx)
	defer connectedStateCloser()

	// transition to connected state
	err = connectedState.ListenAndServe(connectedStateCtx, serviceManager)

	if err != nil {
		logrus.Warnf("connected state failed: %s", err.Error())
		if errors.Is(err, context.Canceled) {
			return nil
		} else {
			return tracerr.Wrap(err)
		}
	}

	return nil
}
