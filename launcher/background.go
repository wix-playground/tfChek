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
	"sync"
	"time"
)

type TaskManager interface {
	Close() error
	Start() error
	//Create task
	TaskOfRunSh(rcs RunShCmd, ctx context.Context) (BackgroundTask, error)
	Launch(bt BackgroundTask) error
	RegisterCancel(task BackgroundTask, cancel func())
}

type BackgroundTask interface {
	Run() error
	GetId() int
	GetStdOut() io.Reader
	GetStdErr() io.Reader
	GetStdIn() io.Writer
	GetStatus() TaskStatus
	SetStatus(status TaskStatus)
	SyncName() string
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
	sysenv := make([]string, 0)
	if d, ok := bti.Context.Value(ENVVARS).(map[string]string); ok {
		for k, v := range d {
			sysenv = append(sysenv, fmt.Sprintf("%s=%s", k, v))
		}
	} else {
		sysenv = os.Environ()
	}
	log.Printf("Task id: %d environment: %s", bti.Id, sysenv)

	command := exec.CommandContext(bti.Context, bti.Command, bti.Args...)
	command.Dir = cwd
	command.Env = sysenv
	log.Printf("Running command and waiting for it to finish...")
	command.Stdout = outPipeWriter
	command.Stderr = errPipeWriter
	command.Stdin = inPipeReader
	bti.Status = STARTED
	err := command.Run()
	if err != nil {
		log.Printf("Command finished with error: %v", err)
		bti.Status = FAILED
	} else {
		bti.Status = DONE
	}

	return err
}

type TaskManagerImpl struct {
	sequence       int
	started        bool
	stop           chan bool
	threads        map[string]chan BackgroundTask
	defaultWorkDir string
	lock           sync.Mutex
	cancel         map[int]func()
}

func (tm *TaskManagerImpl) RegisterCancel(task BackgroundTask, cancel func()) {
	tm.cancel[task.GetId()] = cancel
}

func NewTaskManager() TaskManager {
	return &TaskManagerImpl{started: false,
		stop:           make(chan bool),
		sequence:       0,
		threads:        make(map[string]chan BackgroundTask),
		defaultWorkDir: "/tmp/production_42",
		cancel:         make(map[int]func()),
	}
}

//TODO: Reimplement it Make it more specific to be able to lock by state
func (tm *TaskManagerImpl) TaskOfRunSh(rcs RunShCmd, ctx context.Context) (BackgroundTask, error) {
	command, args, err := rcs.CommandArgs()
	if err != nil {
		return nil, err
	}
	tm.sequence++
	t := BackgroundTaskImpl{Id: tm.sequence, Command: command, Args: args, Context: ctx, Status: OPEN, Socket: make(chan *websocket.Conn)}
	return &t, nil
}

func (tm *TaskManagerImpl) Launch(bt BackgroundTask) error {
	if bt.GetStatus() != OPEN {
		return errors.New("cannot launch task in not open status")
	}
	if tm.threads[bt.SyncName()] == nil {
		tm.lock.Lock()
		if tm.threads[bt.SyncName()] == nil {
			tm.threads[bt.SyncName()] = make(chan BackgroundTask)
		}
		tm.lock.Unlock()
	}
	bt.SetStatus(SCHEDULED)
	tm.threads[bt.SyncName()] <- bt

	return nil
}

func (tm *TaskManagerImpl) Close() error {
	close(tm.stop)
	for id, c := range tm.cancel {
		log.Printf("Cancelling task %d", id)
		c()
		tm.cancel[id] = nil
	}
	return nil
}

func (tm *TaskManagerImpl) Start() error {
	if tm.started {
		return errors.New("dispatcher already has been started")
	}
	started := make(map[string]bool)
	for {
		for s, tasks := range tm.threads {
			if !started[s] {
				go tm.runTasks(tasks)
				started[s] = true
			}
		}
		//Event sourcing
		select {
		case <-tm.stop:
			for _, tasks := range tm.threads {
				close(tasks)
			}
			break
		default:
			time.Sleep(time.Second)
		}
	}
}

func (tm *TaskManagerImpl) runTasks(tasks <-chan BackgroundTask) {
	for t := range tasks {
		err := t.Run()
		if err != nil {
			log.Printf("Task failed: %s", err)
		}
		//Clean up task cancel functions
		tm.cancel[t.GetId()] = nil
	}
}
