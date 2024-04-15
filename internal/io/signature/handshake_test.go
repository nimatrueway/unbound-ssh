package signature

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSpyHelloSignature(t *testing.T) {
	now := uint64(time.Now().UnixNano())
	sig := GenerateSpyStart(NewSpyStart())

	extracted := SpyStart{}
	matchEndIndex := (&extracted).Find(sig)

	require.Equal(t, len(sig), matchEndIndex)
	require.Less(t, int64(extracted.timestamp)-int64(now), time.Second)
}
