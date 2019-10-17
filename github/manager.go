package github

import (
	"log"
	"sync"
)

var ml sync.Mutex
var m *Manager = nil

type Manager struct {
	successful chan string
	client     GitHubClient
	stopped    bool
}

func InitManager(repository, owner, token string) {
	ml.Lock()
	s := make(chan string, 20)
	c := NewClientRunSH(repository, owner, token)
	m = &Manager{successful: s, client: c, stopped: false}
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
		branch := <-m.successful
		if branch != "" {
			pr, err := m.client.CreatePR(branch)
			if err != nil {
				log.Printf("Cannot create PR Error: %s", err)
			} else {
				log.Printf("New PR #%d has been created", pr)
			}
		}
	}
}
func (m *Manager) GetChannel() chan<- string {
	return m.successful
}
func (m *Manager) Close() {
	m.stopped = true
	close(m.successful)
}
