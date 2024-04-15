package term

import (
	ptylib "github.com/creack/pty"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

type SizeAdaptor struct {
	ch chan os.Signal
}

func NewSizeAdaptor(pty *os.File) (*SizeAdaptor, error) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			size, err := ptylib.GetsizeFull(os.Stdin)
			if err != nil {
				logrus.Warnf("error fetching stdin size: %s", err)
				return
			}

			err = ptylib.Setsize(pty, size)
			if err != nil {
				logrus.Warnf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH
	return &SizeAdaptor{ch}, nil
}

func (sa *SizeAdaptor) Close() {
	signal.Stop(sa.ch)
}
