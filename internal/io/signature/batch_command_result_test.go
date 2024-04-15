package signature

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
)

func TestBatchCommandResult(t *testing.T) {
	actual, parsed := GenerateCommandsAndCaptureResult([]string{
		"command -v dd",
		"command -v stty",
	})
	printCmd := fmt.Sprintf(` { set -o pipefail; o_0=$(command -v dd | awk '{print}' ORS='\\\\n'); r_0=$?; o_1=$(command -v stty | awk '{print}' ORS='\\\\n'); r_1=$?; printf "________capture_begin_10000000200030004000500000000000________\nr_0=$r_0\no_0=$o_0\nr_1=$r_1\no_1=$o_1\n_________capture_end_10000000200030004000500000000000_________\n"; }` + "\n")
	require.Equal(t, printCmd, regexp.MustCompile("_[0-9a-f]{32}_").ReplaceAllString(actual, "_10000000200030004000500000000000_"))

	output := `________capture_begin_10000000200030004000500000000000________
r_0=0
o_0=/bin/dd\n
r_1=0
o_1=/bin/stty\n
_________capture_end_10000000200030004000500000000000_________
`
	outputTty := strings.ReplaceAll(output, "\n", "\r\n") // to mimic tty output
	idx := parsed.Find(outputTty)
	require.NotEqual(t, -1, idx)

	require.Equal(t, CommandResult{Result: 0, Output: "/bin/dd"}, parsed.Results[0])
	require.Equal(t, CommandResult{Result: 0, Output: "/bin/stty"}, parsed.Results[1])
}
