package mode

import (
	"encoding/json"
	"github.com/ztrue/tracerr"
	"reflect"
	"time"
)

type MessageId = uint32

var typeRegistry = []any{HelloExchange{}, RegisterStreamExchange{}}
var commandRegistry map[string]reflect.Type

// ----------------------------------------------------------------------------------------

type Exchange[Request, Response any] struct {
	_ Request
	_ Response
}

// HelloExchange used to send hello from listen-mode to spy-mode and receive a response
type HelloExchange = Exchange[HelloRequest, HelloResponse]
type HelloRequest struct{}
type HelloResponse struct{}

// RegisterStreamExchange used to communicate from listen-mode to spy-mode how to serve the next established stream
type RegisterStreamExchange = Exchange[RegisterStreamRequest, RegisterStreamResponse]
type RegisterStreamRequest struct {
	StreamId      uint32 `json:"stream_id"`
	ServiceNumber int    `json:"service_number"`
}
type RegisterStreamResponse struct {
	Error string `json:"error"`
}

// ----------------------------------------------------------------------------------------

type ControlMessage struct {
	Id           MessageId     `json:"id,omitempty"`         // id of the message, unique per sender
	RequestId    MessageId     `json:"request_id,omitempty"` // if this is a response to a request
	ProcessDelay time.Duration `json:"-"`
	Command      string        `json:"command"` // the struct name of "Args" to allow deserializing json later
	Args         any           `json:"args"`    // any of request / response types in the exchange "typeRegistry"
}

func (cm *ControlMessage) IsResponse() bool {
	return cm.RequestId != MessageId(0)
}

func (cm *ControlMessage) String() string {
	marshal, err := Marshal(cm)
	if err != nil {
		return tracerr.Wrap(err).Error()
	}

	return string(marshal)
}

func Marshal(controlMessage *ControlMessage) ([]byte, error) {
	argType := reflect.TypeOf(controlMessage.Args)
	for k, t := range commandRegistry {
		if argType == t {
			controlMessage.Command = k
			marshal, err := json.Marshal(*controlMessage)
			if err != nil {
				return nil, tracerr.Wrap(err)
			}

			return marshal, nil
		}
	}
	return nil, tracerr.Errorf("unknown type: %T", controlMessage.Args)
}

func Unmarshal(data []byte) (ControlMessage, error) {
	var message ControlMessage
	err := json.Unmarshal(data, &message)
	if err != nil {
		return message, tracerr.Wrap(err)
	}

	t, ok := commandRegistry[message.Command]
	if !ok {
		return message, tracerr.Errorf("unknown command: %s", message.Command)
	}

	obj := reflect.New(t).Interface()
	message.Args = obj

	err = json.Unmarshal(data, &message)
	if err != nil {
		return message, tracerr.Wrap(err)
	}

	return message, nil
}

func TypeNameOf(t any) string {
	return reflect.TypeOf(t).Name()
}

func init() {
	commandRegistry = make(map[string]reflect.Type)
	for _, t := range typeRegistry {
		exchangeType := reflect.TypeOf(t)
		for i := 0; i < 2; i++ {
			typeReflect := exchangeType.Field(i).Type
			typeName := typeReflect.Name()
			commandRegistry[typeName] = typeReflect
		}
	}
}
