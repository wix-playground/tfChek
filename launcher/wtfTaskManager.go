package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"github.com/wix-system/tfChek/misc"
	"github.com/wix-system/tfChek/storer"
	"github.com/wix-system/tfResDif/v3/apiv2"
	"github.com/wix-system/tfResDif/v3/helpers"
	"github.com/wix-system/tfResDif/v3/launcher"
	"log"
	"os"
	"sync"
	"time"
)

type WtfTaskManager interface {
	TaskManager
	AddWtfTask(payload *apiv2.TaskDefinition) (int, error)
}

type WtfTaskManagerImpl struct {
	sequence       int
	sequenceFile   string
	started        bool
	stop           chan bool
	threads        map[string]chan Task
	defaultWorkDir string
	lock           sync.Mutex
	cancel         map[int]context.CancelFunc
	tasks          map[int]Task
	//Deprecated
	taskHashes map[string]int
	saveRuns   bool
}

func NewWtfTaskManager() TaskManager {
	return &WtfTaskManagerImpl{started: false,
		stop:       make(chan bool),
		sequence:   readSequence(),
		threads:    make(map[string]chan Task),
		cancel:     make(map[int]context.CancelFunc),
		tasks:      make(map[int]Task),
		saveRuns:   !viper.GetBool(misc.DismissOutKey),
		taskHashes: make(map[string]int),
	}
}

func (tm *WtfTaskManagerImpl) Cancel(id int) error {
	cancel := tm.cancel[id]
	if cancel == nil {
		return errors.New(fmt.Sprintf("task id: %d has no registered cancel function", id))
	}
	log.Printf("Task id %d is set to be cancelled", id)
	cancel()
	return nil
}

func (tm *WtfTaskManagerImpl) IsStarted() bool {
	return tm.started
}

//Deprecated
func (tm *WtfTaskManagerImpl) AddRunSh(rcs *RunShCmd, ctx context.Context) (Task, error) {
	command, args, err := rcs.CommandArgs()
	if err != nil {
		return nil, err
	}
	//outPipeReader, outPipeWriter := io.Pipe()
	//errPipeReader, errPipeWriter := io.Pipe()
	//inPipeReader, inPipeWriter := io.Pipe()

	t := &RunShTask{Command: command, Args: args, Context: ctx,
		Status: misc.OPEN, save: tm.saveRuns,
		Socket:    make(chan *websocket.Conn),
		StateLock: fmt.Sprintf("%s/%s", rcs.Env, rcs.Layer),
		//out:       outPipeReader, err: errPipeReader, in: inPipeWriter,
		//outW: outPipehWriter, errW: errPipeWriter, inR: inPipeReader,
		//Perhaps it is better ot transfer Git Origins via the context
		GitOrigins: rcs.GitOrigins,
	}
	if ee, ok := ctx.Value(misc.EnvVarsKey).(*map[string]string); ok {
		t.ExtraEnv = *ee
	}

	err = tm.Add(t)
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot add task %v. Error: %s", t, err)
		}
	}
	tm.taskHashes[rcs.hash] = t.Id
	err = t.AddWebhookLocks()
	if err != nil {
		misc.Debugf("cannot add webhook locks for task %d", t.Id)
	}
	return t, err
}

