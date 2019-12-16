package api

import (
	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/token"
	"github.com/spf13/viper"
	"tfChek/misc"
	"time"
)

func getAuthOptions() *auth.Opts {
	options := auth.Opts{
		SecretReader: token.SecretFunc(func() (string, error) {
			return "secret", nil
		}),
		TokenDuration:  time.Minute * 5,
		CookieDuration: time.Hour * 24,
		Issuer:         "tfChek-test",
		URL:            "https://899e26bb.ngrok.io/auth",
		AvatarStore:    avatar.NewLocalFS(viper.GetString(misc.AvatarDir)),
		Validator: token.ValidatorFunc(func(_ string, claims token.Claims) bool {
			//return claims.User != nil && strings.HasPrefix(claims.User.Name, "maksymsh")
			return true
		}),
		Logger:        logger.Std,
		SecureCookies: false,
	}
	return &options
}

func GetAuthService() *auth.Service {
	service := auth.NewService(*getAuthOptions())
	service.AddProvider("github", "6b0d2f7683277927623f", "f7cf93ac7b714aa9286dbf328e0b32fe1c0dafe6")
	return service
}
