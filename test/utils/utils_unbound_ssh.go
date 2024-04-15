package utils

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/creack/pty"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/stretchr/testify/require"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Variables that can be used in LaunchConfig
//
// RandomPort0 		random free tcp port number
// RandomSocket0    random free socket address
// RootDir			root directory of the project
// TestWorkspaceDir an isolated temporary directory to launch test in
// BinaryExec 		binary executable name for current platform
// ShellPrompt		the shell prompt (utils.ShellPrompt) to be used in expect tests
type Variables = map[string]string

type LaunchConfig struct {
	WorkDir string
}

type UnboundSshLaunchConfig struct {
	LaunchConfig
	ListenShellCmd string
	SpyLaunchCmd   string
	AppConfig      string
}

var LocalIsolated = UnboundSshLaunchConfig{
	LaunchConfig:   LaunchConfig{WorkDir: "$(TestWorkspaceDir)"},
	ListenShellCmd: "go run $(RootDir)/cmd/cli.go listen sh",
	SpyLaunchCmd:   "go run $(RootDir)/cmd/cli.go spy",
	AppConfig: `
[[service]]
type = "echo"
bind = "unix://$(RandomSocket0)"
`,
}

var LocalExecToLocalExec = UnboundSshLaunchConfig{
	LaunchConfig:   LaunchConfig{WorkDir: "$(TestWorkspaceDir)"},
	ListenShellCmd: "./$(BinaryExec) listen sh",
	SpyLaunchCmd:   "./$(BinaryExec) spy",
	AppConfig: `
[[service]]
type = "echo"
bind = "unix://$(RandomSocket0)"
`,
}

var LocalExecToAlpine = UnboundSshLaunchConfig{
	LaunchConfig:   LaunchConfig{WorkDir: "$(TestWorkspaceDir)"},
	ListenShellCmd: "./$(BinaryExec) listen -- docker run --volume .:/home --workdir /home --env PS1=$(ShellPrompt) --platform linux/amd64 --rm --interactive --tty alpine",
	SpyLaunchCmd:   "./unbound-ssh_linux_amd64 spy",
	AppConfig: `
[[service]]
type = "echo"
bind = "unix://$(RandomSocket0)"
`,
}

var AlpineToAlpine = UnboundSshLaunchConfig{
	LaunchConfig:   LaunchConfig{WorkDir: "$(TestWorkspaceDir)"},
	ListenShellCmd: "docker run --volume .:/home --workdir /home --publish $(RandomPort0):$(RandomPort0) --env PS1=$(ShellPrompt) --platform linux/amd64 --rm --interactive --tty alpine ./unbound-ssh_linux_amd64 listen -- sh",
	SpyLaunchCmd:   "./unbound-ssh_linux_amd64 spy",
	AppConfig: `
[[service]]
type = "echo"
bind = "tcp://0.0.0.0:$(RandomPort0)"
`,
}

func LaunchAndConnect(t *testing.T, testConfig UnboundSshLaunchConfig) (*Console, Variables) {
	variables := Variables{}

	// put log file into the variables
	variables["RandomPort0"] = strconv.Itoa(FindFreePort())
	variables["RandomSocket0"] = NewSocket(t, "rnd0", "")
	variables["RootDir"] = RootDir()
	variables["TestWorkspaceDir"] = TestWorkspaceDir(t)
	variables["BinaryExec"] = fmt.Sprintf("unbound-ssh_%s", BinaryOsArch(t))
	variables["ShellPrompt"] = ShellPrompt

	if testConfig.SpyLaunchCmd != "" {
		createLogFilePair(t)
	}

	testConfig.WorkDir = substituteVars(testConfig.WorkDir, variables)

	// Write the config file
	configTrimmed := strings.TrimSpace(testConfig.AppConfig)
	configFile := path.Join(TestWorkspaceDir(t), "config.toml")
	err := os.WriteFile(configFile, []byte(substituteVars(configTrimmed, variables)), 0644)
	require.NoError(t, err)

	// Launch the listen-mode
	c := LaunchShell(t, substituteVars(testConfig.ListenShellCmd, variables), testConfig.LaunchConfig)
	// Test the prompt showing up
	c.MustExpect(ShellPrompt)

	if testConfig.SpyLaunchCmd != "" {
		// Launch the spy-mode
		c.MustSend(substituteVars(testConfig.SpyLaunchCmd, variables))
		c.MustSend("\n")

		// Test spy-mode initiating handshake by sending
		c.MustExpectRegex(signature.SpyStartRegex)

		// Test listen-mode completing handshake
		c.MustExpect(signature.ListenConnectFmt)
	}

	return c, variables
}

func LaunchShell(t *testing.T, command string, conf LaunchConfig) *Console {
	ptyFile, tty, err := pty.Open()
	require.NoError(t, err)

	args := strings.Split(command, " ")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	if conf.WorkDir != "" {
		cmd.Dir = conf.WorkDir
	} else {
		cmd.Dir = RootDir()
	}
	cmd.Env = append(os.Environ(), fmt.Sprintf("PS1=%s", ShellPrompt))

	err = cmd.Start()
	require.NoError(t, err)

	result := &Console{CtxReader: core.NewContextReader(ptyFile), Writer: ptyFile, cmd: cmd, t: t, onExitCode: make(chan int)}

	go func() {
		_ = cmd.Wait()
		err = tty.Close() // close the tty to signal EOF
		if core.IsAlreadyClosed(err) {
			err = nil
		}
		require.NoError(t, err)
		result.onExitCode <- cmd.ProcessState.ExitCode()
	}()

	t.Cleanup(func() {
		_ = ptyFile.Close()
	})

	return result
}

func relative(base string, path string) string {
	if base == "" {
		return path
	} else if strings.HasPrefix(path, base) {
		return path[len(base):]
	} else {
		return path
	}
}

func substituteVars(in string, vars map[string]string) string {
	for k, v := range vars {
		in = strings.ReplaceAll(in, fmt.Sprintf("$(%s)", k), v)
	}
	return in
}

// createLogFilePair creates two temporary log files for listen and spy modes and siphons those into the test output
// returned string is the path to the log file containing "$(mode)" placeholder which will be interpreted by unbound-ssh
// if used in log.file configuration
func createLogFilePair(t *testing.T) {
	for _, mod := range []string{"listen", "spy"} {
		go func(mod string) {
			mogLogFile := strings.Replace("unbound-ssh-$(mode).log", "$(mode)", mod, 1)
			var logFile *os.File

			// wait until file is created
			err := fs.ErrNotExist
			for wait := AssertionTimeout; wait > 0 && err != nil; wait -= 25 * time.Millisecond {
				logFile, err = os.Open(path.Join(TestWorkspaceDir(t), mogLogFile))
				time.Sleep(25 * time.Millisecond)
			}
			if err != nil {
				t.Logf("failed to open %s log: %s", mod, err.Error())
				return
			}

			t.Logf("merging %s logs into the test.", path.Base(mogLogFile))
			reader := bufio.NewReader(logFile)
			for {
				l, err := reader.ReadString('\n')
				if err != nil {
					if errors.Is(err, io.EOF) {
						time.Sleep(25 * time.Millisecond)
						continue
					}
					t.Logf("failed to read %s log: %s", mod, err.Error())
				}
				t.Logf("[%s]: %s", mod, l)
			}
		}(mod)
	}
}
