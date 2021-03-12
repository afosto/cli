package logging

import (
	log "github.com/sirupsen/logrus"
	"os"
)

var (
	Log = log.New()
)

func init() {
	Log.SetOutput(os.Stdout)
	Log.SetLevel(log.DebugLevel)
	Log.SetFormatter(&log.TextFormatter{
		ForceColors:      true,
		DisableColors:    false,
		DisableTimestamp: true,
	})
}
