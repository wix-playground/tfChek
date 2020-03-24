package misc

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
)

func Debug(msg string) {
	if viper.GetBool(DebugKey) {
		log.Print(msg)
	}
}

func Debugf(format string, args ...interface{}) {
	if viper.GetBool(DebugKey) {
		msg := fmt.Sprintf(format, args)
		Debug(msg)
	}
}
