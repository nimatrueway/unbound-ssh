package core

import (
	"bytes"
	"context"
	"errors"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/sirupsen/logrus"
	"io"
)

var SignatureFound = errors.New("signatures found")

var MaxSignatureLength = int(config.Config.Transfer.SignatureDetectorBuffer)

type DetectorState struct {
	recentBuffer *bytes.Buffer
	lastMatch    signature.Signature
}

type SignatureDetector struct {
	signatures []signature.Signature
	state      *DetectorState
}

func NewSignatureDetector(signatures ...signature.Signature) *SignatureDetector {
	return &SignatureDetector{
		signatures: signatures,
		state: &DetectorState{
			recentBuffer: &bytes.Buffer{},
			lastMatch:    nil,
		},
	}
}

func (c *SignatureDetector) Reset() {
	c.state.lastMatch = nil
}

func (c *SignatureDetector) LastMatch() signature.Signature {
	return c.state.lastMatch
}

func (c *SignatureDetector) Wrap(cbr *ContextReader) *SignatureDetectorContextBindingReader {
	return &SignatureDetectorContextBindingReader{
		SignatureDetector: c,
		cbr:               cbr,
	}
}

// -----------------------------------------------------------

type SignatureDetectorContextBindingReader struct {
	*SignatureDetector
	cbr *ContextReader
}

func (c *SignatureDetectorContextBindingReader) BindTo(ctx context.Context) io.ReadCloser {
	return &SignatureDetectorContextBoundReader{
		SignatureDetector: c.SignatureDetector,
		r:                 c.cbr.BindToConcrete(ctx),
	}
}

// -----------------------------------------------------------

type SignatureDetectorContextBoundReader struct {
	*SignatureDetector
	r *ContextBoundReader
}

func (c *SignatureDetectorContextBoundReader) Close() error {
	return c.r.Close()
}

func (c *SignatureDetectorContextBoundReader) Read(b []byte) (int, error) {
	if c.state.lastMatch != nil {
		return 0, SignatureFound
	}

	n, err := c.r.Read(b)
	if err != nil {
		return n, err
	}

	// TODO: optimize
	c.state.recentBuffer.Write(b[:n])

	if matchedEndIndex := c.findFirstSignature(); matchedEndIndex != -1 {
		buf := c.state.recentBuffer.Bytes()
		logrus.Debugf("signatures detected: %+v", c.state.lastMatch)
		c.r.UnreadBytes(buf[matchedEndIndex:])
		c.state.recentBuffer.Reset()

		newN := n - (len(buf) - matchedEndIndex)
		return newN, nil
	} else {
		if bufLen := c.state.recentBuffer.Len(); bufLen > MaxSignatureLength {
			c.state.recentBuffer.Next(bufLen - MaxSignatureLength)
		}
		return n, nil
	}
}

func (c *SignatureDetector) findFirstSignature() (matchEndIndex int) {
	recentBuffer := c.state.recentBuffer.String()
	matchEndIndex = -1

	for _, sig := range c.signatures {
		if currentIdx := sig.Find(recentBuffer); currentIdx != -1 && (matchEndIndex == -1 || currentIdx < matchEndIndex) {
			matchEndIndex = currentIdx
			c.state.lastMatch = sig
		}
	}

	return matchEndIndex
}
