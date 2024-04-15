package signature

import (
	"github.com/samber/lo"
)

type BatchNamedCommandResult struct {
	*BatchCommandResult
	names map[string]int
}

func GenerateNamedCommandsAndCaptureResult(commands map[string]string) (command string, collector *BatchNamedCommandResult) {
	nameCommandPairs := lo.MapToSlice(commands, func(cmd string, name string) lo.Tuple2[string, string] {
		return lo.Tuple2[string, string]{A: cmd, B: name}
	})
	collector = &BatchNamedCommandResult{names: lo.SliceToMap(lo.Map(nameCommandPairs, func(cmd lo.Tuple2[string, string], idx int) lo.Tuple2[string, int] {
		return lo.Tuple2[string, int]{A: cmd.A, B: idx}
	}), func(cmd lo.Tuple2[string, int]) (string, int) {
		return cmd.Unpack()
	})}
	command, collector.BatchCommandResult = GenerateCommandsAndCaptureResult(lo.Map(nameCommandPairs, func(cmd lo.Tuple2[string, string], _ int) string {
		return cmd.B
	}))
	return command, collector
}

func (b *BatchNamedCommandResult) Get(name string) CommandResult {
	if idx, ok := b.names[name]; ok {
		if idx < len(b.Results) {
			return b.Results[idx]
		}
		return NullCommandResult
	}
	return NullCommandResult
}
