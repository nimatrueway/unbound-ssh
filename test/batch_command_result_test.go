package test

import (
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/nimatrueway/unbound-ssh/test/utils"
	"github.com/stretchr/testify/require"
	"os/exec"
	"strings"
	"testing"
)

func TestBatchCommandResultInDocker(t *testing.T) {
	containerId := "unbound_ssh_batch_command_exec_" + strings.ReplaceAll(utils.UUID7(t).String(), "-", "")
	shell := fmt.Sprintf("docker run --name %s --env PS1=$(ShellPrompt) --platform linux/amd64 --rm --interactive --tty alpine", containerId)
	c, _ := utils.LaunchAndConnect(t, utils.UnboundSshLaunchConfig{
		ListenShellCmd: shell,
	})
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", containerId).Run()
	})
	commands := map[string]string{
		"multi_line":       "cat /etc/fstab",
		"success":          "command -v dd",
		"failure":          "cat Xtty",
		"failure_piped":    "cat Xtty | head -n 1",
		"not_exists":       "command -v Xstty",
		"not_exists_piped": "/usr/bin/sw_vers | head -n 1",
	}

	cmd, parsed := signature.GenerateNamedCommandsAndCaptureResult(commands)
	c.MustSend(cmd)

	_ = c.MustExpectSignature(parsed)

	require.Equal(t, 0, parsed.Get("multi_line").Result)
	require.Equal(t, `/dev/cdrom	/media/cdrom	iso9660	noauto,ro 0 0
/dev/usbdisk	/media/usb	vfat	noauto,ro 0 0`, parsed.Get("multi_line").Output)

	require.Equal(t, 0, parsed.Get("success").Result)
	require.Equal(t, "/bin/dd", parsed.Get("success").Output)

	require.Equal(t, 1, parsed.Get("failure").Result)
	require.Equal(t, "", parsed.Get("failure").Output)

	require.Equal(t, 1, parsed.Get("failure_piped").Result)
	require.Equal(t, "", parsed.Get("failure_piped").Output)

	require.Equal(t, 127, parsed.Get("not_exists").Result)
	require.Equal(t, "", parsed.Get("not_exists").Output)

	require.Equal(t, 127, parsed.Get("not_exists_piped").Result)
	require.Equal(t, "", parsed.Get("not_exists_piped").Output)
}
