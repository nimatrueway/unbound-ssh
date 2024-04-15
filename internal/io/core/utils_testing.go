package core

import (
	"github.com/samber/lo"
	"github.com/samber/mo"
	"io"
	"math/rand/v2"
	"sort"
	"sync"
)

type ChannelReader struct {
	buf  []mo.Result[[]byte]
	lock *sync.Mutex
	cond *sync.Cond
	done bool
}

func NewChannelReader() *ChannelReader {
	mutex := sync.Mutex{}
	cond := sync.NewCond(&mutex)
	return &ChannelReader{lock: &mutex, cond: cond}
}

func (r *ChannelReader) Fail(err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cond.Broadcast()

	r.buf = append(r.buf, mo.Err[[]byte](err))
}

func (r *ChannelReader) Write(p []byte) (n int, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cond.Broadcast()

	r.buf = append(r.buf, mo.Ok(p))

	return len(p), nil
}

func (r *ChannelReader) WriteString(p string) {
	_, _ = r.Write([]byte(p))
}

func RandomlySlice(str string) []string {
	if len(str) == 0 {
		return []string{}
	} else if len(str) == 1 {
		return []string{str}
	}

	var indices []int
	for i := 0; i < len(str)-1; i += rand.IntN(len(str)-i-1) + 1 {
		indices = append(indices, i)
	}
	sort.Ints(indices)

	ranges := lo.Zip2(indices, append(append([]int{}, indices...), len(str))[1:])
	slices := lo.Map(ranges, func(pair lo.Tuple2[int, int], _ int) string {
		return str[pair.A:pair.B]
	})
	return slices
}

func (r *ChannelReader) WriteStringInRandomChunks(str string) {
	for _, s := range RandomlySlice(str) {
		r.WriteString(s)
	}
}

func (r *ChannelReader) Read(p []byte) (n int, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for len(r.buf) == 0 {
		r.cond.Wait()
	}

	result := r.buf[0]
	buffer, e := result.Get()

	// clear the buffer only if it not []{EOF}
	if e != io.EOF {
		r.buf = r.buf[1:]
	}

	if e != nil {
		return 0, e
	}

	if len(buffer) > len(p) {
		rest := mo.Ok(buffer[len(p):])
		singleton := []mo.Result[[]byte]{rest}
		r.buf = append(singleton, r.buf...)
	}

	return copy(p, buffer), nil
}

func (r *ChannelReader) IsEmpty() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return len(r.buf) == 0
}

func (r *ChannelReader) Drain() string {
	r.Fail(io.EOF)
	buf := make([]byte, 1024)
	n, _ := io.ReadFull(r, buf)
	return string(buf[:n])
}
