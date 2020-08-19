package github

import (
	"fmt"
	"github.com/wix-system/tfChek/misc"
	"time"
)

type RepoManager struct {
	path          string
	remote        string
	cloned        bool
	Reference     string
	githubManager *Manager
	webhookLocks map[string]chan string
}

func NewRepomanager(path, remote string, webhookLocks map[string]chan string) *RepoManager {
	rm := RepoManager{path: path, remote: remote, webhookLocks: webhookLocks}
	rm.githubManager = GetManager(remote)
	return &rm
}

func (r RepoManager) Checkout(ref string) error {
	r.Reference = ref
	if r.githubManager == nil {
		//first try to obtain a new instance
		r.githubManager = GetManager(r.remote)
		if r.githubManager == nil {
			return fmt.Errorf("cannot obtain an instance of Github manager")
		}
	}
	err := DownloadRevision(r.githubManager, ref, r.path)
	if err != nil {
		return fmt.Errorf("failed to checkout revision %s Error: %w", ref, err)
	}
	return nil
}

func (r RepoManager) Pull() error {
	if r.Reference == "" {
		return r.Clone()
	} else {
		return r.Checkout(r.Reference)
	}
}

func (r RepoManager) SwitchTo(branch string) error {
	return r.Checkout(branch)
}

func (r RepoManager) Clone() error {
	return r.Checkout("master")
}

func (r RepoManager) Open() error {
	//This is not needed for direct downloads
	if r.cloned {
		return nil
	} else {
		return r.Clone()
	}
}

func (r RepoManager) GetPath() string {
	return r.path
}

func (r RepoManager) GetRemote() string {
	return r.remote
}

func (r RepoManager) IsCloned() bool {
	return r.cloned
}

func (r RepoManager) WaitForWebhook(branch string, timeout int) error {
	if timeout < 0 {
		return fmt.Errorf("timeout value cannot be negative")
	}
	if c, ok := r.webhookLocks[branch]; ok {
		if c != nil {
			select {
			case branchName := <-c:
				if branchName != branch {
					if len(branchName) > 0 {
						misc.Debugf("webhook lock for branch %s received wrong value %s in its bucket. It should be the same. This should never happen. Please contact developers", branch, branchName)
						return fmt.Errorf("webhook lock for branch %s received wrong value %s in its bucket. It should be the same. This should never happen. Please contact developers", branch, branchName)
					} else {
						misc.Debugf("warning. empty branch name in webhook wait at %s branch %s", r.path, branch)
					}
				} else {
					delete(r.webhookLocks, branch)
					misc.Debugf("webhook lock has been successfully consumed for branch %s", branch)
				}
			case <-time.After(time.Duration(timeout) * time.Second):
				misc.Debugf("webhook lock timeout reached after %d seconds for branch %s", timeout, branch)
				//Here I will not return an error, giving  chance eto fetch and checkout needed branch anyway.
				//return fmt.Errorf("webhook lock timeout reached after %d seconds for task %d", timeout,taskId)
			}
		} else {
			misc.Debugf("webhook lock for branch %s is nil", branch)
			return fmt.Errorf("webhook lock for branch %s is nil", branch)
		}
	} else {
		misc.Debugf("no lock for a branch %s in repo %s exists", branch,r.path)
		return fmt.Errorf("no lock for a branch %s in repo %s exists", branch, r.path)
	}
	return nil
}

func (r RepoManager) UnlockWebhookLock(branch string) error {
	if bl, ok := r.webhookLocks[branch]; ok {
		bl <- branch
		close(bl)
	} else {
		return fmt.Errorf("webhook for branch %s has not been registered", branch)
	}
	return nil
}

func (r RepoManager) RegisterWebhookLock(branch string) error {
	if _, ok := r.webhookLocks[branch]; !ok {
		r.webhookLocks[branch] = make(chan string, 1)
		misc.Debugf("webhook lock for a branch %s in repo %s has been successfully registered", branch, r.path)
	} else {
		return fmt.Errorf("git manager for %s already has locking channel for the branch %s", r.path, branch)
	}
	return nil
}
