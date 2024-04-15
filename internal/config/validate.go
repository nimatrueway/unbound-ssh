package config

import (
	"fmt"
	"github.com/google/uuid"
	"os"
	"strings"
	"time"
)

func validateConfig() error {
	for i, s := range Config.Service {
		if s.Type == EmbeddedSsh {
			if stats, err := os.Stat(s.Certificate); stats == nil || stats.Size() == 0 || err != nil {
				return fmt.Errorf("config validation ['service[%d].certificate']: embedded ssh needs an existant private key certificate file", i)
			}
		} else if s.Type == EmbeddedWebdav {

		} else if s.Type == PortForward {
			if s.Destination == (Address{}) {
				return fmt.Errorf("config validation ['service[%d].destination']: port forward needs a destination url", i)
			}
		}
	}

	return nil
}

func ProcessString(str string) string {
	if strings.Contains(str, "$(time)") {
		str = strings.ReplaceAll(str, "$(time)", time.Now().Format("2006-01-02-15-04-05"))
	}
	if strings.Contains(str, "$(random)") {
		str = strings.ReplaceAll(str, "$(random)", uuid.New().String())
	}
	if strings.Contains(str, "$(mode)") {
		str = strings.ReplaceAll(str, "$(mode)", Mode)
	}
	return str
}
