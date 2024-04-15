package test

import (
	"fmt"
	"github.com/Netflix/go-expect"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/test/utils"
	"github.com/stretchr/testify/require"
	"net"
	"strings"
	"testing"
)

func TestConnection(t *testing.T) {
	c, _ := utils.LaunchAndConnect(t, utils.UnboundSshLaunchConfig{
		LaunchConfig:   utils.LaunchConfig{WorkDir: "$(TestWorkspaceDir)"},
		ListenShellCmd: "go run $(RootDir)/cmd/cli.go listen sh",
		SpyLaunchCmd:   "go run $(RootDir)/cmd/cli.go spy",
	})
	verifyGracefulExit(t, c)
}

func TestEchoLocal(t *testing.T) {
	utils.EnsureFreshBinariesInTestWorkspace(t)
	c, vars := utils.LaunchAndConnect(t, utils.LocalIsolated)
	serviceAddr := config.NewAddress("unix", vars["RandomSocket0"])
	verifyEchoService(t, &serviceAddr)
	verifyGracefulExit(t, c)
}

func TestEchoLocalExec(t *testing.T) {
	utils.EnsureFreshBinariesInTestWorkspace(t)
	c, vars := utils.LaunchAndConnect(t, utils.LocalExecToLocalExec)
	serviceAddr := config.NewAddress("unix", vars["RandomSocket0"])
	verifyEchoService(t, &serviceAddr)
	verifyGracefulExit(t, c)
}

func TestEchoLocalToAlpine(t *testing.T) {
	utils.EnsureFreshBinariesInTestWorkspace(t)
	c, vars := utils.LaunchAndConnect(t, utils.LocalExecToAlpine)
	serviceAddr := config.NewAddress("unix", vars["RandomSocket0"])
	verifyEchoService(t, &serviceAddr)
	verifyGracefulExit(t, c)
}

func TestEchoAlpineToAlpine(t *testing.T) {
	utils.EnsureFreshBinariesInTestWorkspace(t)
	c, vars := utils.LaunchAndConnect(t, utils.AlpineToAlpine)
	serviceAddr := config.NewAddress("tcp", fmt.Sprintf("0.0.0.0:%s", vars["RandomPort0"]))
	verifyEchoService(t, &serviceAddr)
	verifyGracefulExit(t, c)
}

func verifyEchoService(t *testing.T, addr net.Addr) {
	conn, err := utils.KeepTrying(func() (net.Conn, error) {
		return net.Dial(addr.Network(), addr.String())
	})
	require.NoError(t, err)

	console, _ := expect.NewTestConsole(t, expect.WithStdin(conn), expect.WithStdout(conn), expect.WithDefaultTimeout(utils.AssertionTimeout))
	_, err = console.SendLine("hello")
	require.NoError(t, err)
	_, err = console.ExpectString("received: hello")
	require.NoError(t, err)
}

func verifyGracefulExit(t *testing.T, c *utils.Console) {
	// Terminate them using Ctrl+C
	c.MustSend("\003")

	// Test the prompt showing back up
	output := c.MustExpect(utils.ShellPrompt)
	require.Equal(t, utils.ShellPrompt, strings.TrimSpace(output))

	// Terminate the shell
	c.MustSend("exit\n")

	c.MustExpectEOF()
	c.MustExit(0)
}
