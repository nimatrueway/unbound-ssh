package core

import (
	"bytes"
	"context"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"io"
	"math/rand/v2"
	"sort"
	"strings"
	"testing"
)

type SimpleSignature struct {
	Signature string
}

func (s *SimpleSignature) Find(in string) int {
	startIdx := strings.Index(in, s.Signature)
	if startIdx == -1 {
		return -1
	}

	return startIdx + len(s.Signature)
}

func TestSignatureDetection(t *testing.T) {
	ctx := context.Background()
	//for i := 0; i < 10000 && !t.Failed(); i++ {
	contextReader := NewContextReader(randomSplitReader("🇨🇦 hello world from Canada! 🇺🇸 hello world from USA!"))
	detector := NewSignatureDetector(&SimpleSignature{"hello world"})
	spyReader := detector.Wrap(contextReader).BindTo(ctx)

	{ // will read until the signatures is found
		chunkUtilAfterSignature, err := io.ReadAll(spyReader)
		require.Equal(t, "🇨🇦 hello world", string(chunkUtilAfterSignature))
		require.ErrorIs(t, err, SignatureFound)
	}

	{ // will read until the signatures is found
		detector.Reset()
		chunkUtilAfterSignature, err := io.ReadAll(spyReader)
		require.Equal(t, " from Canada! 🇺🇸 hello world", string(chunkUtilAfterSignature))
		require.ErrorIs(t, err, SignatureFound)
	}

	{ // continue reading safely from the underlying io after the signatures
		remainder, err := io.ReadAll(contextReader.BindTo(ctx))
		require.ErrorIs(t, err, nil)
		require.Equal(t, " from USA!", string(remainder))
	}
	//}
}

func randomSplitReader(str string) io.Reader {
	var indices []int
	for i := 0; i < len(str)-1; i += rand.IntN(len(str)-i-1) + 1 {
		indices = append(indices, i)
	}
	sort.Ints(indices)

	ranges := lo.Zip2(indices, append(append([]int{}, indices[1:]...), len(str)))
	slices := lo.Map(ranges, func(pair lo.Tuple2[int, int], _ int) string {
		return str[pair.A:pair.B]
	})

	return io.MultiReader(lo.Map(slices, func(s string, _ int) io.Reader {
		return bytes.NewBuffer([]byte(s))
	})...)
}
