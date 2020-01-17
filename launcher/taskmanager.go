package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"sync"
	"tfChek/misc"
	"time"
)

var tm TaskManager
var tml sync.Mutex

type TaskManager interface {
	Close() error
	Start() error
	IsStarted() bool
	//Create task
	AddRunSh(rcs *RunShCmd, ctx context.Context) (Task, error)
	Launch(bt Task) error
	LaunchById(id int) error
	RegisterCancel(id int, cancel context.CancelFunc) error
	Get(id int) Task
	Add(t Task) error
	Cancel(id int) error
}

type TaskManagerImpl struct {
	sequence       int
	sequenceFile   string
	started        bool
	stop           chan bool
	threads        map[string]chan Task
	defaultWorkDir string
	lock           sync.Mutex
	cancel         map[int]context.CancelFunc
	tasks          map[int]Task
	taskHashes     map[string]int
	saveRuns       bool
}

func (tm *TaskManagerImpl) incrementSequence() {

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

func (tm *TaskManagerImpl) AddRunSh(rcs *RunShCmd, ctx context.Context) (Task, error) {
	command, args, err := rcs.CommandArgs()
	if err != nil {
		return nil, err
	}
	outPipeReader, outPipeWriter := io.Pipe()
	errPipeReader, errPipeWriter := io.Pipe()
	inPipeReader, inPipeWriter := io.Pipe()
	t := RunShTask{Command: command, Args: args, Context: ctx,
		Status: misc.OPEN, save: tm.saveRuns,
		Socket:    make(chan *websocket.Conn),
		StateLock: fmt.Sprintf("%s/%s", rcs.Env, rcs.Layer),
		out:       outPipeReader, err: errPipeReader, in: inPipeWriter,
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
	writeSequence(tm.sequence)
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
		tml.Unlock()
	}
	return tm
}

func readSequence() int {
	rd := viper.GetString(misc.RunDirKey)
	if _, err := os.Stat(rd); os.IsNotExist(err) {
		log.Printf("Run directory %s does not exist. Creating one", rd)
		err := os.MkdirAll(rd, 0755)
		if err != nil {
			log.Printf("Cannot create run directory %s Error: %s", rd, err)
		}
	}
	rdf := path.Join(rd, "sequence")
	seqFile, err := os.Open(rdf)
	var seq int
	if err != nil {
		log.Printf("Cannot open sequence file %s Error: %s", rdf, err)
		return 0
	}
	defer seqFile.Close()
	seqBytes, err := ioutil.ReadAll(seqFile)
	if err != nil {
		log.Printf("Cannot read sequence file %s Error: %s", rdf, err)
		return 0
	}
	seq, err = strconv.Atoi(string(seqBytes))
	if err != nil {
		log.Printf("Cannot convert sequence %s form file %s Error: %s", seqBytes, rdf, err)
		return 0
	}
	log.Printf("Starting server from task counter value %d", seq)
	return seq
}

func writeSequence(i int) {
	rundir := viper.GetString(misc.RunDirKey)
	if _, err := os.Stat(rundir); os.IsNotExist(err) {
		err := os.MkdirAll(rundir, 0755)
		if err != nil {
			log.Printf("Cannot create run directory %s Error: %s", rundir, err)
		}
	}
	var seqFile *os.File
	rdf := path.Join(rundir, "sequence")
	seqFile, err := os.OpenFile(rdf, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Printf("Cannot open sequence file %s Error %s", rdf, err)
		return
	}

	defer seqFile.Close()
	_, err = seqFile.Write([]byte(strconv.Itoa(i)))
	if err != nil {
		if DEBUG {
			log.Printf("Cannot save sequence %d to file %s Error: %s", i, rdf, err)
		}
	}

}

func NewTaskManager() TaskManager {
	return &TaskManagerImpl{started: false,
		stop:     make(chan bool),
		sequence: readSequence(),
		threads:  make(map[string]chan Task),
		cancel:   make(map[int]context.CancelFunc),
		tasks:    make(map[int]Task),
		saveRuns: !viper.GetBool(misc.DismissOutKey),
	}
}

func (tm *TaskManagerImpl) Launch(bt Task) error {
	if bt.GetStatus() != misc.OPEN {
		return errors.New("cannot launch task in not open status")
	}
	if tm.threads[bt.SyncName()] == nil {
		tm.lock.Lock()
		if tm.threads[bt.SyncName()] == nil {
			tm.threads[bt.SyncName()] = make(chan Task, viper.GetInt(misc.QueueLengthKey))
		}
		tm.lock.Unlock()
	}
	bt.SetStatus(misc.SCHEDULED)
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
	go tm.starter()
	//Perhaps I should handle errors...
	return nil
}

func (tm *TaskManagerImpl) starter() error {
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
