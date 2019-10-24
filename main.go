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
	"tfChek/misc"
)

func config() {
	flag.Int("port", misc.PORT, "Port application will listen to")
	flag.Bool("debug", false, "Print debug messages")
	flag.String("out_dir", "/var/tfChek/out/", "Directory to save output of the task runs")
	flag.Bool("dismiss_out", true, "Save tasks output to the files in outdir")
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
	viper.SetDefault("run_dir", "/var/run/tfChek/")
	viper.SetEnvPrefix("TFCHEK")
	viper.AutomaticEnv()
	viper.SetConfigName(misc.APPNAME)
	viper.AddConfigPath("/opt/wix/" + misc.APPNAME + "/etc/")
	viper.AddConfigPath("/configs/" + misc.APPNAME)
	viper.AddConfigPath("$HOME/." + misc.APPNAME)
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Cannot read configuration. Error: %s", err)
	}

}

func setupRoutes() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc(misc.WSRUNSH+"{Id}", api.RunShWebsocket).Name("Websocket").Methods("GET")
	router.Path(misc.APIRUNSH + "{Env}/{Layer}").Methods("GET").Name("Env/Layer").HandlerFunc(api.RunShEnvLayer)
	router.Path(misc.APIRUNSH + "{Env}").Methods("GET").Name("Env").HandlerFunc(api.RunShEnv)
	router.Path(misc.APICANCEL + "{Id}").Methods("GET").Name("Cancel").HandlerFunc(api.Cancel)
	router.Path(misc.WEBHOOKRUNSH).Methods("POST").Name("GitHub web hook").HandlerFunc(api.RunShWebHook)
	router.PathPrefix(misc.STATICDIR).Handler(http.StripPrefix(misc.STATICDIR, http.FileServer(http.Dir("."+misc.STATICDIR))))
	router.Path(misc.HEALTHCHECK).HandlerFunc(api.HealthCheck)
	router.Path(misc.READINESSCHECK).HandlerFunc(api.ReadinessCheck)
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
