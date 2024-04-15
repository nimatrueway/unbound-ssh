package signature

import (
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/sirupsen/logrus"
	"math"
	"math/rand/v2"
	"regexp"
	"time"
)

var SpyStartRegex = regexp.MustCompile("\\[spy] start{timestamp:\"([0-9A-Z]{16})\",seed:\"([0-9A-Z]{16})\",checksum:\"([0-9A-Z]{16})\"}")

const ListenConnectFmt = " [listen] connect{}"

const spyStartFmt = "[spy] start{timestamp:\"%s\",seed:\"%s\",checksum:\"%s\"}"

type SpyStart struct {
	timestamp uint64
	seed      uint64
	checksum  uint64
}

func GenerateSpyStart(s *SpyStart) string {
	return fmt.Sprintf(spyStartFmt, toHex(s.timestamp), toHex(s.seed), toHex(calculateChecksum(s.seed)))
}

func NewSpyStart() *SpyStart {
	s := SpyStart{}
	s.timestamp = uint64(time.Now().UnixNano())
	s.seed = rand.Uint64()
	s.checksum = calculateChecksum(s.seed)
	return &s
}

func (s *SpyStart) Find(in string) (matchEndIndex int) {
	groups, matchEndIndex := findRegex(in, SpyStartRegex)
	if matchEndIndex == -1 {
		return matchEndIndex
	}

	s.timestamp, _ = fromHex(groups[1])
	s.seed, _ = fromHex(groups[2])
	s.checksum, _ = fromHex(groups[3])

	if !s.isChecksumValid() {
		logrus.Warnf("invalid checksum on handshake signature: %+v", s)
		return matchEndIndex
	}

	if s.timestampAge() > config.Config.Transfer.ConnectionTimeout {
		logrus.Warnf("the handshake signature was too old: %+v", s)
		return matchEndIndex
	}

	return matchEndIndex
}

func (s *SpyStart) isChecksumValid() bool {
	return s.checksum == calculateChecksum(s.seed)
}

func (s *SpyStart) timestampAge() time.Duration {
	return time.Duration(math.Abs(float64(s.timestamp) - float64(time.Now().UnixNano())))
}
