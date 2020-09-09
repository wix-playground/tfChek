package launcher

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/wix-system/tfChek/misc"
	"github.com/wix-system/tfChek/storer"
	"io"
)

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

	go func() {
		var follower storer.Follower = nil
		for {
			select {
			case status, ok := <-tsc:
				if !ok {
					misc.Debugf("---logging errors of file %s watcher:")
					for lm := range errs {
						if lm != nil {
							misc.Debug(lm.Error())
						}
					}
					misc.Debug("---")
					return
				}
				switch status {
				case misc.STARTED:
					//Create follower only when output log file are actually created
					follower, err := storer.NewFollower(fPath)
					if err != nil {
						errs <- errors.New(fmt.Sprintf("Cannot create follower for file %s Error: %s", fPath, err))
					} else {
						misc.Debugf("Starting follwer of task %d", taskId)
						go follower.Follow(output, errs)
					}
				case misc.TIMEOUT:
					fallthrough
				case misc.FAILED:
					fallthrough
				case misc.DONE:
					misc.Debug("task is over")
					if follower != nil {
						follower.Stop()
					}
					//Remove this
					//close(output)
					return
				default:
					misc.Debugf("received task status change event %s", GetStatusString(status))
				}
			}
		}
	}()
	return output, nil
}

func GetCompletedTaskOutput(taskId int) ([]string, error) {
	data, err := storer.ReadTask(taskId)
	if err != nil {
		misc.Debugf("Failed to get task %d output. Error: %s", taskId, err)
		return nil, err
	}
	br := bytes.NewReader(data)
	bbr := bufio.NewReader(br)
	var lines []string
	for {
		l, err := bbr.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				misc.Debugf("Failed to read from buffer. Error %s", err)
				return lines, err
			}
			break
		}
		lines = append(lines, l)
	}
	return lines, nil
}
