package api

import "net/http"

type IndexHandler struct {
	HandlerFunc http.HandlerFunc
}

func (i *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i.HandlerFunc.ServeHTTP(w, r)
}
