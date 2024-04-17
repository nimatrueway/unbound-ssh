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
	// create base state
	baseState, err := listen.CreateBaseState(cmd)
	defer baseState.Close()
	if err != nil {
		return tracerr.Wrap(err)
	}

	// transition to wiretap state
	wiretapState := listen.CreateWiretapState(baseState)

	for {
		// run wiretap state, continue only if SignatureFound error is returned
		stdinSigs := []signature.Signature{&signature.Preflight{}}
		ptyStdoutSigs := []signature.Signature{&signature.SpyStart{}}

		found, err := wiretapState.TransferUntilFound(ctx, stdinSigs, ptyStdoutSigs)
		if err != nil {
			return err
		}

		if _, ok := found.(*signature.SpyStart); ok {
			logrus.Info("spy hello signature detected, transitioned to connecting state.")
			// transition to connecting state for handshake
			connectingState := listen.NewConnectingState(baseState)
			err := connectingState.Connect(ctx)
			if err != nil {
				logrus.Warnf("connecting/connected state failed, transitioning back to wiretap state: %s", err.Error())
			}
		} else if _, ok := found.(*signature.Preflight); ok {
			logrus.Info("preflight signature detected, transitioning to preflight state.")
			preflightState := listen.NewPreflightState(baseState)
			err := preflightState.Run(ctx)
			if err != nil {
				logrus.Warnf("preflight state failed transitioning back to wiretap state: %s", err.Error())
			}
		} else {
			return nil
		}
	}
}
