package launcher

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
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
	//Create task
	TaskOfRunSh(rcs RunShCmd, ctx context.Context) (Task, error)
	Launch(bt Task) error
	RegisterCancel(task Task, cancel func())
	GetTask(id int) Task
}

type TaskManagerImpl struct {
	sequence       int
	started        bool
	stop           chan bool
	threads        map[string]chan Task
	defaultWorkDir string
	lock           sync.Mutex
	cancel         map[int]func()
	activeTasks    map[int]Task
}

func (tm *TaskManagerImpl) GetTask(id int) Task {
	return tm.activeTasks[id]
}

func (tm *TaskManagerImpl) RegisterCancel(task Task, cancel func()) {
	tm.cancel[task.GetId()] = cancel
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
		cancel:         make(map[int]func()),
		activeTasks:    make(map[int]Task),
	}
}

//TODO: Reimplement it Make it more specific to be able to lock by state
func (tm *TaskManagerImpl) TaskOfRunSh(rcs RunShCmd, ctx context.Context) (Task, error) {
	command, args, err := rcs.CommandArgs()
	if err != nil {
		return nil, err
	}
	tm.sequence++
	t := BackgroundTaskImpl{Id: tm.sequence, Command: command, Args: args, Context: ctx, Status: OPEN, Socket: make(chan *websocket.Conn)}
	tm.activeTasks[t.Id] = &t
	return &t, nil
}

func (tm *TaskManagerImpl) Launch(bt Task) error {
	if bt.GetStatus() != OPEN {
		return errors.New("cannot launch task in not open status")
	}
	if tm.threads[bt.SyncName()] == nil {
		tm.lock.Lock()
		if tm.threads[bt.SyncName()] == nil {
			tm.threads[bt.SyncName()] = make(chan Task)
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
