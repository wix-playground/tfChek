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

const (
	MajorVersion = 0
	MinorVersion = 4
	Revision     = 9
)

func config() {
	flag.Int(misc.PortKey, misc.PORT, "Port application will listen to")
	flag.Bool(misc.DebugKey, false, "Print debug messages")
	flag.String(misc.OutDirKey, "/var/tfChek/out/", "Directory to save output of the task runs")
	flag.Bool(misc.DismissOutKey, true, "Save tasks output to the files in outdir")
	flag.String(misc.TokenKey, "", "GitHub API access token")
	flag.Bool(misc.VersionKey, false, "Show the version info")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatalf("Cannot bind flags. Error: %s", err)
	}
	viper.SetDefault(misc.QueueLengthKey, 10)
	viper.SetDefault(misc.TimeoutKey, 300)
	viper.SetDefault(misc.RepoOwnerKey, "wix-system")
	viper.SetDefault(misc.WebHookSecretKey, "notAsecretAtAll:)")
	viper.SetDefault(misc.RepoDirKey, "/var/tfChek/repos_by_state/")
	viper.SetDefault(misc.RepoNameKey, "production_42")
	viper.SetDefault(misc.RunDirKey, "/var/run/tfChek/")
	viper.SetEnvPrefix(misc.EnvPrefix)
	viper.AutomaticEnv()
	viper.SetConfigName(misc.APPNAME)
	viper.AddConfigPath("/opt/wix/" + misc.APPNAME + "/etc/")
	viper.AddConfigPath("/configs/" + misc.APPNAME)
	viper.AddConfigPath("$HOME/." + misc.APPNAME)
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
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
		//Debug websocket
		//log.Printf("Request %s headers:", request.URL.String())
		//for k, v := range request.Header {
		//	log.Printf("\tHeader field %q, Value %q\n", k, v)
		//}
		//End debug
		http.ServeFile(writer, request, "."+misc.STATICDIR+"index.html")
	})
	return router

}

func initialize() {
	//Prepare configuration
	config()
	//Start GitHub API manager

	repoName := viper.GetString(misc.RepoNameKey)
	repoOwner := viper.GetString(misc.RepoOwnerKey)
	token := viper.GetString(misc.TokenKey)
	if viper.GetBool(misc.DebugKey) {
		misc.Debug = true
		misc.LogConfig()
	}
	github.InitManager(repoName, repoOwner, token)
	github.GetManager().Start()
	//Start task manager
	tm := launcher.GetTaskManager()
	fmt.Println("Starting task manager")
	go tm.Start()
}

func showVersion() {
	fmt.Printf("%d.%d.%d", MajorVersion, MinorVersion, Revision)
}

func main() {
	initialize()
	defer launcher.GetTaskManager().Close()
	defer github.GetManager().Close()
	fmt.Println("Starting server")
	router := setupRoutes()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt(misc.PortKey)), router))
}
