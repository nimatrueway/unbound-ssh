package listen

import (
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"golang.org/x/mod/semver"
	"strings"
)

type NixProbeResult struct {
	*signature.BatchNamedCommandResult
}

var GlobalProbeResult *NixProbeResult

func GeneratePrintNixProbe() (command string, collector *NixProbeResult) {
	cmd := map[string]string{
		"internet":         "nc -w 1 -z github.com 80",
		"dd":               "command -v dd",
		"stty":             "command -v stty",
		"base64":           "command -v base64",
		"uudecode":         "command -v uudecode",
		"tr":               "command -v tr",
		"sha512sum":        "command -v sha512sum",
		"python3":          "python3 --version",
		"wc":               "command -v wc",
		"cut":              "command -v cut",
		"paste":            "command -v paste",
		"gunzip":           "command -v gunzip",
		"curl":             "command -v curl",
		"wget":             "command -v wget",
		"dig":              "command -v dig",
		"ping":             "command -v ping",
		"nc":               "command -v nc",
		"uname":            "uname -a",
		"./unbound-ssh -v": "./unbound-ssh -v",
		"busybox":          "busybox | head -n 1",
		"/etc/lsb-release": "cat /etc/lsb-release",
		"sw_vers":          "/usr/bin/sw_vers",
	}
	collector = &NixProbeResult{}
	command, collector.BatchNamedCommandResult = signature.GenerateNamedCommandsAndCaptureResult(cmd)
	return command, collector
}

func (p *NixProbeResult) Os() string {
	uname := p.Get("uname")
	if uname.Result != 0 {
		return ""
	}

	if strings.HasPrefix(uname.Output, "Darwin") {
		return "darwin"
	} else if strings.HasPrefix(uname.Output, "Linux") {
		return "linux"
	} else {
		return ""
	}
}

func (p *NixProbeResult) Arch() string {
	uname := p.Get("uname")
	if uname.Result != 0 {
		return ""
	}

	if strings.Contains(uname.Output, "arm64") {
		return "arm64"
	} else if strings.Contains(uname.Output, "x86_64") {
		return "amd64"
	} else {
		return ""
	}
}

func (p *NixProbeResult) Binary() string {
	os := p.Os()
	arch := p.Arch()
	if os == "" || arch == "" {
		return ""
	}
	return fmt.Sprintf("unbound-ssh_%s_%s", os, arch)
}

func (p *NixProbeResult) HasGunzip() bool {
	return p.Get("gunzip").IsSuccess()
}

func (p *NixProbeResult) BinaryDownloadUrl() string {
	binary := p.Binary()
	if binary == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/nimatrueway/unbound-ssh/releases/latest/download/%s", binary)
}

func (p *NixProbeResult) HasPython3() bool {
	return p.Get("python3").IsSuccess()
}

func (p *NixProbeResult) IsBusyBox() bool {
	return p.Get("busybox").IsSuccess()
}

func (p *NixProbeResult) HasInternetAccess() bool {
	return p.Get("internet").IsSuccess() && !config.Config.Preflight.AssumeNoInternet
}

func (p *NixProbeResult) HasCurl() bool {
	return p.Get("curl").IsSuccess()
}

func (p *NixProbeResult) HasWget() bool {
	return p.Get("wget").IsSuccess()
}

func (p *NixProbeResult) HasUpdatedUnboundSsh() bool {
	res := p.Get("./unbound-ssh -v")
	if res.Result != 0 {
		return false
	}
	splits := strings.SplitN(res.Output, " ", 3)
	if len(splits) != 3 || splits[0] != "unbound-ssh" || splits[1] != "version" {
		return false
	}

	return semver.Compare(splits[2], config.Version) >= 0
}
