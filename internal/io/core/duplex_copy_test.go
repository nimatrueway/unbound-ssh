package core

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDuplexCopy(t *testing.T) {
	for i := 0; i < 100; i++ {
		ctx, cancelFn := context.WithCancel(context.Background())
		spyOutput := NewChannelReader()

		reader := NewChannelReader()
		cancelableReader := NewContextReader(reader)

		ptyReader := NewChannelReader()
		cancelablePtyReader := NewContextReader(ptyReader)

		go func() {
			// Simulate the origin and pty sending two chunks of data with 50ms delay
			reader.WriteString("<hello one>")
			ptyReader.WriteString("<hello two>")

			// wait until both readers are drained
			for {
				if reader.IsEmpty() && ptyReader.IsEmpty() {
					break
				}
				time.Sleep(1 * time.Millisecond)
			}
			// cancel the readers
			cancelFn()

			// these two won't be copied
			ptyReader.WriteString("<hello two x2>")
			reader.WriteString("<hello one x2>")
		}()

		// stream both origin and pty to stdout
		err := DuplexCopy(ctx, spyOutput, cancelableReader, spyOutput, cancelablePtyReader)
		if err != nil {
			_, _ = fmt.Fprintf(spyOutput, "<error: %s>", err.Error())
		}

		output := spyOutput.Drain()
		if !(output == "<hello one><hello two><error: context canceled>" ||
			output == "<hello two><hello one><error: context canceled>") {
			t.Errorf("cycle %d: expected output to be '<hello one><hello two><error: context canceled>', got: %s", i, output)
			t.FailNow()
		}
	}
}
