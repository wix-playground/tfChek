package git

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"tfChek/misc"
	"time"
)

var lock sync.Mutex

//Key split by ';' on url and state
var repomngrs map[string]Manager

type Manager interface {
	Checkout(ref string) error
	Pull() error
	SwitchTo(branch string) error
	Clone() error
	Open() error
	GetPath() string
	GetRemote() string
	IsCloned() bool
	WaitForWebhook(branch string, timeout int) error
	UnlockWebhookLock(branch string) error
	RegisterWebhookLock(branch string) error
}

type BuiltInManager struct {
	remoteUrl    string
	repoPath     string
	remote       *git.Remote
	repo         *git.Repository
	webhookLocks map[string]chan string
}

func (b *BuiltInManager) GetRemote() string {
	return b.remoteUrl
}

func (b *BuiltInManager) Open() error {
	repository, err := git.PlainOpen(b.repoPath)
	if err != nil {
		log.Printf("Cannot open git repo %s. Error %s", b.repoPath, err)
		return err
	}
	b.repo = repository
	err = b.initRemotes()
	if err != nil {
		log.Printf("Cannot initialize remotes. Error: %s", err)
	}
	return nil
}

func (b *BuiltInManager) IsCloned() bool {
	_, err := os.Stat(b.repoPath + "/.git")
	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func (b *BuiltInManager) WaitForWebhook(branch string, timeout int) error {
	if timeout < 0 {
		return fmt.Errorf("timeout value cannot be negative")
	}
	if c, ok := b.webhookLocks[branch]; ok {
		if c != nil {
			select {
			case branchName := <-c:
				if branchName != branch {
					if len(branchName) > 0 {
						misc.Debugf("webhook lock for branch %s received wrong value %s in its bucket. It should be the same. This should never happen. Please contact developers", branch, branchName)
						return fmt.Errorf("webhook lock for branch %s received wrong value %s in its bucket. It should be the same. This should never happen. Please contact developers", branch, branchName)
					} else {
						misc.Debugf("warning. empty branch name in webhook wait at %s branch %s", b.repoPath, branch)
					}
				} else {
					delete(b.webhookLocks, branch)
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
		misc.Debugf("no lock for a branch %s in repo %s exists", branch, b.repoPath)
		return fmt.Errorf("no lock for a branch %s in repo %s exists", branch, b.repoPath)
	}
	return nil
}

//Locks fetching from the remote repository until corresponding webhook notifies about branch existence
func (b *BuiltInManager) RegisterWebhookLock(branch string) error {
	if _, ok := b.webhookLocks[branch]; !ok {
		b.webhookLocks[branch] = make(chan string, 1)
		misc.Debugf("webhook lock for a branch %s in repo %s has been successfully registered", branch, b.repoPath)
	} else {
		return fmt.Errorf("git manager for %s already has locking channel for the branch %s", b.repoPath, branch)
	}
	return nil
}

func (b *BuiltInManager) UnlockWebhookLock(branch string) error {
	if bl, ok := b.webhookLocks[branch]; ok {
		bl <- branch
		close(bl)
	} else {
		return fmt.Errorf("webhook for branch %s has not been registered", branch)
	}
	return nil
}

func (b *BuiltInManager) GetPath() string {
	return b.repoPath
}

//In case the repo is used for generating .tf files for different terraform states
//Or it can be empty if terraform uses only 1 state
//State can be  a env/layer for example
func GetManager(url, state string) (Manager, error) {
	if repomngrs == nil {
		lock.Lock()
		if repomngrs == nil {
			repomngrs = make(map[string]Manager)
		}
		lock.Unlock()
	}
	key := url
	if state != "" {
		key = key + ";" + state
	}
	if repomngrs[key] == nil {
		lock.Lock()
		if repomngrs[key] == nil {
			urlChunks := strings.Split(url, "/")
			repoName := urlChunks[len(urlChunks)-1]
			path := strings.TrimRight(fmt.Sprintf("%s/%s/%s", viper.GetString(misc.RepoDirKey), repoName, state), "/")
			whl := make(map[string]chan string)
			repomngrs[key] = &BuiltInManager{remoteUrl: url, repoPath: path, webhookLocks: whl}
		}
		lock.Unlock()
	}
	return repomngrs[key], nil
}

func (b *BuiltInManager) Checkout(branchName string) error {
	if b.repo == nil {
		return errors.New("the repository has been not cloned yet")
	}
	if b.repo.Storer == nil {
		log.Println("WARNING! Git repository Storer is nil value. This should never happen!")
	}
	remotes, err := b.repo.Remotes()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
		}
		return err
	}
	worktree, err := b.repo.Worktree()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot get worktree of git repository %s. Error: %s", b.repoPath, err)
		}
		return err
	}
	if len(remotes) == 0 {
		//This should never happen after clone
		log.Printf("Cannot find git remotes")
		return errors.New("cannot find git remotes")
	} else {
		//gitRef, err := b.repo.Head()
		//if err != nil {
		//	log.Printf("Cannot get HEAD. Error: %s", err)
		//	return err
		//}
		origin := b.remote.Config().Name
		branch := plumbing.NewBranchReferenceName(branchName)
		remoteBranch := plumbing.NewRemoteReferenceName(origin, branchName)
		fo, _ := getFetchOptions(branch, b.remote)
		misc.Debug(fmt.Sprintf("Trying to fetch branch %s form repo %s", branch, b.remote.String()))
		attempts := 5
		for i := 0; i < attempts; i++ {
			err = b.repo.Fetch(fo)
			if err != nil {
				if err.Error() == "already up-to-date" {
					misc.Debug(fmt.Sprintf("Branch %s of repo %s is already up to date", branch, b.remoteUrl))
					break
				}
				log.Printf("Checkout failed. Cannot fetch remoteUrl references from branch %s of repo %s. Error: %s", branch, b.remoteUrl, err)
				delay := 4 << i
				log.Printf("Attempt %d from %d failed. Cooldown %d seconds", i+1, attempts, delay)
				//Skip the timeout waiting after the last sleep
				if i < attempts-1 {
					time.Sleep(time.Duration(delay) * time.Second)
				}
			} else {
				break
			}
		}
		if err != nil && err.Error() != "already up-to-date" {
			return err
		}
		err = worktree.Checkout(&git.CheckoutOptions{Branch: branch})
		if err != nil {
			if err.Error() == "reference not found" {
				err = worktree.Checkout(&git.CheckoutOptions{Branch: remoteBranch})
				if err != nil {
					log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", remoteBranch, err)
					return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", branch.String(), err))
				}
				currentRef, err := b.repo.Head()
				if err != nil {
					log.Printf("Cannot get HEAD. Error: %s", err)
					return err
				}
				branchRef := plumbing.NewHashReference(branch, currentRef.Hash())

				if !branchRef.Name().IsRemote() {
					err = worktree.Checkout(&git.CheckoutOptions{Branch: branchRef.Name(), Create: true})
					if err != nil {
						log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", branchName, err)
						return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", branch.String(), err))
					}
				}
			} else {
				log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", branchName, err)
				return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", branch.String(), err))
			}
		}
	}
	return nil
}

