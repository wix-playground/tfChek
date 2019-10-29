package github

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"tfChek/misc"
)

var ml sync.Mutex
var m *Manager = nil

type Manager struct {
	data    chan *TaskResult
	client  Client
	stopped bool
}

type TaskResult struct {
	taskId     int
	successful bool
	log        *string
	authors    *[]string
}

func NewTaskResult(taskId int, successful bool, output *string, authors *[]string) *TaskResult {
	return &TaskResult{log: output, successful: successful, taskId: taskId, authors: authors}
}

func InitManager(repository, owner, token string) {
	ml.Lock()
	s := make(chan *TaskResult, 20)
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

func process(prd *TaskResult) {
	branch := misc.TaskPrefix + strconv.Itoa(prd.taskId)
	switch prd.successful {
	case true:
		number, err := m.client.CreatePR(branch)
		if err != nil {
			log.Printf("Failed to create GitHub PR Error: %s", err)
		} else {
			log.Printf("New PR #%d has been created", *number)
			err = m.client.RequestReview(*number, prd.authors)
			if err != nil {
				log.Println("Failed to assign reviewers")
			}
			err = m.client.Comment(*number, prd.log)
			if err != nil {
				log.Printf("Cannot comment PR %d Error: %s", number, err)
			}
			//Disable review as there is no reason to perform review by the push author
			//Requesting reviewers is just enough
			//err = m.client.Review(*number, "run.sh finished ")
			//if err != nil {
			//	log.Printf("Cannot review PR %d Error: %s", number, err)
			//}
			message := fmt.Sprintf("Automatically merged by tfChek (Authors %v)", *prd.authors)
			sha, err := m.client.Merge(*number, message)
			if err != nil {
				log.Printf("Cannot merge branch %s, Error: %s", branch, err)
			} else {
				log.Printf("Branch %s has been merged. Merge commit hash %s", branch, *sha)
			}
		}
	case false:
		number, err := m.client.CreateIssue(branch, prd.authors)
		if err != nil {
			log.Printf("Failed to create GitHub Issue Error: %s", err)
		} else {
			log.Printf("New Issue #%d has been created", *number)
			err = m.client.Comment(*number, prd.log)
			if err != nil {
				log.Printf("Cannot comment issue %d Error: %s", number, err)
			}
		}
	}
}

//func getOutput(branch string) string{
//	if strings.HasPrefix(branch, misc.TaskPrefix) {
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

func (m *Manager) GetChannel() chan<- *TaskResult {
	return m.data
}
func (m *Manager) Close() {
	m.stopped = true
	close(m.data)
}
