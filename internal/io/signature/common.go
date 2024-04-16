package signature

import (
	"encoding/binary"
	"encoding/hex"
	"hash/crc64"
	"regexp"
	"strings"
)

type Signature interface {
	Find(input string) (matchEndIndex int)
}

func findRegex(in string, re *regexp.Regexp) (groups []string, lastMatchIdx int) {
	lastMatchIdx = -1
	matches := re.FindStringSubmatch(in)
	if matches == nil || len(matches) == 0 {
		return nil, lastMatchIdx
	}

	lastMatchIdx = strings.Index(in, matches[0])
	if lastMatchIdx > -1 {
		lastMatchIdx += len(matches[0])
	}
	return matches, lastMatchIdx
}

func toHex(v uint64) string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return strings.ToUpper(hex.EncodeToString(b))
}

func fromHex(v string) (uint64, error) {
	bytes, err := hex.DecodeString(v)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(bytes), nil
}

func calculateChecksum(num uint64) uint64 {
	randomBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(randomBytes, num)
	return crc64.Checksum(randomBytes, crc64.MakeTable(crc64.ECMA))
}
