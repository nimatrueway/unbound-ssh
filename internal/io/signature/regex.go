package signature

import (
	"regexp"
)

type RegexSignature struct {
	regex  *regexp.Regexp
	Groups []string
}

func NewRegexSignature(regex *regexp.Regexp) *RegexSignature {
	return &RegexSignature{regex: regex}
}

func (s *RegexSignature) Find(in string) (matchEndIndex int) {
	s.Groups, matchEndIndex = findRegex(in, s.regex)
	return matchEndIndex
}
