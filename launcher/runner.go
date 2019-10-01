package launcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	OPEN = iota
	SCHEDULED
	STARTED
	FAILED
	TIMEOUT
	DONE
)

var DEBUG bool = false

type TaskStatus uint8

type Task struct {
	Name             string
	Id               int
	WorkingDirectory string
	Command          string
	Args             []string
	Timeout          time.Duration
	Status           TaskStatus
	socket           *websocket.Conn
}

type StateError struct {
	msg string
}

func (se *StateError) Error() string {
	return se.msg
}

func (t *Task) Schedule() error {
	if t.Status < SCHEDULED {
		t.Status = SCHEDULED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled because it is not in open state. Current state number is %d", t.Status)}
	}
}

func (t *Task) Start() error {
	if t.Status < STARTED {
		if DEBUG {
			log.Printf("Start of unscheduled task %s", t.Name)
		}
		t.Status = STARTED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be started because it is not in scheduled state. Current state number is %d", t.Status)}
	}
}
func (t *Task) Done() error {
	if t.Status == STARTED {
		t.Status = DONE
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be done, because it has been not started. Current state number is %d", t.Status)}
	}
}

func (t *Task) Fail() error {
	if t.Status == STARTED {
		t.Status = FAILED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be failed, because it has been not started. Current state number is %d", t.Status)}
	}
}

func (t *Task) TimeoutFail() error {
	if t.Status == STARTED {
		t.Status = TIMEOUT
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be timed out, because it has been not started. Current state number is %d", t.Status)}
	}
}

func NewTask(name, workDir, command string, args []string, timeout time.Duration, socket *websocket.Conn) *Task {
	return &Task{Name: name, WorkingDirectory: workDir, Command: command, Args: args, Timeout: timeout, Status: OPEN, socket: socket}
}

func CommandRunner(tasks <-chan *Task) {
	for t := range tasks {
		errc := make(chan error)
		r, w := io.Pipe()
		defer w.Close()
		cancel := RunTask(t, errc, w)
		defer cancel()
		reader := bufio.NewReader(r)
		for {
			line, _, err := reader.ReadLine()
			if err == io.EOF {
				err = t.socket.WriteMessage(websocket.TextMessage, line)
				if err != nil {
					log.Println(err)
				}
				break
			}
			////temporary
			//for i, _ :=range line {
			//	if line[i] == '\r' {
			//		log.Printf("Carriage return")
			//		break
			//	}
			//}
			err = t.socket.WriteMessage(websocket.TextMessage, line)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func RunTask(t *Task, errc chan<- error, writer io.Writer) func() {
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), WD, t.WorkingDirectory), t.Timeout)

	err := t.Start()
	if err != nil {
		errc <- err
	}
	go func() {
		err := RunCommand(writer, ctx, t.Command, t.Args...)
		if err != nil {
			t.Status = FAILED
			errc <- err
		} else {
			t.Status = DONE
		}
		log.Printf("Task %s is over", t.Name)
	}()
	return cancel
}

func RunCommand(writer io.Writer, context context.Context, cmd string, args ...string) error {
	if len(cmd) < 1 {
		return errors.New("Empty command received")
	}

	var cwd string
	if d, ok := context.Value(WD).(string); ok {
		cwd = d
	} else {
		d, err := os.Getwd()
		if err != nil {
			return err
		}
		cwd = d
	}
	log.Printf("Working directory: %s", cwd)
	command := exec.CommandContext(context, cmd, args...)
	command.Dir = cwd
	command.Env = append(os.Environ(),
		"TFRESDIF_NOPB=true")
	log.Printf("Running command and waiting for it to finish...")
	command.Stdout = writer
	command.Stderr = writer
	err := command.Run()
	log.Printf("Command finished with error: %v", err)
	return err
}

type RunShCmd struct {
	All     bool
	Omit    bool
	No      bool
	Yes     bool
	Env     string
	Layer   string
	Targets []string
}

func (rsc *RunShCmd) CommandArgs() (string, []string, error) {
	command := "./run.sh"
	var args []string
	if rsc.All {
		args = append(args, "-a")
	}
	if rsc.Omit {
		args = append(args, "-o")
	}
	if rsc.No {
		args = append(args, "-n")
	}
	if rsc.Yes && !rsc.No {
		args = append(args, "-y")
	}
	if rsc.Env != "" {
		if rsc.Layer != "" {
			args = append(args, rsc.Env+"/"+rsc.Layer)
		} else {
			args = append(args, rsc.Env)
		}
	} else {
		return "", nil, errors.New("cannot launch run.sh if environment is not specified")
	}
	return command, args, nil
}
