package signature

import (
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
)

func TestCommandResult(t *testing.T) {
	actual, parsed := GenerateCommandAndCaptureResult("", "hello world")
	printCmd := ` { printf "________capture_begin_10000000200030004000500000000000________\nhello world\n_________capture_end_10000000200030004000500000000000_________\n"; }` + "\n"
	require.Equal(t, printCmd, regexp.MustCompile("_[0-9a-f]{32}_").ReplaceAllString(actual, "_10000000200030004000500000000000_"))

	output := `________capture_begin_10000000200030004000500000000000________
hello world
_________capture_end_10000000200030004000500000000000_________
`
	outputTty := strings.ReplaceAll(output, "\n", "\r\n") // to mimic tty output
	idx := parsed.Find(outputTty)
	require.NotEqual(t, -1, idx)

	require.Equal(t, "10000000-2000-3000-4000-500000000000", parsed.Id.String())
	require.Equal(t, "hello world", parsed.Captured)
}
