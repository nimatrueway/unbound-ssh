package service

import (
	"bufio"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/mode"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
	stdio "io"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

var messageId mode.MessageId = 0

type YamuxControlStream struct {
	stream       stdio.ReadWriteCloser
	responseCh   map[mode.MessageId]chan any           // each Send() will put a channel here to wait for the response
	handlers     map[string]func(*mode.ControlMessage) // received messages will be processed to these handlers by their type name
	RemoteClosed chan any
	isClosed     bool
}

func NewControlStream(stream stdio.ReadWriteCloser) *YamuxControlStream {
	obj := YamuxControlStream{
		stream:       stream,
		responseCh:   make(map[mode.MessageId]chan any),
		handlers:     make(map[string]func(*mode.ControlMessage)),
		RemoteClosed: make(chan any),
	}
	obj.startReceive()
	return &obj
}

func (ycs *YamuxControlStream) startReceive() {
	reader := bufio.NewReader(ycs.stream)
	go func() {
		for {
			err := ycs.receiveOne(reader)
			if err != nil {
				if err == stdio.EOF {
					if !ycs.isClosed {
						logrus.Info("yamux control stream closed from the remote side.")
						close(ycs.RemoteClosed)
					}
				} else {
					logrus.Warnf("failed to receive command from the control stream: %s", err.Error())
				}
				return
			}
		}
	}()
}

func (ycs *YamuxControlStream) receiveOne(lineReader *bufio.Reader) error {
	line, err := lineReader.ReadBytes('\n')
	if err != nil {
		return err
	}
	trimmedLine := strings.TrimSpace(string(line))
	if trimmedLine == "" {
		return nil
	}

	msg, err := mode.Unmarshal([]byte(trimmedLine))
	if err != nil {
		return err
	}

	logrus.Debugf("received a command: %#v", msg)
	ycs.process(&msg)

	return nil
}

func (ycs *YamuxControlStream) process(msg *mode.ControlMessage) {
	if msg.IsResponse() {
		waiter := ycs.responseCh[msg.RequestId]

		if waiter != nil {
			waiter <- msg.Args
			logrus.Infof("processed the response command: %#v", msg)
		} else {
			ycs.queueForRetry(msg)
		}
	} else {
		handler := ycs.handlers[msg.Command]

		if handler != nil {
			go handler(msg)
			logrus.Infof("processed the received command: %#v", msg)
		} else {
			ycs.queueForRetry(msg)
		}
	}
}

func (ycs *YamuxControlStream) queueForRetry(msg *mode.ControlMessage) {
	if msg.ProcessDelay.Seconds() > config.Config.Transfer.RequestTimeout.Seconds() {
		if msg.IsResponse() {
			logrus.Warnf("received a response message with no handler: %#v", msg)
		} else {
			logrus.Warnf("could not process the request with message: %#v", msg)
		}
	} else {
		go func() {
			msg.ProcessDelay += 1 * time.Second

			// retry in a second
			time.Sleep(msg.ProcessDelay)
			ycs.process(msg)
		}()
	}
}

func (ycs *YamuxControlStream) send(content any, responseType any, responseTo mode.MessageId) (any, error) {
	id := atomic.AddUint32(&messageId, 1)
	msg := mode.ControlMessage{
		Id:        id,
		RequestId: responseTo,
		Args:      content,
	}
	cmdJson, err := mode.Marshal(&msg)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	_, err = ycs.stream.Write(cmdJson)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	_, err = ycs.stream.Write([]byte("\n"))
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	logrus.Debugf("sent a command: %#v", msg)

	if msg.IsResponse() {
		return nil, nil
	}

	ycs.responseCh[id] = make(chan any, 1)

	select {
	case response := <-ycs.responseCh[id]:
		if mode.TypeNameOf(&responseType) != mode.TypeNameOf(response) {
			return nil, fmt.Errorf("expected %s in response but received: %s", mode.TypeNameOf(responseType), mode.TypeNameOf(response))
		}
		return response, nil
	case <-time.After(config.Config.Transfer.RequestTimeout):
		return nil, tracerr.Errorf("did not receive a response for: %v", content)
	}
}

func (ycs *YamuxControlStream) Close() error {
	if ycs.isClosed {
		return nil
	}
	ycs.isClosed = true

	err := ycs.stream.Close()
	if err != nil {
		return tracerr.Wrap(err)
	}
	logrus.Debugf("yamux control stream closed.")
	return nil
}

func RpcCreateInvoker[_ mode.Exchange[Request, Response], Request any, Response any](ycs *YamuxControlStream) func(Request) (Response, error) {
	return func(req Request) (res Response, err error) {
		obj, err := ycs.send(req, nil, 0)
		if obj != nil && err == nil {
			resPtr := obj.(*Response)
			res = *resPtr
		}

		return res, err
	}
}

func RpcRegisterResponder[_ mode.Exchange[Request, Response], Request any, Response any](ycs *YamuxControlStream, f func(Request) Response) {
	typeName := reflect.TypeFor[Request]().Name()
	ycs.handlers[typeName] = func(msg *mode.ControlMessage) {
		req := msg.Args.(*Request)
		res := f(*req)
		if _, err := ycs.send(res, nil, msg.Id); err != nil {
			logrus.Errorf("failed to respond to %#v with %#v on the wire: %s", req, res, err.Error())
		}
	}
}

func RpcUnregisterResponder[_ mode.Exchange[Request, Response], Request any, Response any](ycs *YamuxControlStream) {
	typeName := reflect.TypeFor[Request]().Name()
	if ycs.handlers[typeName] != nil {
		ycs.handlers[typeName] = nil
	} else {
		logrus.Warnf("there was no handler for %s to unregister.", typeName)
	}
}

func RpcExpectAndRespond[E mode.Exchange[Request, Response], Request any, Response any](ycs *YamuxControlStream, f func(Request) (Response, error)) error {
	done := make(chan any)
	defer RpcUnregisterResponder[E](ycs)

	var err error
	RpcRegisterResponder(ycs, func(req Request) (res Response) {
		defer close(done)
		res, err = f(req)
		return res
	})

	select {
	case <-done:
		return err
	case <-time.After(config.Config.Transfer.RequestTimeout):
		typeName := reflect.TypeFor[Request]().Name()
		waitTime := config.Config.Transfer.RequestTimeout.String()
		return tracerr.Errorf("did not receive a message of type %s in the last %s", typeName, waitTime)
	}
}
