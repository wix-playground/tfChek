package github

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/wix-playground/tfChek/misc"
	"os"
	"time"
)

type RepoManager struct {
	path          string
	basePath      string
	remote        string
	cloned        bool
	Reference     string
	githubManager *Manager
	webhookLocks  map[string]chan string
}

func NewRepomanager(path, remote string, webhookLocks map[string]chan string) *RepoManager {
	rm := RepoManager{path: path, remote: remote, webhookLocks: webhookLocks, basePath: path}
	rm.githubManager = GetManager(remote)
	if rm.githubManager == nil {
		misc.Debugf("Retrying obtain of repository manager for %s - %s", remote, path)
		repoOwner := viper.GetString(misc.RepoOwnerKey)
		token := viper.GetString(misc.TokenKey)
		InitManager(remote, repoOwner, token)
		rm.githubManager = GetManager(remote)
		if rm.githubManager == nil {
			misc.Debugf("failed to obtain repository manager for %s - %s", remote, path)
		} else {
			misc.Debugf("successfully obtained repository manager for %s - %s", remote, path)
		}
	}
	misc.Debugf("new github repository manager has been created for %s", remote)
	return &rm
}

func (r *RepoManager) Checkout(ref string) error {
	r.Reference = ref
	if r.githubManager == nil {
		//first try to obtain a new instance
		r.githubManager = GetManager(r.remote)
		if r.githubManager == nil {
			return fmt.Errorf("cannot obtain an instance of Github manager")
		}
	}
	extracted, err := DownloadRevision(r.githubManager, ref, r.basePath)
	if err != nil {
		return fmt.Errorf("failed to checkout revision %s Error: %w", ref, err)
	}
	misc.Debugf("successfully extracted repository root to %s", extracted)
	r.path = extracted
	return nil
}

func (r *RepoManager) Pull() error {
	if r.Reference == "" {
		return r.Clone()
	} else {
		return r.Checkout(r.Reference)
	}
}

func (r *RepoManager) SwitchTo(branch string) error {
	if branch == r.Reference {
		return nil
	}
	stat, err := os.Stat(r.path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot stat directory %s Error: %w", r.path, err)
		}
	} else {
		if stat.IsDir() {
			err := os.RemoveAll(r.path)
			if err != nil {
				return fmt.Errorf("failed to cleanup path %s Error: %w", r.path, err)
			}
		}
	}
	return r.Checkout(branch)
}

func (r *RepoManager) Clone() error {
	return r.Checkout("master")
}

func (r *RepoManager) Open() error {
	//This is not needed for direct downloads
	if r.cloned {
		return nil
	} else {
		return r.Clone()
	}
}

func (r *RepoManager) GetPath() string {
	return r.path
}

func (r *RepoManager) GetRemote() string {
	return r.remote
}

func (r *RepoManager) IsCloned() bool {
	return r.cloned
}

func (r *RepoManager) WaitForWebhook(branch string, timeout int) error {
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
		misc.Debugf("no lock for a branch %s in repo %s exists", branch, r.path)
		return fmt.Errorf("no lock for a branch %s in repo %s exists", branch, r.path)
	}
	return nil
}

func (r *RepoManager) UnlockWebhookLock(branch string) error {
	if bl, ok := r.webhookLocks[branch]; ok {
		bl <- branch
		close(bl)
	} else {
		return fmt.Errorf("webhook for branch %s has not been registered", branch)
	}
	return nil
}

func (r *RepoManager) RegisterWebhookLock(branch string) error {
	if _, ok := r.webhookLocks[branch]; !ok {
		r.webhookLocks[branch] = make(chan string, 1)
		misc.Debugf("webhook lock for a branch %s in repo %s has been successfully registered", branch, r.path)
	} else {
		return fmt.Errorf("git manager for %s already has locking channel for the branch %s", r.path, branch)
	}
	return nil
}
