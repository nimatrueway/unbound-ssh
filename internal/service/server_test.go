package service

// var ListenAddr = config.Address{Network: "tcp", String: "unbound-ssh.local:2222"}
//var ListenAddr = config.Address{Network: "unix", String: "/tmp/unbound-ssh.sock"}
//
//type LogrusToTestingLog struct {
//	t *testing.T
//}
//
//func (x *LogrusToTestingLog) Write(p []byte) (n int, err error) {
//	x.t.Log(string(p[:len(p)-1]))
//	return len(p), nil
//}
//
//func TestServer(t *testing.T) {
//	logrus.SetOutput(&LogrusToTestingLog{t})
//	logrus.SetLevel(logrus.TraceLevel)
//
//	go func() {
//		shutdown.Listen(syscall.SIGINT)
//	}()
//
//	addr := ListenAddr
//	listener, err := net.Listen(addr.Network, addr.String)
//	require.NoError(t, err)
//	closeFn := func() {
//		err := listener.Close()
//		require.NoError(t, err)
//	}
//	shutdown.Add(closeFn)
//	defer closeFn()
//
//	// start ftp server
//	server, err := CreateWebdavServer()
//	require.NoError(t, err)
//
//	if addr.Network == "unix" {
//		logrus.Infof("local webdav server listening on unix://%s", listener.Addr().String())
//		logrus.Info("use the following to expose it as tcp:")
//		logrus.Infof("socat TCP-LISTEN:2222 UNIX-CONNECT:%s", listener.Addr().String())
//		logrus.Info("webdav 127.0.0.1 2222")
//	} else {
//		logrus.Infof("local webdav server listening on %s", listener.Addr().String())
//		pattern := regexp.MustCompile("\\[[:]+]:(\\d+)")
//		sshArg := pattern.ReplaceAllString(listener.Addr().String(), "-p $1 localhost")
//		logrus.Infof("webdav %s", sshArg)
//	}
//
//	// start ssh server
//	//server, err := CreateSshServer()
//	//require.NoError(t, err)
//	//
//	//if addr.Network() == "unix" {
//	//	logrus.Infof("local ssh server listening on unix://%s", listener.Addr().String())
//	//	logrus.Info("use the following to connect:")
//	//	logrus.Infof("ssh -o 'NoHostAuthenticationForLocalhost=yes' -o 'ProxyCommand socat - UNIX-CLIENT:%s' foo", listener.Addr().String())
//	//} else {
//	//	logrus.Infof("local ssh server listening on %s", listener.Addr().String())
//	//	pattern := regexp.MustCompile("\\[[:]+]:(\\d+)")
//	//	sshArg := pattern.ReplaceAllString(listener.Addr().String(), "-p $1 localhost")
//	//	logrus.Infof("ssh -o 'NoHostAuthenticationForLocalhost=yes' %s", sshArg)
//	//}
//
//	err = server.Serve(listener)
//	require.NoError(t, err)
//}
