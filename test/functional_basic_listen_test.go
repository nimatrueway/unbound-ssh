package test

import (
	"fmt"
	"github.com/nimatrueway/unbound-ssh/test/utils"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestShell(t *testing.T) {
	c := utils.LaunchShell(t, fmt.Sprintf("go run ./cmd/cli.go listen -- sh"), utils.LaunchConfig{})

	// Test the prompt showing up
	c.MustExpect(">")

	// Test the echo
	c.MustSend("echo Hello World")
	c.MustExpect("echo Hello World")

	// send the command
	c.MustSend("\n")

	// send the command
	c.MustExpect("\r\n")
	c.MustExpect("Hello World\r\n")
	c.MustExpect(">")

	// test the exit
	c.MustSend("exit\n")
	c.MustExit(0)
}

func TestCat(t *testing.T) {
	fileContent := "Hello World One\nHello World Two"
	testFile := utils.TemporaryFile(t, "text_file", fileContent)

	c := utils.LaunchShell(t, fmt.Sprintf("go run ./cmd/cli.go listen -- cat %s", testFile.Name()), utils.LaunchConfig{})

	output := c.MustExpectEOF()
	consoleContent := "Hello World One\r\nHello World Two"
	require.Equal(t, consoleContent, output)

	c.MustExit(0)
}
