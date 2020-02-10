package github

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/whilp/git-urls"
	"log"
	"regexp"
	"strconv"
	"sync"
	"tfChek/misc"
)

var ml sync.Mutex
var managers map[string]*Manager = make(map[string]*Manager)

type Manager struct {
	data    chan *TaskResult
	client  Client
	stopped bool
	started bool
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
	c := NewClientRunSH(extractRepoName(repository), owner, token)
	managers[repository] = &Manager{data: s, client: c, stopped: false, started: false}
	ml.Unlock()
	return
}

func extractRepoName(repository string) string {
	parsed, err := giturls.Parse(repository)
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot parse URL: '%s' falling back to original")
		}
		return repository
	}
	re, err := regexp.Compile(".*/(.*?)(\\.git)*$")
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot compile regex '%s' falling back to original")
		}
		return repository
	}
	submatch := re.FindStringSubmatch(parsed.Path)
	if len(submatch) > 2 {
		return submatch[1]
	} else {
		// Falling back
		return repository
	}

}

func GetManager(repository string) *Manager {
	m := managers[repository]
	if m == nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("No GitHub manager for the repository %s. You might want to initialize this manager first")
		}
	}
	return m
}

func (m *Manager) Start() {
	if !m.started {
		m.started = true
		go m.starter()
	}
}

func (m *Manager) starter() {
	for {
		if m.stopped {
			break
		}
		log.Println("Waiting for a new taskResult to create pull request")
		taskResult := <-m.data
		if taskResult != nil {
			process(m, taskResult)
		}
	}
}

func process(m *Manager, prd *TaskResult) {
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

func CloseAll() {
	for repo, manager := range managers {
		log.Printf("Stopping GitHub manager for git repository %s", repo)
		manager.Close()
	}
}
