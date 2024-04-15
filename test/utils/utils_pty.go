package utils

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/stretchr/testify/require"
	"io"
	"os/exec"
	"regexp"
	"testing"
	"time"
)

// ShellPrompt will be set as the prompt of the launched shell
const ShellPrompt = "shell#>"

// AssertionTimeout is the maximum time to wait for an assertion to complete
const AssertionTimeout = 3 * time.Second

// --------------------------------------------------------------------------------------------

type Console struct {
	CtxReader  *core.ContextReader
	Writer     io.Writer
	cmd        *exec.Cmd
	t          *testing.T
	onExitCode chan int
}

func (c *Console) MustExpectEOF() string {
	output, err := c.expect(nil)
	require.Equal(c.t, io.EOF, err)
	return output
}

func (c *Console) MustExpect(s string) string {
	sig := signature.NewRegexSignature(regexp.MustCompile(regexp.QuoteMeta(s)))
	output, err := c.expect(sig)
	require.NoError(c.t, err)
	return output
}

func (c *Console) MustExpectRegex(r *regexp.Regexp) string {
	sig := signature.NewRegexSignature(r)
	output, err := c.expect(sig)
	require.NoError(c.t, err)
	return output
}

func (c *Console) MustExpectSignature(sig signature.Signature) string {
	output, err := c.expect(sig)
	require.NoError(c.t, err)
	return output
}

func SendAndMustExpect[S signature.Signature](c *Console) func(string, S) S {
	return func(cmd string, sig S) S {
		c.MustSend(cmd)
		_ = c.MustExpectSignature(sig)
		return sig
	}
}

func (c *Console) expect(sig signature.Signature) (string, error) {
	ctx, _ := context.WithTimeout(context.Background(), AssertionTimeout)
	var ctxSigReader io.Reader
	if sig != nil {
		sigReader := core.NewSignatureDetector(sig).Wrap(c.CtxReader)
		ctxSigReader = sigReader.BindTo(ctx)
	} else {
		ctxSigReader = c.CtxReader.BindTo(ctx)
	}
	bufCtxSigReader := bufio.NewReader(ctxSigReader)
	consumedBuffer := bytes.NewBuffer([]byte{})

	defer func() {
		toUnread := bufCtxSigReader.Buffered()
		if toUnread > 0 {
			buf := make([]byte, toUnread)
			n, err := bufCtxSigReader.Read(buf)
			require.NoError(c.t, err)
			require.Equal(c.t, toUnread, n)
			c.CtxReader.UnreadBytes(buf[:n])
		}
	}()

	for {
		line, err := bufCtxSigReader.ReadString('\n')

		if line != "" {
			consumedBuffer.Write([]byte(line))
			c.t.Log(fmt.Sprintf("%#v", line))
		}
		if errors.Is(err, core.SignatureFound) {
			return consumedBuffer.String(), nil
		} else if err != nil {
			return consumedBuffer.String(), err
		}
	}
}

func (c *Console) MustSend(s string) {
	_, err := c.Writer.Write([]byte(s))
	require.NoError(c.t, err)
}

func (c *Console) MustExit(code int) {
	select {
	case exitCode := <-c.onExitCode:
		require.Equal(c.t, code, exitCode)
	case <-time.After(AssertionTimeout):
		_ = c.cmd.Process.Kill()
		c.t.Fatalf("timed out waiting for process to exit")
	}
}
