package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"tfChek/api"
	"tfChek/launcher"
)

const (
	STATICDIR = "/static/"
	PORT      = "8085"
	APPNAME   = "tfChek"
)

func config() {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Cannot get working directory. Error: %s", err)
		wd = "."
	}
	flag.Int("port", 8085, "Port application will listen to")
	flag.Bool("debug", false, "Print debug messages")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
	viper.SetDefault("working_direrctory", wd)
	viper.SetDefault("debug", false)
	viper.SetEnvPrefix("TFCHEK")
	viper.AutomaticEnv()
	viper.SetConfigName(APPNAME)
	viper.AddConfigPath("/etc/" + APPNAME)
	viper.AddConfigPath("$HOME/." + APPNAME)
	viper.AddConfigPath(".")
	viper.ReadInConfig()
	if err != nil {
		log.Printf("Cannot read configuration. Error: %s", err)
	}

}

func setupRoutes() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/ws/runsh/{id}", api.RunShWebsocket).Methods("GET")
	router.Path("/api/v1/runsh/{Env}/{Layer}").Methods("GET").Name("Env/Layer").HandlerFunc(api.RunShEnvLayer)
	router.Path("/api/v1/runsh/{Env}").Methods("GET").Name("Env").HandlerFunc(api.RunShEnv)
	router.PathPrefix(STATICDIR).Handler(http.StripPrefix(STATICDIR, http.FileServer(http.Dir("."+STATICDIR))))
	router.PathPrefix("/").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "./static/index.html")
	})
	return router

}

func main() {
	config()
	tm := launcher.GetTaskManager()
	fmt.Println("Starting task manager")
	go tm.Start()
	defer tm.Close()
	fmt.Println("Starting server")
	router := setupRoutes()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", viper.Get("port")), router))
}
