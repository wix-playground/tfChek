package github

import "fmt"

type RepoManager struct {
	path          string
	remote        string
	cloned        bool
	Reference     string
	githubManager *Manager
}

func NewRepomanager(path, remote string) *RepoManager {
	rm := RepoManager{path: path, remote: remote}
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
	panic("implement me")
}

func (r RepoManager) UnlockWebhookLock(branch string) error {
	panic("implement me")
}

func (r RepoManager) RegisterWebhookLock(branch string) error {
	panic("implement me")
}
