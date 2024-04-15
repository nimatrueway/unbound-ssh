package core

//import (
//	"context"
//	"io"
//)
//
//type ContextWriter struct {
//	w        io.Writer
//	writeBuf chan []byte
//}
//
//func NewContextWriter(w io.Writer) *ContextWriter {
//	core := &ContextWriter{
//		w: w,
//	}
//	core.startWriter()
//	return core
//}
//
//func (sr *ContextWriter) startWriter() {
//	for {
//		buf, ok := <-sr.writeBuf
//		if !ok {
//			return
//		}
//
//		_, _ = sr.w.Write(buf)
//	}
//}
//
//func (sr *ContextWriter) Write(ctx context.Context, p []byte) (int, error) {
//	select {
//	case <-ctx.Done():
//		return 0, context.Cause(ctx)
//	case sr.writeBuf <- p:
//		return len(p), nil
//	}
//}
