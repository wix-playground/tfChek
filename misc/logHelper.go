package misc

import (
	"github.com/spf13/viper"
	"log"
)

func Debug(msg string) {
	if viper.GetBool(DebugKey) {
		log.Print(msg)
	}
}
