package utils

import (
	"bufio"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func NewTappedConnectionPair(t *testing.T, dir string) (server io.ReadWriteCloser, client io.ReadWriteCloser) {
	socket := NewSocket(t, "pair", dir)
	group := errgroup.Group{}
	group.Go(func() error {
		var err error
		listener, err := net.Listen("unix", socket)
		require.NoError(t, err)

		server, err = listener.Accept()
		require.NoError(t, err)

		return nil
	})
	group.Go(func() error {
		var err error
		client, err = KeepTrying(func() (net.Conn, error) {
			return net.Dial("unix", socket)
		})
		require.NoError(t, err)
		return nil
	})
	err := group.Wait()
	require.NoError(t, err)
	return TraceConnection("server", server), TraceConnection("client", client)
}

func TraceConnection(prefix string, rwc io.ReadWriteCloser) io.ReadWriteCloser {
	reader := core.NewReaderLogInterceptor(rwc)
	writer := core.NewWriterLogInterceptor(rwc)
	reader.Prefix = prefix + " "
	writer.Prefix = prefix + " "
	return core.WithRwCloser(core.NewReadWriter(reader, writer), func() error {
		return rwc.Close()
	})
}

func FindFreePort() (port int) {
	for {
		port = rand.Intn(64000) + 1000
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			_ = conn.Close()
		} else {
			break
		}
	}
	return port
}

func NewSocket(t *testing.T, name string, dir string) string {
	// https://unix.stackexchange.com/a/367012/226193
	// There is a 104 or 108 char limitation exists on the socket path
	const MaxSocketPathLength = 103

	if dir == "" {
		dir = os.TempDir()
	}
	testName := regexp.MustCompile("[A-Z][a-z]+").ReplaceAllStringFunc(t.Name(), func(s string) string {
		return "_" + strings.ToLower(s)
	})

retry:
	finalName := fmt.Sprintf("%x_%s_%s.sock", os.Getpid(), testName, name)

	socketPath := filepath.Join(dir, finalName)
	if len(socketPath) >= MaxSocketPathLength {
		diff := len(socketPath) - MaxSocketPathLength
		if len(testName) > diff {
			testName = testName[:len(testName)-diff]
			goto retry
		} else {
			require.FailNow(t, "socket name is too long: ", socketPath)
		}
	}

	_ = os.Remove(socketPath)
	return socketPath
}
func CopyFile(t *testing.T, src string, dst string) {
	srcF, err := os.OpenFile(src, os.O_RDONLY, 0644)
	require.NoError(t, err)

	dstF, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(t, err)

	_, err = io.Copy(dstF, srcF)
	require.NoError(t, err)

	err = srcF.Close()
	require.NoError(t, err)

	err = dstF.Close()
	require.NoError(t, err)
}

var RootDirCache string

func RootDir() string {
	if RootDirCache != "" {
		return RootDirCache
	}

	dir, _ := filepath.Abs("")

	// Look for enclosing go.mod.
	for {
		if fi, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !fi.IsDir() {
			return dir
		}
		d := filepath.Dir(dir)
		if d == dir {
			break
		}
		dir = d
	}
	RootDirCache = dir

	return ""
}

func TemporaryFile(t *testing.T, name string, content string) *os.File {
	testFile, err := os.CreateTemp(TestWorkspaceDir(t), name)
	require.NoError(t, err)
	_, err = testFile.WriteString(content)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, testFile.Close())
		require.NoError(t, os.Remove(testFile.Name()))
	})
	return testFile
}

func RunCmd(t *testing.T, cmd *exec.Cmd) {
	cmd.Env = lo.Filter(os.Environ(), func(item string, index int) bool {
		return !strings.HasPrefix(item, "CGO")
	})
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	go func() {
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadSlice('\n')
			if err != nil {
				break
			}
			t.Log(fmt.Sprintf("[process: out] %s", strings.TrimSpace(string(line))))
		}
	}()
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)
	go func() {
		reader := bufio.NewReader(stderr)
		for {
			line, err := reader.ReadSlice('\n')
			if err != nil {
				break
			}
			t.Log(fmt.Sprintf("[process: err] %s", strings.TrimSpace(string(line))))
		}
	}()
	err = cmd.Run()
	require.NoError(t, err)
}
