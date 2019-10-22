package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"tfChek/api"
	"tfChek/github"
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
	APICANCEL      = APIV1 + "cancel/"
	WEBSOCKETPATH  = "/ws/"
	WSRUNSH        = WEBSOCKETPATH + runshchunk
	WEBHOOKRUNSH   = WEBHOOKPATH + runshchunk
	HEALTHCHECK    = "/health/is_alive"
	READINESSCHECK = "/health/is_ready"
)

func config() {
	flag.Int("port", PORT, "Port application will listen to")
	flag.Bool("debug", false, "Print debug messages")
	flag.String("outdir", "out", "Directory to save output of the task runs")
	flag.Bool("save", true, "Save tasks output to the files in outdir")
	flag.String("token", "", "GitHub API access token")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
	viper.SetDefault("debug", false)
	viper.SetDefault("qlength", 10)
	viper.SetDefault("timeout", 300)
	viper.SetDefault("repo_owner", "wix-system")
	viper.SetDefault("webhook_secret", "notAsecretAtAll:)")
	viper.SetDefault("repo_dir", "/var/tfChek/repos_by_state/")
	viper.SetDefault("repo_name", "production_42")
	viper.SetEnvPrefix("TFCHEK")
	viper.AutomaticEnv()
	viper.SetConfigName(APPNAME)
	viper.AddConfigPath("/opt/wix/" + APPNAME + "/etc/")
	viper.AddConfigPath("/configs/" + APPNAME)
	viper.AddConfigPath("$HOME/." + APPNAME)
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Cannot read configuration. Error: %s", err)
	}

}

func setupRoutes() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc(WSRUNSH+"{Id}", api.RunShWebsocket).Name("Websocket").Methods("GET")
	router.Path(APIRUNSH + "{Env}/{Layer}").Methods("GET").Name("Env/Layer").HandlerFunc(api.RunShEnvLayer)
	router.Path(APIRUNSH + "{Env}").Methods("GET").Name("Env").HandlerFunc(api.RunShEnv)
	router.Path(APICANCEL + "{Id}").Methods("GET").Name("Cancel").HandlerFunc(api.Cancel)
	router.Path(WEBHOOKRUNSH).Methods("POST").Name("GitHub web hook").HandlerFunc(api.RunShWebHook)
	router.PathPrefix(STATICDIR).Handler(http.StripPrefix(STATICDIR, http.FileServer(http.Dir("."+STATICDIR))))
	router.Path(HEALTHCHECK).HandlerFunc(api.HealthCheck)
	router.Path(READINESSCHECK).HandlerFunc(api.ReadinessCheck)
	router.PathPrefix("/").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "./static/index.html")
	})
	return router

}

func initialize() {
	//Prepare configuration
	config()
	//Start GitHub API manager

	repoName := viper.GetString("repo_name")
	repoOwner := viper.GetString("repo_owner")
	token := viper.GetString("token")
	github.InitManager(repoName, repoOwner, token)
	github.GetManager().Start()
	//Start task manager
	tm := launcher.GetTaskManager()
	fmt.Println("Starting task manager")
	go tm.Start()
}

func main() {
	initialize()
	defer launcher.GetTaskManager().Close()
	defer github.GetManager().Close()
	fmt.Println("Starting server")
	router := setupRoutes()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("port")), router))
}