func (tm *WtfTaskManagerImpl) AddWtfTask(payload *apiv2.TaskDefinition) (int, error) {
	ts := time.Unix(payload.Instant, 0)
	misc.Debugf("Creating a new task (ts: %s) ", ts.String())
	task := &WtfTask{context: payload.Context,
		StateLock: payload.Context.Location.GetLocationString(), status: misc.OPEN, Socket: make(chan *websocket.Conn)}
	err := tm.Add(task)
	if err != nil {
		misc.Debugf("cannot add task %v. Error: %s", task, err)
		return -1, fmt.Errorf("cannot add task %v. Error: %w", err)
	}
	tid := task.GetId()
	misc.Debugf("Task %d has been added", tid)
	var sink helpers.DescriptorSink
	sink, err = storer.NewTaskFileSink(tid)
	if err != nil {
		misc.Debugf("failed to create task file sink. Error: %s \nTrying to use standard out", err.Error())
		sink = helpers.NewStandardSink()
	}
	signals := make(chan os.Signal)
	wtfTaskLauncher := launcher.NewSinkSignallauncher(sink, signals)
	task.context.Launcher = wtfTaskLauncher
	task.context.DoGitUpdate = false
	task.context.DoNotify = false
	task.context.Debug = true
	err = task.AddWebhookLocks()
	if err != nil {
		misc.Debugf("cannot add webhook locks for task %d", tid)
		return tid, fmt.Errorf("cannot add webhook locks for task %d Error: %w", tid, err)
	}
	return tid, nil
}

func (tm *WtfTaskManagerImpl) Add(t Task) error {
	if t == nil {
		if viper.GetBool(misc.DebugKey) {
			log.Println("Cannot add nil task")
		}
		return errors.New("cannot add nil task")
	}
	//Sequence should be unique. So synchronizing is required now
	al.Lock()
	//Simple increment is not enough now. Need to update common value each time
	tm.sequence = readSequence()
	tm.sequence++
	t.setId(tm.sequence)
	writeSequence(tm.sequence)
	al.Unlock()
	tm.tasks[t.GetId()] = t
	return nil
}

func (tm *WtfTaskManagerImpl) LaunchById(id int) error {
	t := tm.Get(id)
	if t == nil {
		return errors.New(fmt.Sprintf("there is no task with id %d", id))
	}
	return tm.Launch(t)
}

func (tm *WtfTaskManagerImpl) Get(id int) Task {
	return tm.tasks[id]
}

func (tm *WtfTaskManagerImpl) GetId(hash string) (int, error) {
	if h, ok := tm.taskHashes[hash]; ok {
		return h, nil
	}
	return -1, errors.New(fmt.Sprintf("No task were registered with hash %s", hash))
}

func (tm *WtfTaskManagerImpl) RegisterCancel(id int, cancel context.CancelFunc) error {
	if tm.Get(id) == nil {
		return errors.New(fmt.Sprintf("there is no task with id %d", id))
	}
	tm.cancel[id] = cancel
	return nil
}

func (tm *WtfTaskManagerImpl) Launch(bt Task) error {
	if bt.GetStatus() != misc.OPEN {
		if bt.GetStatus() == misc.SCHEDULED {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Task %d has already been scheduled. Perhaps more than one webhook were precessed", bt.GetId())
			}
			return nil
		}
		return errors.New("cannot launch task in not scheduled status")
	}
	if tm.threads[bt.SyncName()] == nil {
		tm.lock.Lock()
		if tm.threads[bt.SyncName()] == nil {
			tm.threads[bt.SyncName()] = make(chan Task, viper.GetInt(misc.QueueLengthKey))
		}
		tm.lock.Unlock()
	}
	bt.SetStatus(misc.SCHEDULED)
	if viper.GetBool(misc.DebugKey) {
		log.Printf("Task %d has been scheduled", bt.GetId())
	}
	tm.threads[bt.SyncName()] <- bt

	return nil
}

func (tm *WtfTaskManagerImpl) Close() error {
	close(tm.stop)
	for id, c := range tm.cancel {
		log.Printf("Cancelling task %d", id)
		c()
		tm.cancel[id] = nil
	}
	return nil
}

func (tm *WtfTaskManagerImpl) Start() error {
	if tm.started {
		return errors.New("dispatcher already has been Started")
	}

	//TODO: implement readiness check
	go func() {
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
	}()
	return nil
}

func (tm *WtfTaskManagerImpl) runTasks(tasks <-chan Task) {
	for t := range tasks {
		err := t.Run()
		if err != nil {
			log.Printf("Task failed: %s", err)
		}
		//Clean up task cancel functions
		tm.cancel[t.GetId()] = nil
	}
}
