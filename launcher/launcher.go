package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	WD      = "WORKING_DIRECTORY"
	ENVVARS = "ENVIRONMENT_VARIABLES"
)

type Dispatcher interface {
	Launch(conn *websocket.Conn, env, layer string, cmd []string) (int, error)
	Close() error
	Start() error
}

type DispatcherImpl struct {
	started  bool
	stop     chan bool
	sequence int
	threads  map[string]chan *Task
	lock     sync.Mutex
	wd       string
}

func NewDispatcher() Dispatcher {
	//return &DispatcherImpl{started: false, stop: make(chan bool), sequence: 0, threads: make(map[string]chan *Task), wd: "/Users/maksymsh/terraform/production_42"}
	return &DispatcherImpl{started: false, stop: make(chan bool), sequence: 0, threads: make(map[string]chan *Task), wd: "/tmp/production_42"}
}

//Deprecated
func RunCommands(writer io.WriteCloser, context context.Context, commands *map[string][]string) <-chan error {
	errc := make(chan error)
	defer close(errc)
	if commands == nil || len(*commands) == 0 {
		log.Printf("No commands were passed")
		return nil
	}
	for name, cmd := range *commands {
		log.Printf("[%s]\tLaunching command: %s", name, strings.Join(cmd, " "))
		err := RunCommand(writer, context, cmd[0], cmd[1:]...)
		if err != nil {
			log.Printf("Command failed. Error: %s", err)
			errc <- err
		}
	}
	return errc
}

func (d *DispatcherImpl) increment() int {
	d.sequence++
	return d.sequence
}

//This method accepts a task to launch and returns execution ID
//If execution is not possible it returns -1 and error
func (d *DispatcherImpl) Launch(c *websocket.Conn, env, layer string, cmd []string) (int, error) {

	if env == "" {
		//TODO: enhance this output
		return -1, errors.New("you must specify environment")
	}
	state := env
	if layer != "" {
		state = state + "/layer"
	}
	if d.threads[state] == nil {
		//NOT thread safe
		//TODO: improve it using ready channel for example
		d.lock.Lock()
		d.threads[state] = make(chan *Task, 50)
		d.lock.Unlock()
	}
	t := NewTask(fmt.Sprintf("Task %d", d.sequence), d.wd, cmd[0], cmd[1:], 300*time.Second, c)
	d.threads[state] <- t
	t.Schedule()
	return d.increment(), nil
}

func (d *DispatcherImpl) Close() error {
	d.stop <- true
	return nil
}

func (d *DispatcherImpl) Start() error {
	if d.started {
		return errors.New("dispatcher already has been started")
	}
	started := make(map[string]bool)
	for {
		for s, tasks := range d.threads {
			if !started[s] {
				go CommandRunner(tasks)
				started[s] = true
			}
		}
		//Event sourcing
		select {
		case <-d.stop:
			for _, tasks := range d.threads {
				close(tasks)
			}
			break
		default:
			time.Sleep(time.Second)
		}
	}
}

//TODO: Optimize interface
func LaunchRunSh(d Dispatcher, c *websocket.Conn, cmd RunShCmd) (int, error) {
	command, args, err := cmd.CommandArgs()
	if err != nil {
		return -1, err
	}
	comargs := []string{command}
	comargs = append(comargs, args...)
	return d.Launch(c, cmd.Env, cmd.Layer, comargs)
}
