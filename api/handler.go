package api

import "net/http"

type IndexHandler struct {
	HandlerFunc http.HandlerFunc
}

func (i *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i.HandlerFunc(w, r)
}

type AuthInfoHandler struct {
	HandlerFunc http.HandlerFunc
}

func (a *AuthInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.HandlerFunc(w, r)
}
