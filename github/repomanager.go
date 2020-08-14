package github

type RepoManager struct {
	path      string
	remote    string
	cloned    bool
	Reference string
}

func (r RepoManager) Checkout(ref string) error {
	r.Reference = ref
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
