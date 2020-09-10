package tfChekLog

import (
	"fmt"
	"github.com/wix-system/tfResDif/v3/wtflog"
	"io"
	"log"
	"os"
)

type TaskLogger struct {
	log     *log.Logger
	taskId  int
	output  io.Writer
	IsDebug bool
}

func NewTaskLogger(taskId int, out io.Writer) wtflog.Logger {
	prefix := fmt.Sprintf("task-%d: ", taskId)
	tl := &TaskLogger{log: log.New(out, prefix, 0), taskId: taskId, output: out, IsDebug: true}
	return tl
}

func (t *TaskLogger) Debug(msg string) {
	if t.IsDebug {
		t.Log(msg)
	}
}

func (t *TaskLogger) Debugf(format string, args ...interface{}) {
	if t.IsDebug {
		t.Logf(format, args)
	}
}

func (t *TaskLogger) Log(msg string) {
	t.log.Print(msg)
}

func (t *TaskLogger) Logf(format string, args ...interface{}) {
	t.log.Print(fmt.Sprintf(format, args))
}

func (t *TaskLogger) Fatal(exitCode int, msg string) {
	t.Log(msg)
	os.Exit(exitCode)
}

func (t *TaskLogger) Fatalf(exitCode int, format string, args ...interface{}) {
	t.Logf(format, args)
	os.Exit(exitCode)
}

func (t *TaskLogger) GetLogger() *log.Logger {
	return t.log
}
