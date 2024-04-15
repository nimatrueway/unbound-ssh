package test

import (
	"context"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/mode/listen"
	"github.com/nimatrueway/unbound-ssh/test/utils"
	"github.com/stretchr/testify/require"
	"os/exec"
	"strings"
	"testing"
)

const UploadSize = 1 * 1024 * 1024

func TestFileUploadAlpineBase64(t *testing.T) {
	testUpload(t, generateContainerizedShell(t), listen.Base64)
}

func TestFileUploadAlpineGzipBase64(t *testing.T) {
	testUpload(t, generateContainerizedShell(t), listen.GzipBase64)
}

func TestFileUploadAlpineAscii85(t *testing.T) {
	testUpload(t, generateContainerizedShell(t), listen.Ascii85)
}

func TestFileUploadAlpineGzipAscii85(t *testing.T) {
	testUpload(t, generateContainerizedShell(t), listen.GzipAscii85)
}

func generateContainerizedShell(t *testing.T) string {
	containerId := "unbound_ssh_preflight_upload_" + strings.ReplaceAll(utils.UUID7(t).String(), "-", "")
	shellCmd := fmt.Sprintf("docker run --name %s --workdir /home --env PS1=$(ShellPrompt) --platform linux/amd64 --rm --interactive --tty python:alpine /bin/sh", containerId)
	// make sure docker container is removed
	t.Cleanup(func() {
		if strings.Contains(shellCmd, "docker") {
			_ = exec.Command("docker", "rm", "-f", containerId).Run()
		}
	})
	return shellCmd
}

func testUpload(t *testing.T, shellCmd string, transferEnc listen.Encoding) {
	ctx := context.Background()
	c, _ := utils.LaunchAndConnect(t, utils.UnboundSshLaunchConfig{
		LaunchConfig:   utils.LaunchConfig{WorkDir: "$(TestWorkspaceDir)"},
		ListenShellCmd: shellCmd,
	})

	// probe *nix capabilities, uploader relies on it
	listen.GlobalProbeResult = utils.SendAndMustExpect[*listen.NixProbeResult](c)(listen.GeneratePrintNixProbe())

	// dummy data to verify later
	buf := make([]byte, UploadSize)
	for i := 0; i < len(buf); i++ {
		buf[i] = byte(i % 256)
	}

	filename := "test.bin"
	shell := listen.ShellExecutor{
		PtyReader:      c.CtxReader,
		PtyWriter:      c.Writer,
		DefaultTimeout: utils.AssertionTimeout,
	}
	err := listen.Upload(ctx, &shell, string(buf), filename, transferEnc)
	require.NoError(t, err)

	c.MustExpect(utils.ShellPrompt)
	c.MustSend("\004") // ctrl+d
	c.MustExpectEOF()
	c.MustExit(0)
}
