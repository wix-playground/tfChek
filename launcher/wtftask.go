package launcher

import (
	"bytes"
	"fmt"
	"github.com/acarl005/stripansi"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"github.com/wix-playground/tfChek/git"
	"github.com/wix-playground/tfChek/github"
	"github.com/wix-playground/tfChek/misc"
	"github.com/wix-playground/tfChek/storer"
	"github.com/wix-system/tfResDif/v3/core"
	wtfmisc "github.com/wix-system/tfResDif/v3/misc"
	"github.com/wix-system/tfResDif/v3/modes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
)

type WtfTask struct {
	//Deprecated
	name string
	id   int
	//Command   string
	//Args      []string
	//Deprecated
	ExtraEnv  map[string]string
	StateLock string
	context   *core.RunShContext
	status    TaskStatus
	Socket    chan *websocket.Conn
	//Deprecated
	GitOrigins  []string
	sink        bytes.Buffer
	authors     *[]string
	subscribers []chan TaskStatus
}

func (w *WtfTask) Run() error {

	//test
	if w.status != misc.SCHEDULED {
		return fmt.Errorf("cannot run unscheduled task")
	}
	w.status = misc.STARTED
	//Prepare github first
	err := w.prepareGitHub()
	if err != nil {
		log.Printf("Cannot prepare GitHub repositories. Error: %s", err)
		w.ForceFail()
		return err
	}
	//Perform git routines
	err = w.prepareGit()
	if err != nil {
		log.Printf("Cannot prepare git repositories. Error: %s", err)
		w.ForceFail()
		return err
	}

	//Inject RUNSH_PATH (important!)
	err = w.updateRunshPath()
	if err != nil {
		return fmt.Errorf("failed to generate RUNSH_PATH for task %d. Error: %w", w.id, err)
	}
	w.context.DoNotify = false
	w.context.DoGitUpdate = false

	runtimeError := modes.TerraformMode(wtfmisc.TerraformMode, w.context)
	if wtfmisc.CheckRuntimeError(runtimeError) {
		err := w.Fail()
		if err != nil {
			misc.Debugf("failed to set task %d status to failed. Error: %s", w.id, err)
		}
		return fmt.Errorf("Task failed. Error: %w", runtimeError)
	} else {
		err := w.Done()
		if err != nil {
			misc.Debugf("failed to set task %d status to failed. Error: %s", w.id, err)
		}
	}

	upload2s3(w.id, w.status)
	return err
	//test

}

func (w *WtfTask) GetId() int {
	return w.id
}

func (w *WtfTask) setId(id int) {
	w.id = id
}

func (w *WtfTask) Subscribe() chan TaskStatus {
	sts := make(chan TaskStatus, 2)
	sts <- w.status
	//Add channel to subscribers if the task is active
	if !IsCompleted(w) {
		w.subscribers = append(w.subscribers, sts)
	}
	return sts
}

func (w *WtfTask) GetStdOut() io.Reader {
	//TODO: Need closer
	path, err := storer.GetTaskPath(w.id)
	if err != nil {
		misc.Debugf("cannot get task stdout. Error: %s", err)
		return nil
	}
	taskFile, err := os.Open(path)
	if err != nil {
		misc.Debugf("cannot get task stdout. Error: %s", err)
		return nil
	}
	return taskFile
}

func (w *WtfTask) GetCleanOut() string {

	path, err := storer.GetTaskPath(w.id)
	if err != nil {
		misc.Debugf("cannot get task clean out. Error: %s", err)
		return ""
	}
	taskFile, err := os.Open(path)
	if err != nil {
		misc.Debugf("cannot get task stdout. Error: %s", err)
		return ""
	}
	defer taskFile.Close()
	content, err := ioutil.ReadAll(taskFile)
	if err != nil {
		misc.Debugf("cannot get task stdout. failed to read file %s, Error: %s", path, err)
		return ""
	}

	cleanOut := stripansi.Strip(string(content))
	return strings.TrimSpace(cleanOut)
}

func (w *WtfTask) GetStdErr() io.Reader {
	//Is not available in this implementation
	return nil
}

func (w *WtfTask) GetStdIn() io.Writer {
	return nil
}

func (w *WtfTask) GetStatus() TaskStatus {
	return w.status
}

func (w *WtfTask) SetStatus(status TaskStatus) {
	w.status = status
}

func (w *WtfTask) SyncName() string {
	return w.context.Location.GetLocationString()
}

func (w *WtfTask) Schedule() error {
	if w.status == misc.REGISTERED {
		w.status = misc.SCHEDULED
		w.notifySubscribers()
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled because it has been not registered. Please wait for a webhook. Current state number is %d", w.status)}
	}
}

func (w *WtfTask) Start() error {
	if w.status < misc.STARTED {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Start of task %s", w.name)
		}
		w.status = misc.STARTED
		w.notifySubscribers()
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be Started because it is not in scheduled state. Current state number is %d", w.status)}
	}
}

