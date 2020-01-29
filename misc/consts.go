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
	WdKey              = "WORKING_DIRECTORY"
	EnvVarsKey         = "ENVIRONMENT_VARIABLES"
	RunShPathEnvVar    = "RUNSH_PATH"
	OutDirKey          = "out_dir"
	DebugKey           = "debug"
	PortKey            = "port"
	DismissOutKey      = "dismiss_out"
	TokenKey           = "token"
	VersionKey         = "version"
	Fuse               = "condom"
	QueueLengthKey     = "qlength"
	TimeoutKey         = "timeout"
	RepoOwnerKey       = "repo_owner"
	WebHookSecretKey   = "webhook_secret"
	RepoDirKey         = "repo_dir"
	CertSourceKey      = "certs_source"
	RunDirKey          = "run_dir"
	AvatarDir          = "avatar_dir"
	GitHubClientId     = "github_cid"
	GitHubClientSecret = "github_cs"
	OAuthEndpoint      = "oauth_home_page"
	OAuthAppName       = "oauth_app_name"
	JWTSecret          = "jwt_secret"
)

const (
	TaskPrefix = "tfci-"
	EnvPrefix  = "TFCHEK"
)
const (
	STATICDIR   = "/static/"
	WEBHOOKPATH = "/webhook/"

	PORT           = 8085
	APPNAME        = "tfChek"
	runshchunk     = "runsh/"
	hash512Query   = runshchunk + "by-sha512/"
	APIV1          = "/api/v1/"
	APIRUNSH       = APIV1 + runshchunk
	APIRUNSHIDQ    = APIV1 + hash512Query
	APICANCEL      = APIV1 + "cancel/"
	WEBSOCKETPATH  = "/ws/"
	WSRUNSH        = WEBSOCKETPATH + runshchunk
	WEBHOOKRUNSH   = WEBHOOKPATH + runshchunk
	HEALTHCHECK    = "/health/is_alive"
	READINESSCHECK = "/health/is_ready"
	AVATARS        = "/avatars"
	AUTH           = "/auth"
	AUTHINFO       = "/authinfo/"
)

const NOOUTPUT = "---NO OUTPUT AVAILABLE---"

//TODO: remove "production_42" hardcode
const PROD42 = "production_42"
