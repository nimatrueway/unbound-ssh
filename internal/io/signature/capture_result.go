package signature

import (
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

var resultCaptureRegex = regexp.MustCompile("________capture_begin_([0-9a-z]{32})________[\r\n]{1,2}((?s).*)[\r\n]{1,2}_________capture_end_([0-9a-z]{32})_________[\r\n]{1,2}")

const resultCaptureFmt = " { %[2]sprintf \"________capture_begin_%[1]s________\\n%[3]s\\n_________capture_end_%[1]s_________\\n\"; }\n"

type CaptureResult struct {
	Id       uuid.UUID
	Captured string
}

func GenerateCommandAndCaptureResult(prep string, output string) (command string, collector *CaptureResult) {
	id := uuid.New()
	b := make([]byte, 16)
	copy(b[:], id[:])
	return fmt.Sprintf(resultCaptureFmt, hex.EncodeToString(b), prep, output), &CaptureResult{Id: id}
}

func (p *CaptureResult) GenerateLookupRegex() *regexp.Regexp {
	original := resultCaptureRegex.String()
	id, _ := p.Id.MarshalBinary()
	pattern := strings.ReplaceAll(original, "[0-9a-z]{32}", hex.EncodeToString(id))
	return regexp.MustCompile(pattern)
}

func (p *CaptureResult) Find(in string) (matchEndIndex int) {
	groups, matchEndIndex := findRegex(in, resultCaptureRegex)
	if groups == nil {
		return matchEndIndex
	}

	id1, err := hex.DecodeString(groups[1])
	if err != nil {
		return matchEndIndex
	}
	id2, err := hex.DecodeString(groups[3])
	if err != nil {
		return matchEndIndex
	}
	uuid1, err := uuid.FromBytes(id1)
	if err != nil {
		return matchEndIndex
	}
	uuid2, err := uuid.FromBytes(id2)
	if err != nil {
		return matchEndIndex
	}
	if uuid1 != uuid2 {
		logrus.Warnf("command result ids did not match: %+v", p)
		return matchEndIndex
	}

	p.Id = uuid1

	p.Captured = strings.TrimSpace(regexp.MustCompile("\r+\n").ReplaceAllString(groups[2], "\n"))

	return matchEndIndex
}
