package launcher

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"strconv"
	"strings"
	"tfChek/git"
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
	SetGitManager(manager git.Manager)
	GetOrigins() *[]string
	SetAuthors(authors []string)
	GetAuthors() *[]string
}

type RunSHOptions struct {
	Timeout        string
	YN             string
	All            string
	UsePlan        string
	OmitGitCheck   string
	Filter         string
	Region         string
	UpgradeVersion string
	Section        string
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
	location := rc.CommandOptions.Location
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
	//Check fuse (condom) option
	if viper.GetBool(misc.Fuse) {
		if DEBUG {
			log.Print("forcefully disabling applying ability dues to '%s' option is set to true", misc.Fuse)
		}
		no = true
		yes = false
	}

	//TODO: add support of all options
	startTime := time.Unix(rc.Instant, 0)
	cmd = RunShCmd{Layer: layer, Env: env, All: all, Omit: omit, Targets: tgts, No: no, Yes: yes, Started: &startTime}
	return &cmd, nil
}

func (rc *RunSHLaunchConfig) GetTimeout() time.Duration {
	timeout := time.Duration(viper.GetInt(misc.TimeoutKey)) * time.Second
	if rc.CommandOptions.Timeout == "" {
		return timeout
	} else {
		t, err := strconv.Atoi(rc.CommandOptions.Timeout)
		if err != nil {
			if DEBUG {
				log.Printf("Cannot parse timeout %s. Using default value from confguration file %s", rc.CommandOptions.Timeout, viper.GetInt(misc.TimeoutKey))
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
	GetStdOut() io.Reader
	GetOut() string
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

var DEBUG bool = viper.GetBool(misc.DebugKey)
