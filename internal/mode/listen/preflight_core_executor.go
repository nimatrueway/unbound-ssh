package listen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	io2 "github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/sirupsen/logrus"
	stdio "io"
	"os"
	"time"
)

type ShellExecutor struct {
	PtyReader      *io2.ContextReader
	PtyWriter      stdio.Writer
	DefaultTimeout time.Duration
}

func NewShellExecutor(ptyReader *io2.ContextReader, ptyWriter stdio.Writer) *ShellExecutor {
	return &ShellExecutor{PtyReader: ptyReader, PtyWriter: ptyWriter, DefaultTimeout: config.Config.Preflight.CommandTimeout}
}

func (se *ShellExecutor) Execute(ctx context.Context, cmd string, inputData []byte) (signature.CommandResult, error) {
	collector, err := Execute[*signature.BatchCommandResult](ctx, se, inputData)(signature.GenerateCommandsAndCaptureResult([]string{cmd}))
	if err != nil {
		return signature.NullCommandResult, err
	}

	if len(collector.Results) != 1 {
		return signature.NullCommandResult, fmt.Errorf("expected 1 command result received %d: %s", len(collector.Results), collector.Captured)
	}

	result := collector.Results[0]
	if !result.IsSuccess() {
		return signature.NullCommandResult, fmt.Errorf("command '%s' failed with exit code %d: %s", cmd, result.Result, result.Output)
	}

	return result, nil
}

// Execute has a pretty weird signature due to golang type inference limitations, it is meant to be used in this way:
// Execute[*signature.XXX](ctx, shell, inputData)(signature.GenerateXXX())
func Execute[S signature.Signature](ctx context.Context, se *ShellExecutor, input []byte) func(string, S) (S, error) {
	return func(cmd string, sig S) (S, error) {
		err := se.runAndExpect(ctx, cmd, input, sig)
		return sig, err
	}
}

func (se *ShellExecutor) runAndExpect(ctx context.Context, command string, inputData []byte, expect signature.Signature) error {
	robotR, robotW := stdio.Pipe()
	go func() {
		n, err := robotW.Write([]byte(command))
		if err != nil {
			logrus.Warnf("failed to write command to pty: %s", err.Error())
		}
		if n != len(command) {
			logrus.Warnf("failed to write full command to pty: %d/%d", n, len(command))
		}
		if inputData != nil && len(inputData) > 0 {
			_, err = stdio.Copy(robotW, bytes.NewBuffer(inputData))
			if err != nil {
				logrus.Warnf("failed to write input data to pty: %s", err.Error())
			}
		}
	}()

	ctxWithTimeout, _ := context.WithTimeout(ctx, se.DefaultTimeout)
	err := io2.DuplexCopy(ctxWithTimeout, se.PtyWriter, &io2.ContextReadCloser{ReadCloser: robotR}, os.Stdout, io2.NewSignatureDetector(expect).Wrap(se.PtyReader))
	if err != nil && !errors.Is(err, io2.SignatureFound) {
		return err
	}

	return nil
}
