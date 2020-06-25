package api

import (
	"github.com/gorilla/mux"
	"net/http"
	"tfChek/misc"
)

func DeleteCIBranch(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	branch := v[misc.ApiBranchKey]

}