func (b *BuiltInManager) Pull() error {
	gitRef, err := b.repo.Head()
	if err != nil {
		if viper.GetBool(misc.DebugKey) && err.Error() == "reference not found" {
			log.Printf("It looks like git pull process was interrupted before. Directory: %s", b.repoPath)
		}
		return err
	}
	gitHash := gitRef.Hash()
	gwt, err := b.repo.Worktree()
	if err != nil {
		return err
	}
	err = gwt.Reset(&git.ResetOptions{Commit: gitHash, Mode: git.HardReset})
	if err != nil {
		return err
	}
	remotes, err := b.repo.Remotes()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
		}
		return err
	}
	if len(remotes) == 0 {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Got no remotes of git repository %s", b.repoPath)
		}
		return errors.New("no remotes")
	}
	//Pick the first available remoteUrl
	headRefSpec := config.RefSpec("+HEAD:refs/remotes/origin/HEAD")
	err = headRefSpec.Validate()
	if err != nil {
		log.Printf("RefSpec is not valid. Error: %s", err)
		return err
	}

	fo, _ := getFetchOptions(gitRef.Name(), b.remote)
	err = b.repo.Fetch(fo)
	if err == nil {
		log.Printf("Pull failed. Cannot fetch remoteUrl references.  Cannot fetch remoteUrl references from reference %s of repo %s. Error: %s", gitRef.Name(), b.remoteUrl, err)
		if err.Error() != "already up-to-date" {
			return err
		}

	}
	err = gwt.Pull(&git.PullOptions{RemoteName: b.remote.Config().Name, ReferenceName: gitRef.Name(), SingleBranch: true})
	if err != nil && err.Error() != "already up-to-date" {
		log.Printf("Pull failed. Cannot pull remoteUrl references. Error: %s", err)
		if err.Error() != "already up-to-date" {
			return err
		}
	}
	return nil
}

