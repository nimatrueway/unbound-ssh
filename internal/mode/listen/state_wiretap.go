package listen

import (
	"context"
	"errors"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/sirupsen/logrus"
	"os"
)

type WiretapState struct {
	*BaseState
}

func CreateWiretapState(baseState *BaseState) WiretapState {
	return WiretapState{BaseState: baseState}
}

func (bm *WiretapState) TransferUntilFound(ctx context.Context, localSigs []signature.Signature, remoteSigs []signature.Signature) (sig signature.Signature, err error) {
	logrus.Debug("transitioned to wiretap state")

	// start the full duplex transfer
	stdinTapped := core.NewSignatureDetector(localSigs...)
	remoteStdoutTapped := core.NewSignatureDetector(remoteSigs...)
	err = core.DuplexCopy(ctx, bm.Pty, stdinTapped.Wrap(bm.Stdin), os.Stdout, remoteStdoutTapped.Wrap(bm.PtyStdout))
	if errors.Is(err, core.SignatureFound) {
		err = nil
	}

	sig = remoteStdoutTapped.LastMatch()
	if sig == nil {
		sig = stdinTapped.LastMatch()
	}

	return sig, err
}
