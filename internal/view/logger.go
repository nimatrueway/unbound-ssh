package view

import (
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/nimatrueway/unbound-ssh/internal/config"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"runtime"
)

var LogFile = os.Stdout

func AppendRaw(str string) {
	_, err := LogFile.Write([]byte(str))
	if err != nil {
		panic(fmt.Sprintf("Failed to open log LogFile."))
	}
}

func Init() {
	file := config.ProcessString(config.Config.Log.File)
	logFile, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log LogFile: %s", file))
	}
	// global variable used for AppendRaw
	LogFile = logFile

	logrus.SetOutput(LogFile)
	logrus.SetLevel(config.Config.Log.Level)
}

func init() {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&nested.Formatter{
		NoColors:        true,
		TimestampFormat: "2006-01-02 15:04:05.000 ",
		CallerFirst:     true,
		CustomCallerFormatter: func(f *runtime.Frame) string {
			return fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
		},
	})
}
