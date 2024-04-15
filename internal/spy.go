package internal

import (
	"context"
	"errors"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	io2 "github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/nimatrueway/unbound-ssh/internal/mode/spy"
	"github.com/nimatrueway/unbound-ssh/internal/service"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	"io"
	"os"
	"time"
)

func Spy() error {
	ctx := context.Background()

	// run ssh server
	serviceMan, err := service.NewSpyServiceManager(config.Config.Service)
	if err != nil {
		return err
	}
	defer func() {
		err := serviceMan.Close()
		if err != nil {
			logrus.Errorf("error closing service manager: %s", err.Error())
		}
	}()

	baseState, err := spy.NewBaseState()
	defer baseState.RawSwitch.Restore()
	if err != nil {
		return tracerr.Wrap(err)
	}

	// send hello message
	fmt.Print(signature.GenerateSpyStart(signature.NewSpyStart()))
	logrus.Info("Sent hello message to listener-mode")

	// read first line and expect hello back message
	success, err := expectHandshakeResponse(baseState.Stdin.BindToConcrete(ctx))
	if !success && err == nil {
		return nil
	}
	if err != nil {
		return err
	}

	// to yamux mode
	yamuxMode := spy.NewConnectedState(baseState.Stdin, os.Stdout)
	err = yamuxMode.ListenAndServe(ctx, serviceMan)
	if err != nil {
		return tracerr.Wrap(err)
	}

	return nil
}

func expectHandshakeResponse(stdin *io2.ContextBoundReader) (success bool, err error) {
	buffer := make([]byte, len(signature.ListenConnectFmt))
	alreadyRead := false

	// interrupt the read after 3 seconds
	go func() {
		<-time.After(config.Config.Transfer.ConnectionTimeout)
		if !alreadyRead {
			logrus.Warn("Did not receive hello back message from listen-mode. interrupting the read.")
			stdin.Cancel()
		}
	}()

	n, err := io.ReadAtLeast(stdin, buffer, len(signature.ListenConnectFmt))
	alreadyRead = true
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logrus.Errorf("Did not receive hello back message from listen-mode. exit.")
			fmt.Printf("\r\nDid not receive hello back message from listen-mode. exiting...\r\n")
			return false, nil
		} else {
			logrus.Errorf("error reading from stdin: %s", err.Error())
			return false, tracerr.Wrap(err)
		}
	}

	readLine := string(buffer[:n])
	if readLine != signature.ListenConnectFmt {
		logrus.Errorf("expected %v message from listen-mode, got %v", signature.ListenConnectFmt, readLine)
		return false, tracerr.Errorf("expected a hello back message from listen-mode, got %s", readLine)
	}

	logrus.Info("Received hello back message from listen-mode")
	return true, nil
}
