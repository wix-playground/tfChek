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
	OPEN = iota
	SCHEDULED
	STARTED
	FAILED
	TIMEOUT
	DONE
)

var DEBUG bool = false

func (bti *BackgroundTaskImpl) Schedule() error {
	if bti.Status < SCHEDULED {
		bti.Status = SCHEDULED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled because it is not in open state. Current state number is %d", bti.Status)}
	}
}

func (bti *BackgroundTaskImpl) Start() error {
	if bti.Status < STARTED {
		if DEBUG {
			log.Printf("Start of unscheduled task %s", bti.Name)
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
	Name    string
	Id      int
	Command string
	Args    []string
	Context context.Context
	Status  TaskStatus
	Socket  chan *websocket.Conn
	out     io.Reader
	err     io.Reader
	in      io.Writer
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
	outPipeReader, outPipeWriter := io.Pipe()
	errPipeReader, errPipeWriter := io.Pipe()
	inPipeReader, inPipeWriter := io.Pipe()
	bti.out = outPipeReader
	bti.err = errPipeReader
	bti.in = inPipeWriter
	defer outPipeWriter.Close()
	defer errPipeWriter.Close()
	defer inPipeReader.Close()
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
	command.Stdout = outPipeWriter
	command.Stderr = errPipeWriter
	command.Stdin = inPipeReader
	//I will write nothing to the command
	err := inPipeWriter.Close()
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
