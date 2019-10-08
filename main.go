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
	STATICDIR      = "/static/"
	WEBHOOKPATH    = "/webhook/"
	PORT           = 8085
	APPNAME        = "tfChek"
	runshchunk     = "runsh/"
	APIV1          = "/api/v1/"
	APIRUNSH       = APIV1 + runshchunk
	WEBSOCKETPATH  = "/ws/"
	WSRUNSH        = WEBSOCKETPATH + runshchunk
	WEBHOOKRUNSH   = WEBHOOKPATH + runshchunk
	HEALTHCHECK    = "/health/is_alive"
	READINESSCHECK = "/health/is_ready"
)

func config() {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Cannot get working directory. Error: %s", err)
		wd = "."
	}
	flag.Int("port", PORT, "Port application will listen to")
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
	router.HandleFunc(WSRUNSH+"{id}", api.RunShWebsocket).Name("Websocket").Methods("GET")
	router.Path(APIRUNSH + "{Env}/{Layer}").Methods("GET").Name("Env/Layer").HandlerFunc(api.RunShEnvLayer)
	router.Path(APIRUNSH + "{Env}").Methods("GET").Name("Env").HandlerFunc(api.RunShEnv)
	router.Path(WEBHOOKRUNSH).Methods("POST").Name("GitHub web hook").HandlerFunc(api.RunShWebHook)
	router.PathPrefix(STATICDIR).Handler(http.StripPrefix(STATICDIR, http.FileServer(http.Dir("."+STATICDIR))))
	router.Path(HEALTHCHECK).HandlerFunc(api.HealthCheck)
	router.Path(READINESSCHECK).HandlerFunc(api.ReadinessCheck)
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
