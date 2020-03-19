package launcher

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io"
	"os"
	"tfChek/misc"
	"tfChek/storer"
)

type Follower interface {
	Follow(lines chan<- string, errs chan<- error)
	Stop()
}

type controlFlag uint8

const stop controlFlag = 100

type follower struct {
	watcher  *fsnotify.Watcher
	reader   *os.File
	filePath string
	control  chan controlFlag
}

func NewFollower(file string) Follower {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		misc.Debug(err.Error())
		return nil
	}
	err = watcher.Add(file)
	if err != nil {
		misc.Debug(err.Error())
		return nil
	}
	_, err = os.Stat(file)
	if os.IsNotExist(err) {
		misc.Debug(fmt.Sprintf("file %s does not exist. Error: %s", file, err.Error()))
		return nil
	}
	f, err := os.Open(file)
	if err != nil {
		misc.Debugf("Cannot open file %s to obtain reader. Error: %s", file, err)
		return nil
	}
	cc := make(chan controlFlag)
	follower := follower{watcher: watcher, reader: f, filePath: file, control: cc}
	return &follower
}

func GetTaskLineReader(taskId int) (chan string, error) {

	fPath, err := storer.GetTaskPath(taskId)
	if err != nil {
		return nil, err
	}
	tm := GetTaskManager()
	task := tm.Get(taskId)
	if task == nil {
		msg := fmt.Sprintf("Cannot find a task by id: %d", taskId)
		misc.Debug(msg)
		return nil, errors.New(msg)
	}
	tsc := task.Subscribe()
	output := make(chan string)
	errs := make(chan error)
	follower := NewFollower(fPath)
	if follower == nil {
		return nil, errors.New(fmt.Sprintf("Cannot create follower for file %s", fPath))
	}
	go follower.Follow(output, errs)
	go func() {
		for {
			select {
			case status, ok := <-tsc:
				if !ok {
					misc.Debugf("---logging errors of file %s watcher:")
					for lm := range errs {
						misc.Debug(lm.Error())
					}
					misc.Debug("---")
					return
				}
				switch status {
				case misc.TIMEOUT:
					fallthrough
				case misc.FAILED:
					fallthrough
				case misc.DONE:
					misc.Debug("task is over")
					follower.Stop()
					return
				}
			}
		}
	}()
	return output, nil
}

func (f *follower) Follow(lines chan<- string, errs chan<- error) {
	counter := 0
	reader := bufio.NewReader(f.reader)
	for {
		s, err := reader.ReadBytes('\n')
		counter += len(s)
		if counter > 0 {
			lines <- string(s)
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				errs <- err
			}
		}
	}
	defer close(f.control)
	//for  range f.control {
	//	//Do nothing
	//}

	defer close(lines)
	defer close(errs)
	for {
		select {
		case err := <-f.watcher.Errors:
			if err != nil {
				misc.Debugf("watching file error: %s", err)
				errs <- err
			}
		case evt, ok := <-f.watcher.Events:
			if !ok {
				misc.Debugf("Watcher channel for file %s is closed. Stop watching", f.filePath)
				return
			}
			switch evt.Op {
			case fsnotify.Write:
				s, err := reader.ReadBytes('\n')
				counter += len(s)
				lines <- string(s)
				if err != nil {
					if err != io.EOF {
						errs <- err
					}
				}
			default:
				misc.Debug(fmt.Sprintf("File watcher received event: %s - %s", evt.Name, evt.String()))

			}

		case signal, ok := <-f.control:
			if !ok {
				misc.Debugf("Control channel for file %s follower has been closed", f.filePath)
			}
			if signal == stop {

				err := f.watcher.Close()
				if err != nil {
					misc.Debugf("Cannot close watcher for file %s", f.filePath)
				}
				err = f.reader.Close()
				if err != nil {
					misc.Debugf("Cannot close file %s", f.filePath)
				}
				return
			}
		}
	}
}

func (f *follower) Stop() {
	f.control <- stop
}
