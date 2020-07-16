package launcher

import (
	"bytes"
	"context"
	"github.com/gorilla/websocket"
	"io"
)

type WtfTask struct {
	Name      string
	Id        int
	Command   string
	Args      []string
	ExtraEnv  map[string]string
	StateLock string
	Context   context.Context
	Status    TaskStatus
	Socket    chan *websocket.Conn
	//These are not needed anymore here.
	//out, err    io.Reader
	//in          io.Writer
	//inR         io.ReadCloser
	//outW, errW  io.WriteCloser

	//This should be always on
	//Remove this field in the future
	save        bool
	GitOrigins  []string
	sink        bytes.Buffer
	authors     []string
	subscribers []chan TaskStatus
}

func (WtfTask) Run() error {
	panic("implement me")
}

func (WtfTask) GetId() int {
	panic("implement me")
}

func (WtfTask) setId(id int) {
	panic("implement me")
}

func (WtfTask) Subscribe() chan TaskStatus {
	panic("implement me")
}

func (WtfTask) GetStdOut() io.Reader {
	panic("implement me")
}

func (WtfTask) GetCleanOut() string {
	panic("implement me")
}

func (WtfTask) GetStdErr() io.Reader {
	panic("implement me")
}

func (WtfTask) GetStdIn() io.Writer {
	panic("implement me")
}

func (WtfTask) GetStatus() TaskStatus {
	panic("implement me")
}

func (WtfTask) SetStatus(status TaskStatus) {
	panic("implement me")
}

func (WtfTask) SyncName() string {
	panic("implement me")
}

func (WtfTask) Schedule() error {
	panic("implement me")
}

func (WtfTask) Start() error {
	panic("implement me")
}

func (WtfTask) Done() error {
	panic("implement me")
}

func (WtfTask) Fail() error {
	panic("implement me")
}

func (WtfTask) ForceFail() {
	panic("implement me")
}

func (WtfTask) TimeoutFail() error {
	panic("implement me")
}