func (w *WtfTask) Done() error {
	if w.status == misc.STARTED {
		w.status = misc.DONE
		w.notifySubscribers()
		gitManagers, err := w.getGitManagers()
		if err != nil {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Cannot get Git managers. Error: %s", err)
			}
			return err
		}
		for gurl, _ := range gitManagers {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Processing GitHub manager of %s", gurl)
			}
			manager := github.GetManager(gurl)
			if manager != nil {
				c := manager.GetChannel()
				o := w.GetCleanOut()
				if o == "" {
					o = misc.NOOUTPUT
				}
				data := github.NewTaskResult(w.id, true, &o, w.GetAuthors())
				c <- data
			}
		}
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be done, because it has been not Started. Current state number is %d", w.status)}
	}
	return nil
}

func (w *WtfTask) Fail() error {
	if w.status == misc.STARTED {
		w.status = misc.FAILED
		w.notifySubscribers()
		fgm, err := w.getFirstGitManager()
		if err != nil {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Cannot get first Git manager. Error: %s", err)
			}
			return err
		}
		manager := github.GetManager(fgm.GetRemote())
		if manager != nil {
			c := manager.GetChannel()
			o := w.GetCleanOut()
			if o == "" {
				o = misc.NOOUTPUT
			}
			data := github.NewTaskResult(w.id, false, &o, w.GetAuthors())
			c <- data
		}
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be failed, because it has been not Started. Current state number is %d", w.status)}
	}
}

func (w *WtfTask) ForceFail() {
	w.status = misc.FAILED
	w.notifySubscribers()
}

func (w *WtfTask) TimeoutFail() error {
	if w.status == misc.STARTED {
		w.status = misc.TIMEOUT
		w.notifySubscribers()
		fgm, err := w.getFirstGitManager()
		if err != nil {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Cannot get first Git manager. Error: %s", err)
			}
			return err
		}
		manager := github.GetManager(fgm.GetRemote())
		if manager != nil {
			c := manager.GetChannel()
			o := w.GetCleanOut()
			if o == "" {
				o = misc.NOOUTPUT
			}
			data := github.NewTaskResult(w.id, false, &o, w.GetAuthors())
			c <- data
		}
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be timed out, because it has been not Started. Current state number is %d", w.status)}
	}

}

func (w *WtfTask) GetOrigins() *[]string {
	panic("implement me")
}

func (w *WtfTask) SetAuthors(authors []string) {
	w.authors = &authors
}

func (w *WtfTask) GetAuthors() *[]string {
	return w.authors
}

func (w *WtfTask) AddWebhookLocks() error {
	managers, err := w.getGitManagers()
	if err != nil {
		return fmt.Errorf("cannot get git managers for task %d %w", w.id, err)
	}
	branch := fmt.Sprintf("%s%d", misc.TaskPrefix, w.id)
	for _, m := range managers {
		err := m.RegisterWebhookLock(branch)
		if err != nil {
			return fmt.Errorf("cannot add webhook lock for task %d at %s, %w", w.id, m.GetPath(), err)
		}
	}
	return nil
}

func (w *WtfTask) UnlockWebhookRepoLock(fullName string) error {
	managers, err := w.getGitManagers()
	if err != nil {
		return fmt.Errorf("cannot get git managers for task %d %w", w.id, err)
	}
	branch := fmt.Sprintf("%s%d", misc.TaskPrefix, w.id)
	for _, m := range managers {
		frn, err := git.GetFullRepoName(m.GetRemote())
		if err != nil {
			return fmt.Errorf("cannot get full repository name of %s %w", m.GetRemote(), err)
		}
		if frn == fullName {
			err := m.UnlockWebhookLock(branch)
			if err != nil {
				return fmt.Errorf("cannot add webhook lock for task %d at %s, %w", w.id, m.GetPath(), err)
			}
			misc.Debugf("successfully unlocked task %d at %s", w.id, m.GetPath())
		}
	}
	return nil
}

func (w *WtfTask) notifySubscribers() {
	for _, sc := range w.subscribers {
		sc <- w.status
	}
	//Remove all subscribers after notification that task is completed
	if IsCompleted(w) {
		w.subscribers = nil
	}
}

//In this implementation I return production_42 repository always, because RG can be obsoleted in a future
//TODO: return a repository where modifications has been made
//Deprecated
func (w *WtfTask) getFirstGitManager() (git.Manager, error) {
	managers, err := w.getGitManagers()
	if err != nil {

		misc.Debugf("error obtaining git managers %s ", err)
		return nil, err
	}
	if len(managers) == 0 {
		misc.Debugf("no git managers were returned for a task %d", w.id)

	}
	m42, ok := managers[misc.PROD42]
	if !ok {
		misc.Debugf("failed to get %s repository. Falling back to %s one", misc.PROD42, misc.RG)
		mrg := managers[misc.RG]
		if mrg == nil {
			return nil, fmt.Errorf("failed to obtain git repository manager from task %d", w.id)
		}
		return mrg, nil
	}
	return m42, nil
}

func getApiVersionForRepomanager() int {
	if viper.GetBool(misc.GitHubDownload) {
		return 2
	} else {
		return 1
	}
}

