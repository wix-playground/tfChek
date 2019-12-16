package launcher

import (
	"github.com/spf13/viper"
	"io"
	"tfChek/git"
	"tfChek/misc"
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
	SetAuthors(authors []string)
	GetAuthors() *[]string
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
		return "started"
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
