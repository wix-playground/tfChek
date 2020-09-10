package storer

import (
	"fmt"
	"github.com/wix-system/tfResDif/v3/wtflog"
	"io"
)

type TaskFileSink struct {
	out, err io.WriteCloser
	tid      int
}

func (f *TaskFileSink) GetStdErr() io.WriteCloser {
	return f.err
}

func (f *TaskFileSink) GetStdOut() io.WriteCloser {
	return f.out
}

func (f *TaskFileSink) GetStdIn() io.Reader {
	//By now I do not accept the user input
	return nil
}

func (f *TaskFileSink) Close() {
	err := f.out.Close()
	if err != nil {
		wtflog.GetLogger().Debugf("cannot close sink standard out for task %d", f.tid)
	}
	if f.err != f.out {
		err = f.err.Close()
		if err != nil {
			wtflog.GetLogger().Debugf("cannot close sink standard err for task %d", f.tid)
		}
	}
}

func NewTaskFileSink(taskId int) (*TaskFileSink, error) {
	fout, err := GetTaskFileWriteCloser(taskId)
	if err != nil {
		return nil, fmt.Errorf("failed to create a file sink for a task %d Error: %w", taskId, err)
	}
	fileSink := &TaskFileSink{tid: taskId, out: fout, err: fout}
	return fileSink, err
}
