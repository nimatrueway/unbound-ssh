package signature

import (
	"regexp"
)

const Bell byte = 7 // Ctrl+G in ascii
var preflightRegex = regexp.MustCompile(string([]byte{Bell, Bell, Bell}))

type Preflight struct{}

func (p *Preflight) Find(in string) (matchEndIndex int) {
	_, matchEndIndex = findRegex(in, preflightRegex)
	return
}
