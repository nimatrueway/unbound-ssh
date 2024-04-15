package service

import (
	"fmt"
	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"github.com/riywo/loginshell"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	stdssh "golang.org/x/crypto/ssh"
	"io"
	"os"
	"os/exec"
)

var shell string

func CreateSshServer(certificate string) (Server, error) {
	SftpHandler := func(sess ssh.Session) {
		debugStream := io.Discard
		serverOptions := []sftp.ServerOption{
			sftp.WithDebug(debugStream),
		}
		server, err := sftp.NewServer(
			sess,
			serverOptions...,
		)
		if err != nil {
			logrus.Infof("sftp server init error: %s\n", err)
			return
		}
		if err := server.Serve(); err == io.EOF {
			err = server.Close()
			if err != nil && err != io.EOF {
				logrus.Warn("sftp server close error:", err)
			}
			logrus.Info("sftp client exited session.")
		} else if err != nil {
			logrus.Warn("sftp server completed with error:", err)
		}
	}
	forwardHandler := &ssh.ForwardedTCPHandler{}
	setWinSize := func(f *os.File, w, h int) {
		err := pty.Setsize(f, &pty.Winsize{Cols: uint16(w), Rows: uint16(h)})
		if err != nil {
			logrus.Warn("failed to set window size: ", err)
		}
	}
	ssh.Handle(func(s ssh.Session) {
		cmd := exec.Command(shell)
		ptyReq, winCh, isPty := s.Pty()
		if isPty {
			termEnv := fmt.Sprintf("TERM=%s", ptyReq.Term)
			cmd.Env = append(cmd.Env, termEnv)
			logrus.Debug("launching shell.")
			logrus.Debugf("path: %s", cmd.Path)
			logrus.Debugf("args: %v", cmd.Args)
			logrus.Debugf("dir: %s", cmd.Env)
			logrus.Debugf("env: %v", cmd.Env)
			f, err := pty.Start(cmd)
			if err != nil {
				logrus.Errorf("failed to start pty: %s", err)
				panic(err)
			}
			go func() {
				for win := range winCh {
					setWinSize(f, win.Width, win.Height)
				}
			}()
			go func() {
				_, err := io.Copy(f, s) // stdin
				if err != nil {
					logrus.Warn(s, "pty forward transfer failed: ", err)
				}
			}()
			_, err = io.Copy(s, f) // stdout
			if err != nil {
				logrus.Warn(s, "pty reversed transfer failed: ", err)
			}

			err = cmd.Wait()
			if err != nil {
				logrus.Warn(s, "pty wait failed: ", err)
			}

		} else {
			logrus.Warn(s, "no pty requested.")
			s.Exit(1)
		}
	})

	open, err := os.Open(certificate)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	privateKeyContent, err := io.ReadAll(open)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	privateKey, err := stdssh.ParseRawPrivateKey(privateKeyContent)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	signer, err := stdssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	server := ssh.Server{
		HostSigners:     []ssh.Signer{signer},
		PasswordHandler: nil,
		Handler:         nil,
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": SftpHandler,
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session":      ssh.DefaultSessionHandler,
			"direct-tcpip": ssh.DirectTCPIPHandler,
		},
		LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
			logrus.Infof("accepted forward: %s:%d", dhost, dport)
			return true
		}),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			logrus.Info("attempt to bind", host, port, "granted")
			return true
		}),
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
	}

	return &server, nil
}

func init() {
	shell = os.Getenv("SHELL")
	if shell == "" {
		shell, _ = loginshell.Shell()
	}
}
