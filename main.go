package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"tfChek/api"
	"tfChek/launcher"
)

const (
	STATICDIR = "/static/"
	PORT      = "8085"
)

func setupRoutes() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/ws/runsh/{id}", api.RunShWs).Methods("GET")
	router.Path("/api/v1/runsh/{Env}/{Layer}").Methods("GET").Name("Env/Layer").HandlerFunc(api.ApiRSEL)
	router.PathPrefix(STATICDIR).Handler(http.StripPrefix(STATICDIR, http.FileServer(http.Dir("."+STATICDIR))))
	router.PathPrefix("/").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "./static/index.html")
	})
	return router

}

func main() {
	tm := launcher.GetTaskManager()
	fmt.Println("Starting task manager")
	go tm.Start()
	defer tm.Close()
	fmt.Println("Starting server")
	router := setupRoutes()
	log.Fatal(http.ListenAndServe(":"+PORT, router))
}
