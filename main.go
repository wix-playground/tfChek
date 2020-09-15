package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/wix-system/tfChek/api"
	"github.com/wix-system/tfChek/github"
	"github.com/wix-system/tfChek/launcher"
	"github.com/wix-system/tfChek/misc"
	"github.com/wix-system/tfResDif/v3/helpers"
	"log"
	"net/http"
)

const (
	MajorVersion = 0
	MinorVersion = 9
	Revision     = 1
)

func config() {

	//Initialize configuration for wtf first
	helpers.InitViper()
	//Then rewrite it with tfChek keys

	flag.Int(misc.PortKey, misc.PORT, "Port application will listen to")
	flag.Bool(misc.DebugKey, false, "Print debug messages")
	flag.String(misc.OutDirKey, "/var/tfChek/out/", "Directory to save output of the task runs")
	flag.String(misc.TokenKey, "", "GitHub API access token")
	flag.Bool(misc.VersionKey, false, "Show the version info")
	flag.Bool(misc.Fuse, false, "Prevent server from applying run.sh")
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
	viper.SetDefault(misc.CertSourceKey, "")
	viper.SetDefault(misc.RunDirKey, "/var/run/tfChek/")
	viper.SetDefault(misc.AvatarDir, "/var/tfChek/avatars")
	viper.SetDefault(misc.GitHubClientId, "client_id_here")
	viper.SetDefault(misc.GitHubClientSecret, "client_secret_here")
	viper.SetDefault(misc.OAuthAppName, misc.APPNAME)
	viper.SetDefault(misc.OAuthEndpoint, "https://bo.wixpress.com/tfchek")
	viper.SetDefault(misc.JWTSecret, "secret")
	viper.SetDefault(misc.S3BucketName, "wix-terraform-ci")
	viper.SetDefault(misc.AWSRegion, "us-east-1")
	viper.SetDefault(misc.AWSAccessKey, "") //Configures your AWS access key
	viper.SetDefault(misc.AWSSecretKey, "") //Configures your AWS secret key
	viper.SetDefault(misc.AWSSequenceTable, "tfChek-sequence")
	viper.SetDefault(misc.UseExternalSequence, true)
	viper.SetDefault(misc.WebhookWaitTimeoutKey, 180)
	viper.SetDefault(misc.SkipPullFastForward, true) //TODO: set it to false when wtf is ready for fast forward pull of the branch
	viper.SetDefault(misc.GitHubDownload, true)
	viper.SetEnvPrefix(misc.EnvPrefix)
	viper.AutomaticEnv()
	viper.SetConfigName(misc.APPNAME)
	viper.AddConfigPath("/configs")
	viper.AddConfigPath("/opt/wix/" + misc.APPNAME + "/etc/")
	viper.AddConfigPath("/etc/" + misc.APPNAME)
	viper.AddConfigPath("$HOME/." + misc.APPNAME)
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Printf("Cannot read configuration. Error: %s", err)
	} else {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Configuration has been loaded")
		}
	}

}

func setupRoutes() *mux.Router {
	authService := api.GetAuthService()
	middleware := authService.Middleware()
	authRoutes, avatarRoutes := authService.Handlers()
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc(misc.WSRUNSH+api.FormatIdParam(), api.RunShWebsocket).Name("Websocket").Methods(http.MethodGet)
	router.Path(misc.APIRUNSHIDQ + "{Hash}").Methods(http.MethodGet).Name("Query by hash").HandlerFunc(api.GetTaskIdByHash)
	router.Path(misc.APIRUNSH).Methods(http.MethodPost).Name("run.sh universal task accepting endpoint").HandlerFunc(api.RunShPost)
	router.Path(misc.API2RUNSH).Methods(http.MethodPost).Name("run.sh universal task accepting endpoint").HandlerFunc(api.RunShPost)
	router.Path(misc.APIWTF).Methods(http.MethodPost).Name("wtf task accepting endpoint").HandlerFunc(api.WtfPost)
	router.Path(misc.APICANCEL + api.FormatIdParam()).Methods(http.MethodGet).Name("Cancel").HandlerFunc(api.Cancel)
	router.Path(misc.APIDELETEBRANCH + "{id}").Methods(http.MethodDelete).Name("DeleteBranch").HandlerFunc(api.DeleteCIBranch)
	router.Path(misc.APICLEANUPBRANCH).Methods(http.MethodPost).Name("Clean-up branches").HandlerFunc(api.Cleanupbranches)
	router.Path(misc.WEBHOOKRUNSH).Methods(http.MethodPost).Name("GitHub web hook").HandlerFunc(api.RunShWebHook)

	router.Path(misc.HEALTHCHECK).HandlerFunc(api.HealthCheck)
	router.Path(misc.AUTHINFO + "{Provider}").Name("Authentication info endpoint").Methods(http.MethodGet).Handler(api.GetAuthInfoHandler())
	router.PathPrefix(misc.AVATARS).Name("Avatars").Handler(avatarRoutes)
	router.PathPrefix(misc.AUTH).Name("Authentication endpoint").Handler(authRoutes)
	router.Path(misc.READINESSCHECK).HandlerFunc(api.ReadinessCheck)
	router.Path("/login").Methods(http.MethodGet).Name("Login").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+misc.STATICDIR+"login.html")
	})
	router.Path(misc.STATICDIR + "script/auth_provider.js").Methods(http.MethodGet).Name("Login Script").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+misc.STATICDIR+"script/auth_provider.js")
	})
	router.Path(misc.STATICDIR + "css/main.css").Methods(http.MethodGet).Name("CSS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+misc.STATICDIR+"css/main.css")
	})
	router.PathPrefix(misc.STATICDIR + "pictures").Name("Pictures").Methods(http.MethodGet).Handler(http.StripPrefix(misc.STATICDIR+"pictures", http.FileServer(http.Dir("."+misc.STATICDIR+"pictures"))))
	router.Path("/favicon.ico").Name("Icon").Methods(http.MethodGet).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+misc.STATICDIR+"pictures/tfChek_logo.ico")
	})
	router.PathPrefix(misc.STATICDIR).Handler(middleware.Auth(http.StripPrefix(misc.STATICDIR, http.FileServer(http.Dir("."+misc.STATICDIR)))))
	router.Path("/").Handler(&api.IndexHandler{
		HandlerFunc: func(writer http.ResponseWriter, request *http.Request) {
			http.ServeFile(writer, request, "."+misc.STATICDIR+"index.html")
		},
	})

	return router

}

func initialize() {
	//Prepare configuration
	config()

	if viper.GetBool(misc.DebugKey) {
		misc.LogConfig()
	}
	//Start task manager
	tm := launcher.GetTaskManager()
	fmt.Println("Starting task manager")
	go tm.Start()
}

func showVersion() {
	fmt.Print(getVersion())
}

func getVersion() string {
	return fmt.Sprintf("%d.%d.%d", MajorVersion, MinorVersion, Revision)
}

func main() {
	log.Printf("Starting tfChek version: %s", getVersion())
	initialize()
	defer launcher.GetTaskManager().Close()
	defer github.CloseAll()
	fmt.Println("Starting server")
	router := setupRoutes()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt(misc.PortKey)), router))
}
