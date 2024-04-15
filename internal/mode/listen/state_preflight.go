package listen

import (
	"context"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/term"
	"github.com/sirupsen/logrus"
	"os"
)

// type the following in console on the server side to start preflight mode
// > echo unbound-ssh:preflight:$(date +%s)

type PreflightState struct {
	*BaseState
}

func NewPreflightState(baseState *BaseState) PreflightState {
	return PreflightState{BaseState: baseState}
}

func (bm *PreflightState) Run(ctx context.Context) (err error) {
	logrus.Debug("transitioned to preflight state")

	// allow user to kill the process with ctrl+c or ctrl+d
	ctx, connectedStateCloser := context.WithCancel(ctx)
	reactToClose := term.NewReadInBackground(bm.BaseState.Stdin.BindTo(ctx))
	stop := func() {
		logrus.Info("received interrupt signal, existing preflight mode.")
		connectedStateCloser()
	}
	reactToClose.ReactTo(EndOfText, stop).Start(ctx)
	defer connectedStateCloser()

	// TODO: make this cancellable by stdin ctrl+c/ctrl+d
	shell := ShellExecutor{
		PtyReader:      bm.PtyStdout,
		PtyWriter:      bm.Pty,
		DefaultTimeout: config.Config.Transfer.Control.RequestTimeout,
	}

	// print error on stdout if preflight failed
	defer func() {
		if err != nil {
			_, err := bm.Pty.Write([]byte(fmt.Sprintf("printf \"\n\n\n\\033[0;31m   preflight failed:\n\n   %s\\033[0m\n\n\n\"\n", err.Error())))
			if err != nil {
				logrus.Errorf("failed to write error message to robot: %s", err.Error())
			}
		}
	}()

	// disable history for the duration of the preflight
	histFileRes, err := shell.Execute(ctx, "echo $HISTFILE", nil)
	if err != nil {
		logrus.Errorf("failed to get HISTFILE: %s", err.Error())
		return err
	}
	_, err = shell.Execute(ctx, "unset HISTFILE", nil)
	if err != nil {
		logrus.Errorf("failed to disable history: %s", err.Error())
		return err
	}
	defer func() {
		_, err := shell.Execute(ctx, fmt.Sprintf("export HISTFILE=\"%s\"", histFileRes.Output), nil)
		if err != nil {
			logrus.Errorf("failed to re-enable history: %s", err.Error())
		}
	}()

	// start the full duplex transfer
	GlobalProbeResult, err := Execute[*NixProbeResult](ctx, &shell, nil)(GeneratePrintNixProbe())
	if err != nil {
		logrus.Errorf("failed to write available commands probe to robot: %s", err.Error())
		return err
	} else {
		logrus.Infof("available *nix commands probed: %#v", GlobalProbeResult)
	}

	enc := GzipAscii85
	if !GlobalProbeResult.HasPython3() {
		enc = GzipBase64
	}

	logrus.Info("uploading file.")
	if !GlobalProbeResult.HasUpdatedUnboundSsh() {
		url := GlobalProbeResult.BinaryDownloadUrl()
		filename := "unbound-ssh"

		if GlobalProbeResult.HasInternetAccess() && GlobalProbeResult.HasWget() {
			_, err = shell.Execute(ctx, fmt.Sprintf(`wget -O "%[1]s.tmp" "%[2]s" && mv "%[1]s.tmp" %[1]s`, filename, url), nil)
			if err != nil {
				logrus.Errorf("failed to download %s binary using wget: %s", filename, err.Error())
			}
		} else if GlobalProbeResult.HasInternetAccess() && GlobalProbeResult.HasCurl() {
			_, err = shell.Execute(ctx, fmt.Sprintf(`curl -L -O "%[1]s.tmp" "%[2]s" && mv "%[1]s.tmp" %[1]s`, filename, url), nil)
			if err != nil {
				logrus.Errorf("failed to download %s binary using curl: %s", filename, err.Error())
			}
		} else {
			err = FetchUrlAndUpload(ctx, &shell, url, filename, enc)
			if err != nil {
				logrus.Errorf("failed to upload file: %s", err.Error())
			}
		}

		if err != nil {
			return err
		} else {
			logrus.Info("successfully uploaded file.")
		}

		_, err = shell.Execute(ctx, fmt.Sprintf("chmod +x %s", filename), nil)
		if err != nil {
			logrus.Errorf("failed to make it executable: %s", err.Error())
			return nil
		}
	}

	// collect all dependency files and their content to upload
	dependencyFiles := make(map[string]string)
	conf := config.Config.Clone()
	for i := range conf.Service {
		certFile := conf.Service[i].Certificate
		if certFile != "" {
			data, err := os.ReadFile(conf.Service[i].Certificate)
			if err != nil {
				logrus.Errorf("error reading certificate file %s to transfer to server: %s", conf.Service[i].Certificate, err.Error())
			} else {
				conf.Service[i].Certificate = fmt.Sprintf("service%d_certificate.pem", i)
				dependencyFiles[conf.Service[i].Certificate] = string(data)
			}
		}
	}
	dependencyFiles["config.toml"] = conf.SaveData()

	// upload all dependency files
	for filename, content := range dependencyFiles {
		err = Upload(ctx, &shell, content, filename, enc)
		if err != nil {
			logrus.Errorf("failed to upload file: %s", err.Error())
			return err
		}
	}

	return nil
}
