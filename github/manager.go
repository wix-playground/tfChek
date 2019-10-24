package github

import (
	"log"
	"strconv"
	"sync"
	"tfChek/misc"
)

var ml sync.Mutex
var m *Manager = nil

type Manager struct {
	data    chan *PRData
	client  GitHubClient
	stopped bool
}

type PRData struct {
	taskId     int
	successful bool
	log        *string
}

func NewPRData(taskId int, successful bool, output *string) *PRData {
	return &PRData{log: output, successful: successful, taskId: taskId}
}

func InitManager(repository, owner, token string) {
	ml.Lock()
	s := make(chan *PRData, 20)
	c := NewClientRunSH(repository, owner, token)
	m = &Manager{data: s, client: c, stopped: false}
	ml.Unlock()
	return
}

func GetManager() *Manager {
	return m
}

func (m *Manager) Start() {
	go m.starter()
}

func (m *Manager) starter() {
	for {
		if m.stopped {
			break
		}
		log.Println("Waiting for a new branch to create pull request")
		branch := <-m.data
		if branch != nil {
			process(branch)
		}
	}
}

func process(prd *PRData) {
	branch := misc.TASKPREFIX + strconv.Itoa(prd.taskId)
	number, err := m.client.CreatePR(branch)
	if err != nil {
		log.Printf("Failed to create PR Error: %s", err)
	} else {
		log.Printf("New PR #%d has been created", *number)
	}
	err = m.client.RequestReview(*number, &[]string{"maskimko"})
	if err != nil {
		log.Println("Failed to assign reviewers")
	}
	err = m.client.Comment(*number, prd.log)
	err = m.client.Review(*number, "run.sh finished ")
}

//func getOutput(branch string) string{
//	if strings.HasPrefix(branch, misc.TASKPREFIX) {
//		chunks := strings.Split(branch,"-")
//		if len(chunks) != 2 {
//			log.Printf("Cannot get task id from branch name %s", branch)
//			return misc.NOOUTPUT
//		}
//		id, err := strconv.Atoi(chunks[1])
//		if err != nil {
//			log.Printf("Cannot extract task id from branch %s", branch)
//			return misc.NOOUTPUT
//		}
//		tm := launcher.GetTaskManager()
//		t := tm.Get(id)
//		if t== nil {
//			log.Printf("Cannot obtain task %d",id)
//			return misc.NOOUTPUT
//		}
//		out := t.GetOut()
//		return out
//	}
//		log.Printf("Cannot parse branch name %s", branch)
//	return misc.NOOUTPUT
//}

func (m *Manager) GetChannel() chan<- *PRData {
	return m.data
}
func (m *Manager) Close() {
	m.stopped = true
	close(m.data)
}
