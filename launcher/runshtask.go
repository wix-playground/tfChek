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
	"tfChek/storer"
)

func (rst *RunShTask) Register() error {
	if rst.Status == OPEN {
		rst.Status = REGISTERED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled registered, beacuse it is not open. Please make get request. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) Schedule() error {
	if rst.Status == REGISTERED {
		rst.Status = SCHEDULED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled because it has been not registered. Please wait for a webhook. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) Start() error {
	if rst.Status < STARTED {
		if DEBUG {
			log.Printf("Start of task %s", rst.Name)
		}
		rst.Status = STARTED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be started because it is not in scheduled state. Current state number is %d", rst.Status)}
	}
}
func (rst *RunShTask) Done() error {
	if rst.Status == STARTED {
		rst.Status = DONE
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be done, because it has been not started. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) Fail() error {
	if rst.Status == STARTED {
		rst.Status = FAILED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be failed, because it has been not started. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) TimeoutFail() error {
	if rst.Status == STARTED {
		rst.Status = TIMEOUT
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be timed out, because it has been not started. Current state number is %d", rst.Status)}
	}
}

type RunShTask struct {
	Name       string
	Id         int
	Command    string
	Args       []string
	StateLock  string
	Context    context.Context
	Status     TaskStatus
	Socket     chan *websocket.Conn
	out, err   io.Reader
	in         io.Writer
	inR        io.ReadCloser
	outW, errW io.WriteCloser
	save       bool
}

func (rst *RunShTask) GetStatus() TaskStatus {
	return rst.Status
}

func (rst *RunShTask) SetStatus(status TaskStatus) {
	rst.Status = status
}

func (rst *RunShTask) GetId() int {
	return rst.Id
}

func (rst *RunShTask) setId(id int) {
	rst.Id = id
}

func (rst *RunShTask) SyncName() string {
	if rst.StateLock != "" {
		return rst.StateLock
	}
	return rst.Command
}

func (rst *RunShTask) GetStdOut() io.Reader {
	return rst.out
}

func (rst *RunShTask) GetStdErr() io.Reader {
	return rst.err
}

func (rst *RunShTask) GetStdIn() io.Writer {
	return rst.in
}

func (rst *RunShTask) Run() error {
	if rst.Status != SCHEDULED {
		return errors.New("cannot run unscheduled task")
	}
	defer rst.outW.Close()
	defer rst.errW.Close()
	defer rst.inR.Close()
	//Get working directory
	var cwd string
	if d, ok := rst.Context.Value(WD).(string); ok {
		cwd = d
	} else {
		d, err := os.Getwd()
		if err != nil {
			return err
		}
		cwd = d
	}
	log.Printf("Task id: %d working directory: %s", rst.Id, cwd)
	//Get environment
	sysenv := os.Environ()
	if d, ok := rst.Context.Value(ENVVARS).(map[string]string); ok {
		for k, v := range d {
			sysenv = append(sysenv, fmt.Sprintf("%s=%s", k, v))
		}
	}
	log.Printf("Task id: %d environment: %s", rst.Id, sysenv)

	command := exec.CommandContext(rst.Context, rst.Command, rst.Args...)
	command.Dir = cwd
	command.Env = sysenv
	log.Printf("Running command and waiting for it to finish...")
	command.Stdout = rst.outW
	command.Stderr = rst.errW
	//command.Stdin = rst.inR
	command.Stdin = nil
	//Ugly but I did not found a better place
	if rst.save {
		out, err := storer.Save2FileFromWriter(rst.Id)
		if err != nil {
			log.Printf("Save to file for task %d is disabled. Error: %s", rst.Id, err)
		} else {
			ow := io.MultiWriter(rst.outW, out)
			eow := io.MultiWriter(rst.errW, out)
			command.Stdout = ow
			command.Stderr = eow
		}
	}

	//I will write nothing to the command
	//So closing stdin immediately
	err := rst.inR.Close()
	if err != nil {
		log.Printf("Cannot close stdin for task id: %d", rst.Id)
	}

	rst.Status = STARTED
	err = command.Run()
	if err != nil {
		if err.Error() == "context deadline exceeded" {
			log.Printf("Command timed out error: %v", err)
			rst.Status = TIMEOUT
		} else {
			log.Printf("Command finished with error: %v", err)
			rst.Status = FAILED
		}
	} else {
		rst.Status = DONE
		log.Println("Command completed successfully")
	}
	return err
}
