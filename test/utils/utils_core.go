package utils

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func UUID7(t *testing.T) uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		require.NoError(t, err)
	}
	return id
}

func KeepTrying[T any](f func() (T, error)) (T, error) {
	var res T
	var err error
	elapsed := time.Duration(0)
	for {
		res, err = f()
		if err == nil || elapsed > AssertionTimeout {
			break
		} else {
			time.Sleep(25 * time.Millisecond)
			elapsed += 25 * time.Millisecond
		}
	}
	return res, err
}

var testWorkspace = map[*testing.T]string{}

func TestWorkspaceDir(t *testing.T) string {
	if testWorkspace[t] == "" {
		// normal temporary directory does not work
		// as docker on macOS can not mount the directory

		var err error
		trashDir := filepath.Join(RootDir(), ".trash")
		if _, err := os.Stat(trashDir); err != nil {
			_ = os.MkdirAll(trashDir, 0755)
		}
		testWorkspace[t], err = os.MkdirTemp(trashDir, "unbound-ssh-test-*")
		require.NoError(t, err)

		t.Cleanup(func() {
			_ = os.RemoveAll(testWorkspace[t])
		})
	}
	return testWorkspace[t]
}

var ExecutablesCached = false

// EnsureFreshBinariesInTestWorkspace ensures that the test workspace has the latest binaries in it
func EnsureFreshBinariesInTestWorkspace(t *testing.T) {
	// produce executables
	if ExecutablesCached {
		cmd := exec.Command("./scripts/build.sh")
		cmd.Dir = RootDir()
		RunCmd(t, cmd)
		ExecutablesCached = true
	}

	// copy them to the temporary test workspace
	for bins, err := filepath.Glob(filepath.Join(RootDir(), "./output/unbound-ssh_*")); err == nil && len(bins) > 0; bins = bins[1:] {
		src := bins[0]
		dst := filepath.Join(TestWorkspaceDir(t), filepath.Base(src))
		CopyFile(t, src, dst)
		require.NoError(t, err)
		err := os.Chmod(dst, 0755)
		require.NoError(t, err)
	}
}

func BinaryOsArch(t *testing.T) string {
	var operatingSystem string
	if runtime.GOOS == "darwin" {
		operatingSystem = "darwin"
	} else if runtime.GOOS == "linux" || runtime.GOOS == "alpine" {
		operatingSystem = "linux"
	} else {
		require.FailNow(t, "unsupported OS: ", runtime.GOOS)
	}
	var architecture string
	if runtime.GOARCH == "amd64" {
		architecture = "amd64"
	} else if runtime.GOARCH == "arm64" {
		architecture = "arm64"
	} else {
		require.FailNow(t, "unsupported ARCH: ", runtime.GOARCH)
	}

	return operatingSystem + "_" + architecture
}
