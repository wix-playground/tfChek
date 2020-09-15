package launcher

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"github.com/wix-system/tfChek/misc"
	"github.com/wix-system/tfChek/storer"
	"io"
	"os"
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
	return getCompletedTaskOutput(taskId, true)
}
func getCompletedTaskOutput(taskId int, retry bool) ([]string, error) {
	data, err := storer.ReadTask(taskId)
	if err != nil {
		if os.IsNotExist(err) {
			misc.Debugf("failed to get task %d output. Reason: %s, Trying to get it from S3", taskId, err)
			err := PullS3TaskOutput(taskId)
			if err != nil {
				misc.Debugf("failed to get task %d output. Error: %s", taskId, err)
				return nil, fmt.Errorf("failed to get task %d output. Error: %s", taskId, err)
			} else {
				if retry {
					return getCompletedTaskOutput(taskId, false)
				} else {
					return nil, fmt.Errorf("cannot retry reading task %d more than once", taskId)
				}
			}
		} else {
			misc.Debugf("failed to get task %d output. Error: %s", taskId, err)
			return nil, fmt.Errorf("failed to get task %d output. Error: %s", taskId, err)
		}
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

func PullS3TaskOutput(taskId int) error {
	tm := GetTaskManager()
	task := tm.Get(taskId)
	if task == nil {
		misc.Debugf("failed to find task by id %d. Assuming it has been done before", taskId)
		err := pullS3TaskOutputWithStatus(taskId, misc.DONE)
		if err != nil {
			misc.Debugf("failed to download task %d output from S3 in status %s. trying nex one... Error: %s", taskId, GetStatusString(misc.DONE), err)
			err := pullS3TaskOutputWithStatus(taskId, misc.FAILED)
			if err != nil {
				misc.Debugf("failed to download task %d output from S3 in status %s. trying nex one... Error: %s", taskId, GetStatusString(misc.FAILED), err)
				err := pullS3TaskOutputWithStatus(taskId, misc.TIMEOUT)
				if err != nil {
					misc.Debugf("failed to download task %d output from S3 in status %s.  Error: %s", taskId, GetStatusString(misc.TIMEOUT), err)
					return fmt.Errorf("failed to download task %d output.  Error: %s", taskId, err)
				}
			}
		}
	} else {
		return pullS3TaskOutputWithStatus(taskId, task.GetStatus())
	}
	return nil
}

func pullS3TaskOutputWithStatus(taskId int, status TaskStatus) error {
	suffix := GetStatusString(status)
	bucketName := viper.GetString(misc.S3BucketName)
	err := storer.S3DownloadTaskWithSuffix(bucketName, taskId, &suffix)
	if err != nil {
		misc.Debugf("failed to download task output from S3. Error: %s", err)
		return fmt.Errorf("failed to download task output from S3. Error: %w", err)
	}
	return nil
}
