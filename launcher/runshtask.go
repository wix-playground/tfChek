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
	"tfChek/github"
	"tfChek/storer"
)

const TASKPREFIX = "tfci-"

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
		manager := github.GetManager()
		if manager != nil {
			c := manager.GetChannel()
			c <- fmt.Sprintf("%s%d", TASKPREFIX, rst.Id)
		}
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
	Task
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
	GitManager git.Manager
}

func (rst *RunShTask) ForceFail() {
	rst.Status = FAILED
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

func (rst *RunShTask) prepareGit() error {
	if rst.GitManager == nil {
		return errors.New("Git manager has been not initialized")
	}
	if rst.GitManager.IsCloned() {
		err := rst.GitManager.Open()
		if err != nil {
			log.Printf("Cannot open git repository. Error: %s", err)
			return err
		}
	} else {
		path := rst.GitManager.GetPath()
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			err := os.MkdirAll(path, 0755)
			if err != nil {
				log.Printf("Cannot create directory for git repository. Error: %s", err)
				return err
			}
		}
		err = rst.GitManager.Clone()
		if err != nil {
			log.Printf("Cannot clone repository. Error: %s", err)
			return err
		}
	}
	branch := fmt.Sprintf("%s%d", TASKPREFIX, rst.Id)
	err := rst.GitManager.Checkout(branch)
	if err != nil {
		log.Printf("Cannot checkout branch ")
		return err
	}
	err = rst.GitManager.Pull()
	if err != nil {
		log.Printf("Cannot pull changes. Error: %s", err)

	}
	return err
}

func (rst *RunShTask) Run() error {
	if rst.Status != SCHEDULED {
		return errors.New("cannot run unscheduled task")
	}
	//Perform git routines
	err := rst.prepareGit()
	if err != nil {
		log.Printf("Cannot prepare git repository. Error: %s", err)
		rst.ForceFail()
		return err
	}
	defer rst.outW.Close()
	defer rst.errW.Close()
	defer rst.inR.Close()
	//Get working directory
	cwd := rst.GitManager.GetPath()
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
	err = rst.inR.Close()
	if err != nil {
		log.Printf("Cannot close stdin for task id: %d", rst.Id)
	}

	err = rst.Start()
	if err != nil {
		log.Printf("Cannot change task state. Error: %s", err)
	}
	err = command.Run()
	if err != nil {
		if err.Error() == "context deadline exceeded" {
			log.Printf("Command timed out error: %v", err)
			err = rst.TimeoutFail()
			if err != nil {
				log.Printf("Cannot change task state. Error: %s", err)
			}
		} else {
			log.Printf("Command finished with error: %v", err)
			err = rst.Fail()
			if err != nil {
				log.Printf("Cannot change task state. Error: %s", err)
			}
		}
	} else {
		err = rst.Done()
		if err != nil {
			log.Printf("Cannot change task state. Error: %s", err)
		}
		log.Println("Command completed successfully")
	}
	return err
}
