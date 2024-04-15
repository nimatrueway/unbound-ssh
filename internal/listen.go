package internal

import (
	"context"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/nimatrueway/unbound-ssh/internal/mode/listen"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

func Listen(cmd []string) error {
	ctx := context.Background()
	// launch the base mode
	baseState, err := listen.CreateBaseState(cmd)
	defer baseState.Close()
	if err != nil {
		return tracerr.Wrap(err)
	}

	for {
		// to signature detect mode
		signatureSearchMode := listen.CreateWiretapState(baseState)

		// run the signature search mode, continue only if SignatureFound error is returned
		stdinSigs := []signature.Signature{&signature.Preflight{}}
		ptyStdoutSigs := []signature.Signature{&signature.SpyStart{}}
		found, err := signatureSearchMode.TransferUntilFound(ctx, stdinSigs, ptyStdoutSigs)
		if err != nil {
			return err
		}

		if _, ok := found.(*signature.SpyStart); ok {
			logrus.Info("spy hello signature detected, transitioned to pre-yamux mode.")
			// switch to pre-yamux mode for handshaking and connecting
			preYamuxMode := listen.NewConnectingState(baseState)
			err := preYamuxMode.Connect(ctx)
			if err != nil {
				logrus.Warnf("yamux mode failed going back to transfer mode: %s", err.Error())
			}
		} else if _, ok := found.(*signature.Preflight); ok {
			logrus.Info("preflight signature detected, transitioned to preflight mode.")
			preflightMode := listen.NewPreflightState(baseState)
			err := preflightMode.Run(ctx)
			if err != nil {
				logrus.Warnf("preflight mode failed going back to transfer mode: %s", err.Error())
			}
		} else {
			return nil
		}
	}
}
