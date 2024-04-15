package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/yamux"
	"github.com/nimatrueway/unbound-ssh/internal/io/core"
	"github.com/nimatrueway/unbound-ssh/internal/mode"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	stdio "io"
	"net"
)

type YamuxStreamManager struct {
	silencer      stdio.Closer
	Session       *yamux.Session
	ControlStream *YamuxControlStream
	connections   []YamuxForwarder
	stdio.Closer
}

func NewYamuxForwarderManager(session *yamux.Session, silencer stdio.Closer) *YamuxStreamManager {
	return &YamuxStreamManager{
		silencer:    silencer,
		Session:     session,
		connections: make([]YamuxForwarder, 0),
	}
}

// ReceiveAndOpenYamux Used by listen-mode, to receive connections and forward them to the yamux Session
func (ym *YamuxStreamManager) ReceiveAndOpenYamux(ctx context.Context, serviceMan *ListenServiceManager) (err error) {
	if ym.ControlStream == nil {
		return tracerr.New("control stream is not opened")
	}

	defer func() {
		if err == stdio.EOF || !core.IsAlreadyClosed(err) /* this only makes sense if yamux is closed*/ {
			err = nil
		}
	}()

	// close the listener when the context is done
	go func() {
		isRemoteInitiated := func() bool {
			select {
			case <-ctx.Done():
				return false
			case <-ym.ControlStream.RemoteClosed:
				return true
			}
		}()

		if !isRemoteInitiated {
			logrus.Info("initiating yamux stream manager shutdown from listen-mode.")
		} else {
			logrus.Info("continuing yamux stream manager shutdown initiated by spy-mode.")
		}

		err := ym.Close(isRemoteInitiated)
		if err != nil {
			logrus.Warnf("Failed to close yamux stream manager: %s", err.Error())
		}
		err = serviceMan.Close()
		if err != nil {
			logrus.Warnf("Failed to close listener: %s", err.Error())
		}
	}()

	// serve the yamux sessions to the received connections
	for {
		conn, serviceNumber, err := serviceMan.Accept()
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			} else {
				return tracerr.Wrap(err)
			}
		}
		logrus.Info("opened a connection from client local-addr: ", conn.LocalAddr())

		stream, err := ym.Session.OpenStream()
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			} else {
				return tracerr.Wrap(err)
			}
		}
		logrus.Info("opened a yamux stream: ", stream.StreamID())

		forwarder := NewYamuxForwarder(conn, stream)
		ym.connections = append(ym.connections, forwarder)

		forwarder.start(ctx)

		registerStream := RpcCreateInvoker[mode.RegisterStreamExchange](ym.ControlStream)
		res, err := registerStream(mode.RegisterStreamRequest{StreamId: stream.StreamID(), ServiceNumber: serviceNumber})
		if err != nil {
			return err
		}
		if res.Error != "" {
			return tracerr.Errorf("failed to register connection map: %s", res.Error)
		}
	}
}

// AcceptYamuxAndForward Used by spy-mode, to forward the yamux Session to the received address
func (ym *YamuxStreamManager) AcceptYamuxAndForward(ctx context.Context, serviceMan *SpyServiceManager) (err error) {
	defer func() {
		if err == stdio.EOF || core.IsAlreadyClosed(err) /* this only makes sense if yamux is closed*/ {
			err = nil
		}
	}()

	if ym.ControlStream == nil {
		return tracerr.New("control stream is not accepted yet")
	}

	// close the listener when the context is done
	go func() {
		isRemoteInitiated := func() bool {
			select {
			case <-ctx.Done():
				return false
			case <-ym.ControlStream.RemoteClosed:
				return true
			}
		}()

		if !isRemoteInitiated {
			logrus.Info("initiating yamux stream manager shutdown from spy-mode.")
		} else {
			logrus.Info("continuing yamux stream manager shutdown initiated by listen-mode.")
		}

		err = ym.Close(isRemoteInitiated)
		if err != nil {
			logrus.Warnf("Failed to close yamux stream manager: %s", err.Error())
		}
	}()

	// serve the yamux sessions to the received connections
	for {
		yamuxStream, err := ym.Session.AcceptStream()
		if err != nil {
			if errors.Is(err, yamux.ErrSessionShutdown) {
				return nil
			}
			return tracerr.Wrap(err)
		}
		logrus.Debug("accepted a yamux stream: ", yamuxStream.StreamID())

		serviceNumber, err := ym.receiveServiceNumberOf(yamuxStream.StreamID())
		if err != nil {
			return err
		} else if serviceNumber == -1 {
			return tracerr.New("service number is not received")
		}

		addr := serviceMan.Addr(serviceNumber)
		conn, err := net.Dial(addr.Network(), addr.String())
		if err != nil {
			return tracerr.Wrap(err)
		}
		logrus.Debug("forwarding yamux connection traffic to: ", addr)

		forwarder := NewYamuxForwarder(conn, yamuxStream)
		ym.connections = append(ym.connections, forwarder)

		forwarder.start(ctx)
	}
}

func (ym *YamuxStreamManager) receiveServiceNumberOf(streamId uint32) (int, error) {
	var internalErr error
	serviceNumber := -1

	err := RpcExpectAndRespond(ym.ControlStream, func(streamService mode.RegisterStreamRequest) (mode.RegisterStreamResponse, error) {
		if streamService.StreamId != streamId {
			internalErr = fmt.Errorf("stream id mismatch: %d (control) != %d (yamux)", streamService.StreamId, streamId)
			return mode.RegisterStreamResponse{Error: internalErr.Error()}, nil
		}

		serviceNumber = streamService.ServiceNumber
		return mode.RegisterStreamResponse{}, nil
	})
	if err != nil {
		return -1, err
	}
	if internalErr != nil {
		return -1, err
	}
	return serviceNumber, nil
}

func (ym *YamuxStreamManager) Close(isRemoteInitiated bool) error {
	var err error
	errs := make([]error, 0)
	logrus.Debug("yamux stream manager is closing.")

	if isRemoteInitiated {
		// listen-mode often initiates the shutdown of the yamux control stream by sending a FIN to spy-mode, then
		// putting itself in half-closed state; meaning it stops writing on the wire.
		// spy-mode reacts to FIN flag by sending a FIN to listen-mode, which completes the closing of the stream.
		//
		// since we immediately shut down the whole session after closing the control stream, listen-mode does not
		// receive the FIN flag back so it ends up on the console as gibberish binary.
		//
		// to prevent this, on spy-mode we fully prevent yamux from writing on the wire by invoking the "silencer".
		_ = ym.silencer.Close()
	}

	if err = ym.ControlStream.Close(); err != nil {
		errs = append(errs, tracerr.Errorf("failed to close yamux control stream: %s", err.Error()))
	}

	for i, conn := range ym.connections {
		if err = conn.Close(); err != nil {
			errs = append(errs, err)
		} else {
			logrus.Debugf("yamux forwarder %d closed.", i)
		}
	}

	err = ym.Session.Close()
	if err != nil {
		errs = append(errs, tracerr.Errorf("failed to close yamux Session: %s", err.Error()))
	}
	logrus.Debugf("yamux session closed.")

	if len(errs) > 0 {
		return tracerr.Errorf("failed to close yamux stream manager: %v", errs)
	} else {
		return nil
	}
}
