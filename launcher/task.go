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
)

type StateError struct {
	msg string
}
type TaskStatus uint8

func (se *StateError) Error() string {
	return se.msg
}

type Task interface {
	Run() error
	GetId() int
	setId(id int)
	GetStdOut() io.Reader
	GetStdErr() io.Reader
	GetStdIn() io.Writer
	GetStatus() TaskStatus
	SetStatus(status TaskStatus)
	SyncName() string
	Schedule() error
	Start() error
	Done() error
	Fail() error
	TimeoutFail() error
}

const (
	OPEN       = iota //Task has been just created
	REGISTERED        //Corresponding webhook arrived to the server
	SCHEDULED         //Task has been accepted to the job queue
	STARTED           //Task has been started
	FAILED            //Task failed
	TIMEOUT           //Task failed to finish in time
	DONE              //Task completed
)

var DEBUG bool = false

func (bti *BackgroundTaskImpl) Register() error {
	if bti.Status == OPEN {
		bti.Status = REGISTERED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled registered, beacuse it is not open. Please make get request. Current state number is %d", bti.Status)}
	}
}

func (bti *BackgroundTaskImpl) Schedule() error {
	if bti.Status == REGISTERED {
		bti.Status = SCHEDULED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled because it has been not registered. Please wait for a webhook. Current state number is %d", bti.Status)}
	}
}

func (bti *BackgroundTaskImpl) Start() error {
	if bti.Status < STARTED {
		if DEBUG {
			log.Printf("Start of task %s", bti.Name)
		}
		bti.Status = STARTED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be started because it is not in scheduled state. Current state number is %d", bti.Status)}
	}
}
func (bti *BackgroundTaskImpl) Done() error {
	if bti.Status == STARTED {
		bti.Status = DONE
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be done, because it has been not started. Current state number is %d", bti.Status)}
	}
}

func (bti *BackgroundTaskImpl) Fail() error {
	if bti.Status == STARTED {
		bti.Status = FAILED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be failed, because it has been not started. Current state number is %d", bti.Status)}
	}
}

func (bti *BackgroundTaskImpl) TimeoutFail() error {
	if bti.Status == STARTED {
		bti.Status = TIMEOUT
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be timed out, because it has been not started. Current state number is %d", bti.Status)}
	}
}

type BackgroundTaskImpl struct {
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
	if bti.Status != SCHEDULED {
		return errors.New("cannot run unscheduled task")
	}
	defer bti.outW.Close()
	defer bti.errW.Close()
	defer bti.inR.Close()
	//Get working directory
	var cwd string
	if d, ok := bti.Context.Value(WD).(string); ok {
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
	if d, ok := bti.Context.Value(ENVVARS).(map[string]string); ok {
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
	command.Stdin = bti.inR

	//I will write nothing to the command
	//So closing stdin immediately
	err := bti.inR.Close()
	if err != nil {
		log.Printf("Cannot close stdin for task id: %d", bti.Id)
	}

	bti.Status = STARTED
	err = command.Run()
	if err != nil {
		if err.Error() == "context deadline exceeded" {
			log.Printf("Command timed out error: %v", err)
			bti.Status = TIMEOUT
		} else {
			log.Printf("Command finished with error: %v", err)
			bti.Status = FAILED
		}
	} else {
		bti.Status = DONE
		log.Println("Command completed successfully")
	}
	return err
}
