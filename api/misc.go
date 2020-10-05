package api

import (
	"github.com/wix-playground/tfChek/launcher"
	"net/http"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	tm := launcher.GetTaskManager()
	if tm.IsStarted() {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	} else {
		w.WriteHeader(100)
		w.Write([]byte("Is not started"))
	}
}
