package misc

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"strings"
)

var Debug = false

func LogConfig() {
	builder := strings.Builder{}
	builder.WriteString("Loaded Configuration:\n")
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", PortKey, viper.GetInt(PortKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", DebugKey, viper.GetBool(DebugKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", DismissOutKey, viper.GetBool(DismissOutKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", OutDirKey, viper.GetString(OutDirKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", TokenKey, viper.GetString(TokenKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", VersionKey, viper.GetBool(VersionKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", QueueLengthKey, viper.GetInt(QueueLengthKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", TimeoutKey, viper.GetInt(TimeoutKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", RepoOwnerKey, viper.GetString(RepoOwnerKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", RepoDirKey, viper.GetString(RepoDirKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", RepoNameKey, viper.GetString(RepoNameKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", RunDirKey, viper.GetString(RunDirKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", AvatarDir, viper.GetString(AvatarDir)))
	if Debug {
		log.Printf(builder.String())
	}
}
