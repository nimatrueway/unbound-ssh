package listen

import (
	hex2 "encoding/hex"
	"fmt"
	"github.com/nimatrueway/unbound-ssh/internal/io/signature"
	"github.com/sirupsen/logrus"
	"strconv"
)

type GetFileSizeAndHash struct {
	*signature.BatchCommandResult
	Size uint64
	Hash [32]byte
}

func GenerateGetFileSizeAndHash(filename string) (command string, collector *GetFileSizeAndHash) {
	collector = &GetFileSizeAndHash{}
	command, collector.BatchCommandResult = signature.GenerateCommandsAndCaptureResult([]string{
		fmt.Sprintf("wc -c '%s' | grep --color=never -oE '^[ ]*([0-9]+)'", filename),
		fmt.Sprintf("sha256sum '%s' | grep --color=never -oE '^([0-9a-z]+)'", filename),
	})
	return command, collector
}

func (p *GetFileSizeAndHash) Find(in string) (matchEndIndex int) {
	if idx := p.BatchCommandResult.Find(in); idx != -1 {
		fileSizeResult := p.Results[0]
		if fileSizeResult.Result != 0 {
			logrus.Warnf("failed to get file size: %s", fileSizeResult.Output)
			return -1
		}

		fileHashResult := p.Results[1]
		if fileHashResult.Result != 0 {
			logrus.Warnf("failed to get file sha256 hash: %s", fileHashResult.Output)
			return -1
		}

		size, err := strconv.ParseUint(fileSizeResult.Output, 10, 64)
		if err != nil {
			logrus.Warnf("failed to parse file size: %s", fileSizeResult.Output)
			return -1
		} else {
			p.Size = size
		}

		if len(fileHashResult.Output) != 64 {
			logrus.Warnf("failed to parse file hash: %s", fileSizeResult.Output)
			return -1
		} else if hash, err := hex2.DecodeString(fileHashResult.Output); err != nil {
			logrus.Warnf("failed to parse file hash: %s", fileSizeResult.Output)
			return -1
		} else {
			p.Hash = [32]byte(hash)
		}

		return idx
	}
	return -1
}
