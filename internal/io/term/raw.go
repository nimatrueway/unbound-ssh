package term

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/term"
	"os"
)

type RawSwitch struct {
	fd    int
	state *term.State
}

func NewRawSwitch() (*RawSwitch, error) {
	stdinFd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(stdinFd)
	if err != nil {
		return nil, err
	}
	logrus.Debug("switched to raw mode.")

	return &RawSwitch{fd: stdinFd, state: state}, nil
}

func (r *RawSwitch) Restore() {
	_ = term.Restore(int(os.Stdin.Fd()), r.state)

	logrus.Debug("switched to cooked mode.")
}
