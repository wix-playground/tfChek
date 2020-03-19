package launcher

import (
	"errors"
	"fmt"
	"tfChek/misc"
	"tfChek/storer"
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
	follower := storer.NewFollower(fPath)
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
						if lm != nil {
							misc.Debug(lm.Error())
						}
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
					close(output)
					return
				default:
					misc.Debugf("received task status change event %s", GetStatusString(status))
				}
			}
		}
	}()
	return output, nil
}
