package spy

import (
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	term2 "github.com/nimatrueway/unbound-ssh/internal/io/term"
	"os"
)

type BaseState struct {
	Stdin     *core.ContextReader
	RawSwitch *term2.RawSwitch
}

func NewBaseState() (BaseState, error) {
	// create raw switch to switch between raw and cooked tty mode
	rawSwitch, err := term2.NewRawSwitch()
	if err != nil {
		return BaseState{}, err
	}

	baseState := BaseState{
		Stdin:     core.NewContextReader(os.Stdin),
		RawSwitch: rawSwitch,
	}

	return baseState, nil
}
