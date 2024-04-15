package core

import (
	"bufio"
	"context"
	"fmt"
	"github.com/samber/mo"
	"time"
)

// ExampleInterruptableReader_Interrupt
// demonstrates how to use the InterruptableReader to interrupt a read operation.
func ExampleContextBoundReader_Cancel() {
	ctx := context.Background()
	cancelableReader := createCancelableReader()
	reader := cancelableReader.BindToConcrete(ctx)
	lineCh := lineReaderChannel(reader)

	// first attempt to read a line
	select {
	case result := <-lineCh:
		printResult(result)
	case <-time.After(300 * time.Millisecond):
		fmt.Println("Result: timeout!")
	}

	// cancel the reader
	reader.Cancel()

	// second attempt to read a line
	select {
	case result := <-lineCh:
		printResult(result)
	case <-time.After(300 * time.Millisecond):
		fmt.Println("Result: timeout!")
	}

	// create a new copy of reader uncanceled
	newCancelableReader := cancelableReader.BindToConcrete(ctx)
	newLineCh := lineReaderChannel(newCancelableReader)

	// third attempt to read a line from the refreshed lineCh
	select {
	case result := <-newLineCh:
		printResult(result)
	case <-time.After(300 * time.Millisecond):
		fmt.Println("Result: timeout!")
	}

	// Output:
	// Result: timeout!
	// Result: error "context canceled"
	// Result: line "blahblah"
}

func createCancelableReader() *ContextReader {
	origin := NewChannelReader()
	cancelableOrigin := NewContextReader(origin)

	// Simulate a slow read operation that takes 300ms to complete
	go func() {
		time.Sleep(500 * time.Millisecond)
		origin.WriteString("blahblah\n")
	}()

	return cancelableOrigin
}

func printResult(r mo.Result[string]) {
	if line, err := r.Get(); err == nil {
		fmt.Printf("Result: line \"%s\"\n", line)
	} else {
		fmt.Printf("Result: error \"%v\"\n", err)
	}
}

func lineReaderChannel(reader *ContextBoundReader) <-chan mo.Result[string] {
	bufReader := bufio.NewReader(reader)
	ch := make(chan mo.Result[string])
	go func() {
		for {
			select {
			case <-reader.Context().Done():
				ch <- mo.Err[string](reader.Context().Err())
				return
			default:
				if line, _, err := bufReader.ReadLine(); err == nil {
					ch <- mo.Ok(string(line))
				} else {
					ch <- mo.Err[string](err)
					return
				}
			}
		}
	}()
	return ch
}
