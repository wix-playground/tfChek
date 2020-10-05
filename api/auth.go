package api

import (
	"errors"
	"fmt"
	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/token"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/wix-playground/tfChek/misc"
	"net/http"
	"strings"
	"time"
)

func getAuthOptions() *auth.Opts {
	options := auth.Opts{
		SecretReader: token.SecretFunc(func() (string, error) {
			return misc.JWTSecret, nil
		}),
		TokenDuration:  time.Minute * 5,
		CookieDuration: time.Hour * 24,
		Issuer:         viper.GetString(misc.OAuthAppName),
		URL:            viper.GetString(misc.OAuthEndpoint),
		AvatarStore:    avatar.NewLocalFS(viper.GetString(misc.AvatarDir)),
		Validator: token.ValidatorFunc(func(_ string, claims token.Claims) bool {
			//return claims.User != nil && strings.HasPrefix(claims.User.Name, "maksymsh")
			return true
		}),
		Logger:        logger.Std,
		SecureCookies: false,
		DisableXSRF:   true,
	}
	return &options
}

func GetAuthService() *auth.Service {
	service := auth.NewService(*getAuthOptions())
	service.AddProvider("github", viper.GetString(misc.GitHubClientId), viper.GetString(misc.GitHubClientSecret))
	return service
}

func getClientId(provider string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "github":
		return viper.GetString(misc.GitHubClientId), nil
	default:
		return "", errors.New(fmt.Sprintf("no such provider %s", provider))
	}
}

func GetAuthInfoHandler() *AuthInfoHandler {
	h := AuthInfoHandler{HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
		v := mux.Vars(r)
		provider := v["Provider"]
		cid, err := getClientId(provider)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(cid))
		}
	}}
	return &h
}
