package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"io"
	"log"
	"sync"
	"time"
)

const (
	WD      = "WORKING_DIRECTORY"
	ENVVARS = "ENVIRONMENT_VARIABLES"
)

var tm TaskManager
var tml sync.Mutex

type TaskManager interface {
	Close() error
	Start() error
	IsStarted() bool
	//Create task
	AddRunSh(rcs RunShCmd, ctx context.Context) (Task, error)
	Launch(bt Task) error
	LaunchById(id int) error
	RegisterCancel(id int, cancel context.CancelFunc) error
	Get(id int) Task
	Add(t Task) error
	Cancel(id int) error
}

type TaskManagerImpl struct {
	sequence       int
	started        bool
	stop           chan bool
	threads        map[string]chan Task
	defaultWorkDir string
	lock           sync.Mutex
	cancel         map[int]context.CancelFunc
	tasks          map[int]Task
	saveRuns       bool
}

func (tm *TaskManagerImpl) Cancel(id int) error {
	cancel := tm.cancel[id]
	if cancel == nil {
		return errors.New(fmt.Sprintf("task id: $d has no registered cancel function"))
	}
	log.Printf("Task id %d is set to be cancelled")
	cancel()
	return nil
}

func (tm *TaskManagerImpl) IsStarted() bool {
	return tm.started
}

func (tm *TaskManagerImpl) AddRunSh(rcs RunShCmd, ctx context.Context) (Task, error) {
	command, args, err := rcs.CommandArgs()
	if err != nil {
		return nil, err
	}
	outPipeReader, outPipeWriter := io.Pipe()
	errPipeReader, errPipeWriter := io.Pipe()
	inPipeReader, inPipeWriter := io.Pipe()
	t := BackgroundTaskImpl{Command: command, Args: args, Context: ctx,
		Status: OPEN, save: tm.saveRuns,
		Socket: make(chan *websocket.Conn),
		out:    outPipeReader, err: errPipeReader, in: inPipeWriter,
		outW: outPipeWriter, errW: errPipeWriter, inR: inPipeReader,
	}
	err = tm.Add(&t)
	if err != nil {
		if DEBUG {
			log.Printf("Cannot add task %v. Error: %s", t, err)
		}
	}
	return &t, err
}

func (tm *TaskManagerImpl) Add(t Task) error {
	if t == nil {
		if DEBUG {
			log.Println("Cannot add nil task")
		}
		return errors.New("cannot add nil task")
	}
	tm.sequence++
	t.setId(tm.sequence)
	tm.tasks[t.GetId()] = t
	return nil
}

func (tm *TaskManagerImpl) LaunchById(id int) error {
	t := tm.Get(id)
	if t == nil {
		return errors.New(fmt.Sprintf("there is no task with id %d", id))
	}
	return tm.Launch(t)
}

func (tm *TaskManagerImpl) Get(id int) Task {
	return tm.tasks[id]
}

func (tm *TaskManagerImpl) RegisterCancel(id int, cancel context.CancelFunc) error {
	if tm.Get(id) == nil {
		return errors.New(fmt.Sprintf("there is no task with id %d", id))
	}
	tm.cancel[id] = cancel
	return nil
}

func GetTaskManager() TaskManager {
	if tm == nil {
		tml.Lock()
		if tm == nil {
			tm = NewTaskManager()
		}
	}
	return tm
}

func NewTaskManager() TaskManager {
	return &TaskManagerImpl{started: false,
		stop:           make(chan bool),
		sequence:       0,
		threads:        make(map[string]chan Task),
		defaultWorkDir: "/tmp/production_42",
		cancel:         make(map[int]context.CancelFunc),
		tasks:          make(map[int]Task),
		saveRuns:       viper.GetBool("save"),
	}
}

func (tm *TaskManagerImpl) Launch(bt Task) error {
	if bt.GetStatus() != OPEN {
		return errors.New("cannot launch task in not open status")
	}
	if tm.threads[bt.SyncName()] == nil {
		tm.lock.Lock()
		if tm.threads[bt.SyncName()] == nil {
			tm.threads[bt.SyncName()] = make(chan Task, viper.GetInt("qlength"))
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

func (tm *TaskManagerImpl) runTasks(tasks <-chan Task) {
	for t := range tasks {
		err := t.Run()
		if err != nil {
			log.Printf("Task failed: %s", err)
		}
		//Clean up task cancel functions
		tm.cancel[t.GetId()] = nil
	}
}
