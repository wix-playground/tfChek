package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"os"
	"os/exec"
	"tfChek/git"
	"tfChek/misc"
	"tfChek/storer"
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

var DEBUG bool = false

func (bti *BackgroundTaskImpl) Register() error {
	if bti.Status == misc.OPEN {
		bti.Status = misc.REGISTERED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled registered, beacuse it is not open. Please make get request. Current state number is %d", bti.Status)}
	}
}

func (bti *BackgroundTaskImpl) Schedule() error {
	if bti.Status == misc.REGISTERED {
		bti.Status = misc.SCHEDULED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled because it has been not registered. Please wait for a webhook. Current state number is %d", bti.Status)}
	}
}

func (bti *BackgroundTaskImpl) Start() error {
	if bti.Status < misc.STARTED {
		if DEBUG {
			log.Printf("Start of task %s", bti.Name)
		}
		bti.Status = misc.STARTED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be started because it is not in scheduled state. Current state number is %d", bti.Status)}
	}
}
func (bti *BackgroundTaskImpl) Done() error {
	if bti.Status == misc.STARTED {
		bti.Status = misc.DONE
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be done, because it has been not started. Current state number is %d", bti.Status)}
	}
}

func (bti *BackgroundTaskImpl) Fail() error {
	if bti.Status == misc.STARTED {
		bti.Status = misc.FAILED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be failed, because it has been not started. Current state number is %d", bti.Status)}
	}
}

func (bti *BackgroundTaskImpl) TimeoutFail() error {
	if bti.Status == misc.STARTED {
		bti.Status = misc.TIMEOUT
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be timed out, because it has been not started. Current state number is %d", bti.Status)}
	}
}

type BackgroundTaskImpl struct {
	Task
	Name       string
	Id         int
	Command    string
	Args       []string
	Context    context.Context
	Status     TaskStatus
	Socket     chan *websocket.Conn
	out, err   io.Reader
	in         io.Writer
	inR        io.ReadCloser
	outW, errW io.WriteCloser
	save       bool
}

func (bti *BackgroundTaskImpl) GetStatus() TaskStatus {
	return bti.Status
}

func (bti *BackgroundTaskImpl) SetStatus(status TaskStatus) {
	bti.Status = status
}

func (bti *BackgroundTaskImpl) GetId() int {
	return bti.Id
}

func (bti *BackgroundTaskImpl) setId(id int) {
	bti.Id = id
}

func (bti *BackgroundTaskImpl) SyncName() string {
	return bti.Command
}

func (bti *BackgroundTaskImpl) GetStdOut() io.Reader {
	return bti.out
}

func (bti *BackgroundTaskImpl) GetStdErr() io.Reader {
	return bti.err
}

func (bti *BackgroundTaskImpl) GetStdIn() io.Writer {
	return bti.in
}

func (bti *BackgroundTaskImpl) Run() error {
	if bti.Status != misc.SCHEDULED {
		return errors.New("cannot run unscheduled task")
	}
	defer bti.outW.Close()
	defer bti.errW.Close()
	defer bti.inR.Close()
	//Get working directory
	var cwd string
	if d, ok := bti.Context.Value(misc.WD).(string); ok {
		cwd = d
	} else {
		d, err := os.Getwd()
		if err != nil {
			return err
		}
		cwd = d
	}
	log.Printf("Task id: %d working directory: %s", bti.Id, cwd)
	//Get environment
	sysenv := os.Environ()
	if d, ok := bti.Context.Value(misc.ENVVARS).(map[string]string); ok {
		for k, v := range d {
			sysenv = append(sysenv, fmt.Sprintf("%s=%s", k, v))
		}
	}
	log.Printf("Task id: %d environment: %s", bti.Id, sysenv)

	command := exec.CommandContext(bti.Context, bti.Command, bti.Args...)
	command.Dir = cwd
	command.Env = sysenv
	log.Printf("Running command and waiting for it to finish...")
	command.Stdout = bti.outW
	command.Stderr = bti.errW
	//command.Stdin = bti.inR
	command.Stdin = nil
	//Ugly but I did not found a better place
	if bti.save {
		out, err := storer.Save2FileFromWriter(bti.Id)
		if err != nil {
			log.Printf("Save to file for task %d is disabled. Error: %s", bti.Id, err)
		} else {
			ow := io.MultiWriter(bti.outW, out)
			eow := io.MultiWriter(bti.errW, out)
			command.Stdout = ow
			command.Stderr = eow
		}
	}

	//I will write nothing to the command
	//So closing stdin immediately
	err := bti.inR.Close()
	if err != nil {
		log.Printf("Cannot close stdin for task id: %d", bti.Id)
	}

	bti.Status = misc.STARTED
	err = command.Run()
	if err != nil {
		if err.Error() == "context deadline exceeded" {
			log.Printf("Command timed out error: %v", err)
			bti.Status = misc.TIMEOUT
		} else {
			log.Printf("Command finished with error: %v", err)
			bti.Status = misc.FAILED
		}
	} else {
		bti.Status = misc.DONE
		log.Println("Command completed successfully")
	}
	return err
}
