package launcher

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"strconv"
	"strings"
	"tfChek/misc"
	"time"
)

type StateError struct {
	msg string
}
type TaskStatus uint8

func (se *StateError) Error() string {
	return se.msg
}

type GitHubAwareTask interface {
	Task
	GetOrigins() *[]string
	SetAuthors(authors []string)
	GetAuthors() *[]string
	AddWebhookLocks() error
	UnlockWebhookRepoLock(fullName string) error
}

type RunSHOptions struct {
	Timeout        string
	YN             string
	All            string
	UsePlan        string
	OmitGitCheck   string
	Filter         string
	Region         string
	Debug          string
	UpgradeVersion string
	Location       string
	Targets        string
}

type RunSHLaunchConfig struct {
	RepoSources    []string
	FullCommand    string
	CommandOptions *RunSHOptions
	Instant        int64
}

func (rc *RunSHLaunchConfig) GetHashedCommand(hash string) (*RunShCmd, error) {
	cmd, err := rc.GetCommand()
	if err != nil {
		return nil, err
	}
	cmd.hash = hash
	return cmd, nil
}

func (rc *RunSHLaunchConfig) GetCommand() (*RunShCmd, error) {
	var cmd RunShCmd
	location := strings.TrimSpace(rc.CommandOptions.Location)
	if location == "" {
		return nil, errors.New("given location cannot be empty")
	}
	el := strings.Split(location, "/")
	env := el[0]
	if len(el) > 2 {
		return nil, errors.New(fmt.Sprintf("Cannot parse environment and layer '%s'. Too many slashes", location))
	}
	layer := ""
	if len(el) == 2 {
		layer = el[1]
	}
	all := strings.ToLower(strings.TrimSpace(rc.CommandOptions.All)) == "y"
	tgts := strings.Split(strings.ToLower(strings.TrimSpace(rc.CommandOptions.Targets)), " ")
	yes := strings.ToLower(strings.TrimSpace(rc.CommandOptions.YN)) == "y"
	no := strings.ToLower(strings.TrimSpace(rc.CommandOptions.YN)) == "n"
	omit := strings.ToLower(strings.TrimSpace(rc.CommandOptions.OmitGitCheck)) == "1"
	usePlan := strings.ToLower(strings.TrimSpace(rc.CommandOptions.UsePlan)) == "n"
	filter := strings.TrimSpace(rc.CommandOptions.Filter)
	debug := strings.ToLower(strings.TrimSpace(rc.CommandOptions.Debug)) == "true"
	region := strings.TrimSpace(rc.CommandOptions.Region)
	terraform := strings.TrimSpace(rc.CommandOptions.UpgradeVersion)

	//Check fuse (condom) option
	if viper.GetBool(misc.Fuse) {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("forcefully disabling applying ability due to '%s' option is set to true", misc.Fuse)
		}
		no = true
		yes = false
	}
	if viper.GetBool(misc.SkipPullFastForward) {
		misc.Debugf("forcefully setting git omit option due to %q option is set to true", misc.SkipPullFastForward)
		omit = true
	}
	gorigins := normalizeGitRemotes(&rc.RepoSources)
	startTime := time.Unix(rc.Instant, 0)
	cmd = RunShCmd{Layer: layer, Env: env, All: all, Omit: omit,
		UsePlan: usePlan, Filter: filter, Region: region, TerraformVersion: terraform,
		Targets: tgts, No: no, Yes: yes, Debug: debug, GitOrigins: *gorigins, Started: &startTime}
	return &cmd, nil
}

func (rc *RunSHLaunchConfig) GetTimeout() time.Duration {
	timeout := time.Duration(viper.GetInt(misc.TimeoutKey)) * time.Second
	if rc.CommandOptions.Timeout == "" {
		return timeout
	} else {
		t, err := strconv.Atoi(rc.CommandOptions.Timeout)
		if err != nil {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Cannot parse timeout %s. Using default value from confguration file %d", rc.CommandOptions.Timeout, viper.GetInt(misc.TimeoutKey))
			}
			return timeout
		}
		return time.Duration(t) * time.Second
	}
}

type Task interface {
	Run() error
	GetId() int
	setId(id int)
	Subscribe() chan TaskStatus
	GetStdOut() io.Reader
	GetCleanOut() string
	GetStdErr() io.Reader
	GetStdIn() io.Writer
	GetStatus() TaskStatus
	SetStatus(status TaskStatus)
	SyncName() string
	Schedule() error
	Start() error
	Done() error
	Fail() error
	ForceFail()
	TimeoutFail() error
}

func GetStatusString(status TaskStatus) string {
	switch status {
	case misc.OPEN:
		return "open"
	case misc.REGISTERED:
		return "registered"
	case misc.SCHEDULED:
		return "scheduled"
	case misc.STARTED:
		return "Started"
	case misc.FAILED:
		return "failed"
	case misc.TIMEOUT:
		return "timeout"
	case misc.DONE:
		return "done"
	default:
		return "unknown"
	}
}