//TODO: get rid of it
func (w *WtfTask) getGitManagers() (map[string]git.Manager, error) {
	gms := make(map[string]git.Manager)
	for _, cs := range w.context.ConfigSources {
		manager, err := git.GetManager(cs.RemoteUrl, w.StateLock, getApiVersionForRepomanager())
		if err != nil {
			return gms, fmt.Errorf("failed to get git manager for task %d (url: %s) error: %w", w.id, cs, err)
		}
		gms[cs.RemoteUrl] = manager
	}

	if len(gms) == 0 {
		return nil, fmt.Errorf("Cannot obtain a git manager. Task id %d contains no git remotes", w.id)
	} else {
		return gms, nil
	}
}

func (w *WtfTask) prepareGitHub() error {

	repoOwner := viper.GetString(misc.RepoOwnerKey)
	token := viper.GetString(misc.TokenKey)
	gms, err := w.getGitManagers()
	if err != nil {
		return fmt.Errorf("failed to obtain git managers for a task %d error: %w", w.id, err)
	}
	for key, gm := range gms {
		misc.Debugf("preparing GitHub manager %s", key)
		gitHubManager := github.GetManager(gm.GetRemote())
		if gitHubManager == nil {
			//Initialize GitHub manager
			github.InitManager(gm.GetRemote(), repoOwner, token)
			gitHubManager = github.GetManager(gm.GetRemote())
		}
		gitHubManager.Start()
	}
	return nil
}

func (w *WtfTask) prepareGit() error {
	//TODO: Perhaps it worth using Downloader directly right here instead of git interface

	//create RUNSH_APTH here for launching run.sh
	branch := fmt.Sprintf("%s%d", misc.TaskPrefix, w.id)
	//Prehaps here I have to convert git url to ssh url (in form of "git@github.com:...")
	gms, err := w.getGitManagers()
	if err != nil {
		return err
	}
	gmwg := &sync.WaitGroup{}
	errc := make(chan error)
	for gurl, gitman := range gms {
		gmwg.Add(1)
		go func(manager git.Manager, errChan chan<- error) {
			defer gmwg.Done()
			//Wait for corresponding webhook to come
			misc.Debugf("Waiting for a webhook to come from %s repo for a task %d", manager.GetRemote(), w.id)
			wht := viper.GetInt(misc.WebhookWaitTimeoutKey)
			err := manager.WaitForWebhook(branch, wht)
			if err != nil {
				errChan <- fmt.Errorf("failed to wait for a webhook lock. Error: %w", err)
				return
			}
			misc.Debugf("Preparing Git repo %s", gurl)

			//Clone it if needed
			if manager.IsCloned() {
				err := manager.Open()
				if err != nil {
					log.Printf("Cannot open git repository. Error: %s", err)
					errChan <- err
					return
				}
			} else {
				path := manager.GetPath()
				_, err := os.Stat(path)
				if os.IsNotExist(err) {
					err := os.MkdirAll(path, 0755)
					if err != nil {
						log.Printf("Cannot create directory for git repository. Error: %s", err)
						errChan <- err
						return
					}
				}
				err = manager.Clone()
				if err != nil {
					log.Printf("Cannot clone repository. Error: %s", err)
					errChan <- err
					return
				}
			}

			//Switch the branch
			err = manager.SwitchTo(branch)
			if err != nil {
				log.Printf("Cannot switch branch")
				errChan <- err
				return
			}
		}(gitman, errc)
	}
	go func() {
		gmwg.Wait()
		close(errc)
	}()
	for gme := range errc {
		if gme != nil {
			return gme
		}
	}
	misc.Debugf("preparation of git repositories succesfully finished for task %d", w.id)
	return nil
}

//Deprecated
func (w *WtfTask) generateRunshPath() (string, error) {
	var paths []string
	for _, cs := range w.context.ConfigSources {
		//repoName, err := git.GetFullRepoName(cs)
		//if err != nil {
		//	return "", fmt.Errorf("failed to get full repository name. Error: %w", err)
		//}

		manager, err := git.GetManager(cs.RemoteUrl, w.StateLock, getApiVersionForRepomanager())
		if err != nil {
			return "", fmt.Errorf("cannot obtain git manager for a task %d error: %w", w.id, err)
		}
		paths = append(paths, manager.GetPath())
	}
	return strings.Join(paths, ":"), nil
}

func (w *WtfTask) updateRunshPath() error {
	var paths []core.ConfigSource
	for _, cs := range w.context.ConfigSources {
		//repoName, err := git.GetFullRepoName(cs)
		//if err != nil {
		//	return "", fmt.Errorf("failed to get full repository name. Error: %w", err)
		//}

		manager, err := git.GetManager(cs.RemoteUrl, w.StateLock, getApiVersionForRepomanager())
		if err != nil {
			return fmt.Errorf("cannot obtain git manager for a task %d error: %w", w.id, err)
		}
		paths = append(paths, core.ConfigSource{manager.GetPath(), cs.RemoteUrl})
	}
	misc.Debugf("setting configuration sources for task %d to %v", w.id, paths)
	w.context.ConfigSources = paths
	return nil
}
