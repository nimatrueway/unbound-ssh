package mode

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEmptyArgs(t *testing.T) {
	expected := `{"id":2,"command":"HelloRequest","args":{}}`
	arg := HelloRequest{}

	// marshal it
	{
		actual, err := Marshal(&ControlMessage{
			Id:   2,
			Args: arg,
		})

		require.Nil(t, err)
		require.Equal(t, expected, string(actual))
	}

	// unmarshal it back
	{
		msg, err := Unmarshal([]byte(expected))

		require.Nil(t, err)
		require.Equal(t, MessageId(2), msg.Id)
		require.Equal(t, msg.Args, &arg)
	}
}

func TestNonEmptyArgs(t *testing.T) {
	expected := `{"id":2,"command":"RegisterStreamResponse","args":{"error":"error-msg"}}`
	arg := RegisterStreamResponse{Error: "error-msg"}

	// marshal it
	{
		actual, err := Marshal(&ControlMessage{
			Id:   2,
			Args: arg,
		})

		require.Nil(t, err)
		require.Equal(t, expected, string(actual))
	}

	// unmarshal it back
	{
		msg, err := Unmarshal([]byte(expected))

		require.Nil(t, err)
		require.Equal(t, MessageId(2), msg.Id)
		require.Equal(t, msg.Args, &arg)
	}
}
