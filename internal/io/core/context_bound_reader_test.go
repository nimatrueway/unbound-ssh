package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInterrupt(t *testing.T) {
	ctx := context.Background()
	channelReader := NewChannelReader()
	contextReader := NewContextReader(channelReader)
	cancelableReader := contextReader.BindToConcrete(ctx)
	buf := make([]byte, 5)

	// cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancelableReader.Cancel()
	}()

	// data for Read() should be available after 250ms
	go func() {
		time.Sleep(250 * time.Millisecond)
		channelReader.WriteString("hello")
	}()

	{ // test that interrupting a read will return an Interrupted error
		n, err := cancelableReader.Read(buf)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected Canceled error, got (%d, %v) instead.", n, err)
			t.FailNow()
		}
	}

	{ // test a read will return an Interrupted error if it's not cleared
		n, err := cancelableReader.Read(buf)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected Interrupted error for the second time, got (%d, %v) instead.", n, err)
			t.FailNow()
		}
	}

	{ // test a read will return an Interrupted error if it's not cleared
		newCancelableReader := contextReader.BindTo(ctx)
		n, err := newCancelableReader.Read(buf)
		if n != 5 || err != nil {
			t.Errorf("Expected a successful read of 5 bytes, got (%d, %v) instead.", n, err)
			t.FailNow()
		}
	}
}