func (b *BuiltInManager) SwitchTo(branch string) error {
	if b.repo == nil {
		return errors.New("the repository has been not cloned yet")
	}
	if b.repo.Storer == nil {
		log.Println("WARNING! Git repository Storer is nil value. This should never happen!")
	}
	remotes, err := b.repo.Remotes()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
		}
		return err
	}
	worktree, err := b.repo.Worktree()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot get worktree of git repository %s. Error: %s", b.repoPath, err)
		}
		return err
	}
	if len(remotes) == 0 {
		//This should never happen after clone
		log.Printf("Cannot find git remotes")
		return errors.New("cannot find git remotes")
	} else {
		origin := b.remote.Config().Name
		localBranch := plumbing.NewBranchReferenceName(branch)
		remoteBranch := plumbing.NewRemoteReferenceName(origin, branch)

		po := getPullOptions(localBranch, origin)
		misc.Debug(fmt.Sprintf("Trying to pull localBranch %s form repo %s", localBranch, b.remote.String()))
		attempts := 5
		for i := 0; i < attempts; i++ {
			err = worktree.Pull(po)
			if err != nil && err.Error() != "already up-to-date" {
				log.Printf("Pull failed. Cannot pull remoteUrl references. Error: %s", err)
				if err.Error() != "already up-to-date" {
					return err
				}
			}
			if err != nil {
				if err.Error() == "already up-to-date" {
					misc.Debug(fmt.Sprintf("Branch %s of repo %s is already up to date", localBranch, b.remoteUrl))
					break
				}
				log.Printf("Pull failed. Cannot fetch remoteUrl references from localBranch %s of repo %s. Error: %s", localBranch, b.remoteUrl, err)
				delay := 4 << i
				log.Printf("Attempt %d from %d failed. Cooldown %d seconds", i+1, attempts, delay)
				//Skip the timeout waiting after the last sleep
				if i < attempts-1 {
					time.Sleep(time.Duration(delay) * time.Second)
				}
			} else {
				break
			}
		}
		if err != nil && err.Error() != "already up-to-date" {
			return err
		}
		err = worktree.Checkout(&git.CheckoutOptions{Branch: localBranch, Create: true, Keep: false})
		if err != nil {
			if err.Error() == "reference not found" {
				err = worktree.Checkout(&git.CheckoutOptions{Branch: remoteBranch})
				if err != nil {
					log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", remoteBranch, err)
					return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", localBranch.String(), err))
				}
				currentRef, err := b.repo.Head()
				if err != nil {
					log.Printf("Cannot get HEAD. Error: %s", err)
					return err
				}
				branchRef := plumbing.NewHashReference(localBranch, currentRef.Hash())

				if !branchRef.Name().IsRemote() {
					err = worktree.Checkout(&git.CheckoutOptions{Branch: branchRef.Name(), Create: true})
					if err != nil {
						log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", branch, err)
						return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", localBranch.String(), err))
					}
				} else {
					misc.Debugf("branch name %s should be local one", branchRef)
				}
			} else {
				log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", branch, err)
				return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", localBranch.String(), err))
			}
		}
	}
	return nil
}

