package signature

import (
	"fmt"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type BatchCommandResult struct {
	*CaptureResult
	Results []CommandResult
}

type CommandResult struct {
	Result int
	Output string
}

var NullCommandResult = CommandResult{Result: -1, Output: "<unset>"}

func (cr CommandResult) IsSuccess() bool {
	return cr.Result == 0
}

func GenerateCommandsAndCaptureResult(commands []string) (command string, collector *BatchCommandResult) {
	prep := `set -o pipefail; ` + strings.Join(lo.Map(commands, func(cmd string, i int) string {
		return fmt.Sprintf(`o_%[1]d=$(%[2]s | awk '{print}' ORS='\\\\n'); r_%[1]d=$?; `, i, cmd)
	}), "")
	output := strings.Join(lo.Map(commands, func(cmd string, i int) string {
		return fmt.Sprintf(`r_%[1]d=$r_%[1]d\no_%[1]d=$o_%[1]d`, i)
	}), "\\n")

	collector = &BatchCommandResult{}
	command, collector.CaptureResult = GenerateCommandAndCaptureResult(prep, output)

	return command, collector
}

func (p *BatchCommandResult) Find(in string) (matchEndIndex int) {
	defer func() {
		if matchEndIndex != -1 {
			p.parseOutput(p.Captured)
		}
	}()
	return p.CaptureResult.Find(in)
}

func (p *BatchCommandResult) parseOutput(output string) {
	split := strings.Split(output, "\n")
	p.Results = make([]CommandResult, len(split)/2)
	for i := range p.Results {
		p.Results[i] = NullCommandResult
	}

	for _, s := range split {
		kv := strings.SplitN(s, "=", 2)
		if len(kv) != 2 {
			logrus.Warnf("failed to split the result line: %s", s)
			continue
		}
		arrayVal := kv[1]
		arrayVar := strings.SplitN(kv[0], "_", 2)
		if len(arrayVar) != 2 {
			logrus.Warnf("failed to split the result line: %s", s)
			continue
		}
		arrayVarName := arrayVar[0]
		arrayVarIdx, err := strconv.Atoi(arrayVar[1])
		if err != nil || arrayVarIdx < 0 || arrayVarIdx >= len(p.Results) {
			logrus.Warnf("failed to parse the result line: %s", s)
			continue
		}
		if arrayVarName == "o" {
			p.Results[arrayVarIdx].Output = strings.TrimSpace(strings.ReplaceAll(arrayVal, `\n`, "\n"))
		} else if arrayVarName == "r" {
			res, err := strconv.Atoi(arrayVal)
			if err != nil {
				p.Results[arrayVarIdx].Result = -1
				logrus.Warnf("failed to parse the result line: %s", s)
				continue
			}
			p.Results[arrayVarIdx].Result = res
		} else {
			logrus.Warnf("failed to parse the result line: %s", s)
		}
	}
}
