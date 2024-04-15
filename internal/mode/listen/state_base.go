package listen

import (
	"errors"
	creackpty "github.com/creack/pty"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/internal/io/term"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	"os"
	"os/exec"
	"sync/atomic"
)

type BaseState struct {
	Process     *exec.Cmd
	Pty         *os.File
	Stdin       *core.ContextReader
	PtyStdout   *core.ContextReader
	RawSwitch   *term.RawSwitch
	SizeAdaptor *term.SizeAdaptor
	isClosed    atomic.Bool
}

func CreateBaseState(cmd []string) (*BaseState, error) {
	// create slave process and capture its tty in a pty
	logrus.Info("shell to execute: ", cmd)
	proc := exec.Command(cmd[0], cmd[1:]...)
	pty, err := creackpty.Start(proc)
	logrus.Info("slave process started with pid: ", proc.Process.Pid)
	if err != nil {
		return nil, err
	}

	// create raw switch to switch between raw and cooked mode
	rawSwitch, err := term.NewRawSwitch()
	if err != nil {
		return nil, err
	}

	// create size adaptor to adapt the size of the pty
	sizeAdaptor, err := term.NewSizeAdaptor(pty)
	if err != nil {
		return nil, err
	}

	baseState := BaseState{
		Process:     proc,
		Pty:         pty,
		Stdin:       core.NewContextReader(os.Stdin),
		PtyStdout:   core.NewContextReader(pty),
		RawSwitch:   rawSwitch,
		SizeAdaptor: sizeAdaptor,
	}

	baseState.interruptInputsOnExit()

	return &baseState, nil
}

// Check if the process is exited and return the error if it is exited with non-zero exit code
func (bm *BaseState) isProcessExited() (bool, error) {
	processState := bm.Process.ProcessState
	if processState != nil && processState.Exited() {
		exitCode := processState.ExitCode()
		if exitCode == 0 {
			logrus.Debug("slave process exited with no error.")
			return true, nil
		} else {
			logrus.Warn("slave process exited with error code: ", exitCode)
			return true, tracerr.Errorf("process exited with code %d", exitCode)
		}
	} else {
		return false, nil
	}
}

// Interrupt the readers when the process is exited to gracefully shut down the duplex transfer
func (bm *BaseState) interruptInputsOnExit() {
	go func() {
		// wait for process to exit to be able to read its exit code
		if err := bm.Process.Wait(); err != nil {
			logrus.Warn("slave process did not exit gracefully: ", err)
		} else {
			logrus.Debug("slave process exited with code: ", bm.Process.ProcessState.ExitCode())
		}

		// interrupt the all blocked readers
		bm.Close()
	}()
}

func (bm *BaseState) Close() {
	if bm.isClosed.Load() {
		return
	}
	bm.isClosed.Store(true)

	bm.RawSwitch.Restore()
	bm.SizeAdaptor.Close()

	// EOF makes more sense than Cancel
	err := os.Stdin.Close()
	if err != nil {
		if !errors.Is(err, os.ErrClosed) {
			logrus.Warn("error while closing stdin: ", err)
		}
	} else {
		logrus.Debug("stdin closed.")
	}

	// EOF makes more sense than Cancel
	err = bm.Pty.Close()
	if err != nil {
		if !errors.Is(err, os.ErrClosed) {
			logrus.Warn("error while closing slave process pty: ", err)
		}
	} else {
		logrus.Debug("slave process pty closed.")
	}
}