func getRefSpecs(gitRef plumbing.ReferenceName, remote *git.Remote) (*[]config.RefSpec, error) {
	refName := gitRef.Short()
	//Disable head temporary
	//headRefSpec := config.RefSpec(fmt.Sprintf("+HEAD:refs/remotes/%s/HEAD", remote.Config().Name))
	//refSpecs := []config.RefSpec{headRefSpec}
	refSpecs := []config.RefSpec{}
	if gitRef.IsBranch() {
		branchSpec := config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", refName, remote.Config().Name, refName))
		err := branchSpec.Validate()
		if err != nil {
			log.Printf("RefSpec is not valid. Error: %s", err)
			return &refSpecs, err
		}
		refSpecs = append(refSpecs, branchSpec)
	}
	return &refSpecs, nil
}

func getFetchOptions(gitRef plumbing.ReferenceName, remote *git.Remote) (*git.FetchOptions, error) {
	refSpecs, err := getRefSpecs(gitRef, remote)
	if err != nil {
		log.Printf("Could not get all ref specs. Error: %s", err)
	}

	fo := &git.FetchOptions{RemoteName: remote.Config().Name, Depth: 1, RefSpecs: *refSpecs}
	return fo, nil
}

func getPullOptions(gitRef plumbing.ReferenceName, remoteName string) *git.PullOptions {
	po := &git.PullOptions{RemoteName: remoteName, SingleBranch: true, ReferenceName: gitRef, RecurseSubmodules: git.NoRecurseSubmodules, Force: true}
	return po
}

//func getFetchAllOptions(remote *git.Remote) *git.FetchOptions {
//	headRefSpec := config.RefSpec(fmt.Sprintf("+HEAD:refs/remotes/%s/HEAD", remote.Config().Name))
//	branchesSpec := config.RefSpec(fmt.Sprintf("+refs/heads/*:refs/remotes/%s/*", remote.Config().Name))
//	return &git.FetchOptions{RemoteName: remote.Config().Name, Depth: 10, RefSpecs: []config.RefSpec{headRefSpec, branchesSpec}}
//}

func (b *BuiltInManager) Clone() error {
	var prog io.Writer = nil
	if viper.GetBool(misc.DebugKey) {
		prog = os.Stderr
	}
	var err error
	b.repo, err = git.PlainClone(b.repoPath, false, &git.CloneOptions{URL: b.remoteUrl, Progress: prog, SingleBranch: true})
	if err != nil {
		b.repo = nil
		log.Printf("Cannot clone repository from %s to %s. Error: %s", b.remoteUrl, b.repoPath, err)
		return err
	}
	err = b.initRemotes()
	if err != nil {
		log.Printf("Cannot initialize remotes. Error: %s", err)
	}
	return nil
}

func (b *BuiltInManager) initRemotes() error {
	remotes, err := b.repo.Remotes()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
		}
		//Not critical
		return err
	}
	//This should never happen
	if len(remotes) == 0 {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Got no remotes of git repository %s", b.repoPath)
		}
		//Not critical
		return errors.New("No remotes available")
	} else {
		for _, r := range remotes {
			if r.Config().Name == "origin" {
				b.remote = r
			}
		}
		if b.remote == nil {
			//Pick the first one
			b.remote = remotes[0]
		}
	}
	return nil
}
