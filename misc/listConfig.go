package misc

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"strings"
)

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
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", CertSourceKey, viper.GetString(CertSourceKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", RunDirKey, viper.GetString(RunDirKey)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", AvatarDir, viper.GetString(AvatarDir)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", GitHubClientId, viper.GetString(GitHubClientId)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", GitHubClientSecret, maskPass(viper.GetString(GitHubClientSecret))))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", OAuthAppName, viper.GetString(OAuthAppName)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", JWTSecret, maskPass(viper.GetString(JWTSecret))))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", OAuthEndpoint, viper.GetString(OAuthEndpoint)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", Fuse, viper.GetString(Fuse)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", S3BucketName, viper.GetString(S3BucketName)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", AWSRegion, viper.GetString(S3BucketName)))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", AWSAccessKey, maskPass(viper.GetString(AWSAccessKey))))
	builder.WriteString(fmt.Sprintf("\t%s: %v;\n", AWSSecretKey, maskPass(viper.GetString(AWSSecretKey))))

	if viper.GetBool(DebugKey) {
		log.Printf(builder.String())
	}
}

func maskPass(pass string) string {
	var container []byte = make([]byte, len(pass)*len("*"))
	bp := 0
	for i := 0; i < len(pass); i++ {
		bp += copy(container[bp:], "*")
	}
	return string(container)
}
