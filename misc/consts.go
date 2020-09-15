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
	AwsAccessKeyVar    = "AWS_ACCESS_KEY_ID"
	AwsSecretKeyVar    = "AWS_SECRET_ACCESS_KEY"
	NotifyTfChekEnvVar = "NOTIFY_TFCHEK"
	OutDirKey          = "out_dir"
	DebugKey           = "debug"
	PortKey            = "port"
	//Deprecated
	DismissOutKey         = "dismiss_out"
	TokenKey              = "token"
	VersionKey            = "version"
	Fuse                  = "condom"
	SkipPullFastForward   = "omit_git_pull"
	QueueLengthKey        = "qlength"
	TimeoutKey            = "timeout"
	RepoOwnerKey          = "repo_owner"
	WebHookSecretKey      = "webhook_secret"
	RepoDirKey            = "repo_dir"
	RepomanagerPathSuffix = "from_github_api"
	GitmanagerPathSuffix  = "from_git"
	CertSourceKey         = "certs_source"
	LambdaSourceKey       = "lambdas_source"
	RunDirKey             = "run_dir"
	RunshExe              = "run.sh"
	WtfExe                = "wtf"
	GitHubDownload        = "github_download"
	AvatarDir             = "avatar_dir"
	GitHubClientId        = "github_cid"
	GitHubClientSecret    = "github_cs"
	OAuthEndpoint         = "oauth_home_page"
	OAuthAppName          = "oauth_app_name"
	JWTSecret             = "jwt_secret"
	S3BucketName          = "aws_s3_bucket_name"
	AWSRegion             = "aws_region"
	AWSAccessKey          = "aws_access_key"
	AWSSecretKey          = "aws_secret_key"
	AWSSequenceTable      = "aws_sequence_table"
	UseExternalSequence   = "use_external_sequence"
	WebhookWaitTimeoutKey = "webhook_timeout"
	GitSectionRemote      = "remote"
	GitSectionBranch      = "branch"
	GitSectionOptionFetch = "fetch"
	GitSectionOptionMerge = "merge"
	ApiHashKey            = "Hash"
	ApiBranchKey          = "branch"
	IssueLabel            = APPNAME
	IssueLabelDesc        = "tfChek managed issue"
	IssueAllFilter        = "all"
	IdParam               = "id"
	ApiMergeKey           = "merged"
	ApiBeforeKey          = "before"
	ContentTypeKey        = "Content-Type"
	ContentTypeJson       = "application/json"
	ContentTypeMarkdown   = "text/markdown"
)

const (
	TaskPrefix = "tfci-"
	EnvPrefix  = "TFCHEK"
)
const (
	STATICDIR   = "/static/"
	WEBHOOKPATH = "/webhook/"

	PORT             = 8085
	APPNAME          = "tfChek"
	runshchunk       = "runsh/"
	wtfchunk         = "wtf/"
	hash512Query     = runshchunk + "by-sha512/"
	APIV1            = "/api/v1/"
	APIV2            = "/api/v2/"
	APIRUNSH         = APIV1 + runshchunk
	APIRUNSHIDQ      = APIV1 + hash512Query
	APICANCEL        = APIV1 + "cancel/"
	APIDELETEBRANCH  = APIV2 + "delete/branch/"
	APICLEANUPBRANCH = APIV2 + "cleanup"
	APIWTF           = APIV2 + wtfchunk
	API2RUNSH        = APIV2 + runshchunk
	WEBSOCKETPATH    = "/ws/"
	WSRUNSH          = WEBSOCKETPATH + runshchunk
	WEBHOOKRUNSH     = WEBHOOKPATH + runshchunk
	HEALTHCHECK      = "/health/is_alive"
	READINESSCHECK   = "/health/is_ready"
	AVATARS          = "/avatars"
	AUTH             = "/auth"
	AUTHINFO         = "/authinfo/"
)

const NOOUTPUT = "---NO OUTPUT AVAILABLE---"

//TODO: remove "production_42" hardcode
const PROD42 = "production_42"
const RG = "rg"
