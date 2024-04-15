package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"io"
)

// DuplexCopy simultaneously transfers stdin->targetStdin and targetStdout->stdout
func DuplexCopy(context context.Context, targetStdin io.Writer, ctxStdin ContextBindingReader, stdout io.Writer, ctxTargetStdout ContextBindingReader) error {
	wg, ctx := errgroup.WithContext(context)

	// make both inputs cancellable
	var stdin io.Reader = ctxStdin.BindTo(ctx)
	var targetStdout io.Reader = ctxTargetStdout.BindTo(ctx)

	// fully log byte transfers if trace-level logging is enabled
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		targetStdout = NewReaderLogInterceptor(targetStdout)
		stdout = NewWriterLogInterceptor(stdout)
		stdin = NewReaderLogInterceptor(stdin)
		targetStdin = NewWriterLogInterceptor(targetStdin)
	}

	wg.Go(transfer(targetStdin, stdin))
	wg.Go(transfer(stdout, targetStdout))

	err := wg.Wait()
	if err == io.EOF {
		return nil
	}
	return err
}

func transfer(dst io.Writer, src io.Reader) func() error {
	return func() error {
		log := fmt.Sprintf("transferring from \"%s\" to \"%s\"", DetermineReaderName(src), DetermineWriterName(dst))
		logrus.Debug(log)
		var buffer []byte
		if config.Config.Transfer.Buffer > 0 {
			buffer = make([]byte, config.Config.Transfer.Buffer)
		}
		_, err := io.CopyBuffer(dst, src, buffer)
		if err != nil && !errors.Is(err, SignatureFound) {
			logrus.Warnf("finished %s with error: %s", log, err.Error())
		} else {
			logrus.Debugf("finished %s", log)
			err = io.EOF
		}
		return err
	}
}
