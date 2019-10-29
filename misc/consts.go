package misc

const (
	OPEN       = iota //Task has been just created
	REGISTERED        //Corresponding webhook arrived to the server
	SCHEDULED         //Task has been accepted to the job queue
	STARTED           //Task has been started
	FAILED            //Task failed
	TIMEOUT           //Task failed to finish in time
	DONE              //Task completed
)

const (
	WdKey            = "WORKING_DIRECTORY"
	EnvVarsKey       = "ENVIRONMENT_VARIABLES"
	OutDirKey        = "out_dir"
	DebugKey         = "debug"
	PortKey          = "port"
	DismissOutKey    = "dismiss_out"
	TokenKey         = "token"
	VersionKey       = "version"
	QueueLengthKey   = "qlength"
	TimeoutKey       = "timeout"
	RepoOwnerKey     = "repo_owner"
	WebHookSecretKey = "webhook_secret"
	RepoDirKey       = "repo_dir"
	RepoNameKey      = "repo_name"
	RunDirKey        = "run_dir"
)

const (
	TaskPrefix = "tfci-"
	EnvPrefix  = "TFCHEK"
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

const NOOUTPUT = "---NO OUTPUT AVAILABLE---"

//TODO: remove "production_42" hardcode
const PROD42 = "production_42"