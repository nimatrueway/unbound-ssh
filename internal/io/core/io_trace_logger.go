package core

import (
	"fmt"
	"github.com/hashicorp/yamux"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"strings"
)

type WriterLogInterceptor struct {
	io.Writer
	LogLevel logrus.Level
	Prefix   string
}

func NewWriterLogInterceptor(w io.Writer) *WriterLogInterceptor {
	return &WriterLogInterceptor{Writer: w, LogLevel: logrus.TraceLevel}
}

func (wi *WriterLogInterceptor) Write(p []byte) (n int, err error) {
	write, err := wi.Writer.Write(p)

	writerName := DetermineWriterName(wi.Writer)
	if write > 0 {
		logrus.StandardLogger().Logf(wi.LogLevel, "%swrote %d bytes to \"%s\": %#v", wi.Prefix, len(p), writerName, string(p))
	}

	return write, err
}

// --------------------------------------------------------------------------------------------------------------------

type ReaderLogInterceptor struct {
	io.Reader
	LogLevel logrus.Level
	Prefix   string
}

func NewReaderLogInterceptor(r io.Reader) *ReaderLogInterceptor {
	return &ReaderLogInterceptor{Reader: r, LogLevel: logrus.TraceLevel}
}

func (ri *ReaderLogInterceptor) Read(p []byte) (n int, err error) {
	read, err := ri.Reader.Read(p)

	readerName := DetermineReaderName(ri.Reader)
	if read > 0 {
		logrus.StandardLogger().Logf(ri.LogLevel, "%sread %d bytes from \"%s\": %#v", ri.Prefix, read, readerName, string(p[:read]))
	}
	if err != nil && err != io.EOF {
		logrus.StandardLogger().Logf(ri.LogLevel, "%sread error from \"%s\": %s", ri.Prefix, readerName, err.Error())
	}

	return read, err
}

func DetermineWriterName(writer io.Writer) string {
	if logInterceptor, ok := writer.(*WriterLogInterceptor); ok {
		return DetermineWriterName(logInterceptor.Writer)
	} else if w, ok := writer.(*withRwCloser); ok {
		return DetermineWriterName(w.ReadWriter)
	} else if w, ok := writer.(*ContextBoundWriter); ok {
		return DetermineWriterName(w.w)
	} else if w, ok := writer.(*readWriter); ok {
		return DetermineWriterName(w.Writer)
	} else if yamuxStream, ok := writer.(*yamux.Stream); ok {
		return fmt.Sprintf("yamux://stream/%d", yamuxStream.StreamID())
	} else if conn, ok := writer.(net.Conn); ok {
		return determineConnectionName(conn)
	} else if writer == os.Stdout {
		return fmt.Sprintf("stdout")
	} else if file, ok := writer.(*os.File); ok {
		return fmt.Sprintf("file://%s", file.Name())
	} else {
		return fmt.Sprintf("%T", writer)
	}
}

func DetermineReaderName(reader io.Reader) string {
	if r, ok := reader.(*ReaderLogInterceptor); ok {
		return DetermineReaderName(r.Reader)
	} else if r, ok := reader.(*withRwCloser); ok {
		return DetermineReaderName(r.ReadWriter)
	} else if r, ok := reader.(*readWriter); ok {
		return DetermineReaderName(r.Reader)
	} else if r, ok := reader.(*ContextBoundReader); ok {
		return DetermineReaderName(r.r.r)
	} else if r, ok := reader.(*ContextBoundReadCloser); ok {
		return DetermineReaderName(r.ContextReadCloser.ReadCloser)
	} else if r, ok := reader.(*ContextReadCloser); ok {
		return DetermineReaderName(r.ReadCloser)
	} else if r, ok := reader.(*SignatureDetectorContextBoundReader); ok {
		return DetermineReaderName(r.r.r.r)
	} else if yamuxStream, ok := reader.(*yamux.Stream); ok {
		return fmt.Sprintf("yamux://stream/%d", yamuxStream.StreamID())
	} else if conn, ok := reader.(net.Conn); ok {
		return determineConnectionName(conn)
	} else if reader == os.Stdin {
		return fmt.Sprintf("stdin")
	} else if file, ok := reader.(*os.File); ok {
		return fmt.Sprintf("file://%s", file.Name())
	} else {
		return fmt.Sprintf("%T", reader)
	}
}

func determineConnectionName(conn net.Conn) string {
	network := conn.LocalAddr().Network()
	addr := conn.LocalAddr().String()
	if network == "unix" {
		if addr == "" {
			f, err := conn.(*net.UnixConn).File()
			if err == nil {
				addr = f.Name()
				if strings.HasPrefix(addr, "unix:->") {
					addr = addr[7:]
				}
			}
		}
		if strings.HasPrefix(addr, os.TempDir()) {
			addr = "/$TMPDIR/" + addr[len(os.TempDir()):]
		}
	}
	return fmt.Sprintf("%s://%s", network, addr)
}
