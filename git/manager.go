package git

import (
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	fConfig "github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/spf13/viper"
	"github.com/wix-system/tfChek/github"
	"github.com/wix-system/tfChek/misc"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"
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
			if viper.GetBool(misc.GitHubDownload) {
				repomngrs[key] = github.NewRepomanager(path, url, whl)
			} else {
				repomngrs[key] = &BuiltInManager{remoteUrl: url, repoPath: path, webhookLocks: whl}
			}
		}
		lock.Unlock()
	}
	return repomngrs[key], nil
}

//Deprecated
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

//Deprecated
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
	//checkConfig(b.repo)
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
		}
		return err
	}
	gwt, err := b.repo.Worktree()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot get gwt of git repository %s. Error: %s", b.repoPath, err)
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
		fo, _ := getFetchOptions(localBranch, b.remote)
		//fo, _ := getFetchAllOptions(b.remote)
		misc.Debugf("Remote %s fetch refspecs: %v", b.remote.Config().Name, b.remote.Config().Fetch)
		misc.Debug(fmt.Sprintf("Trying to fetch localBranch %s form repo %s", localBranch, b.remote.String()))
		attempts := 5
		for i := 0; i < attempts; i++ {
			err = b.repo.Fetch(fo)
			if err != nil {
				if err.Error() == "already up-to-date" {
					misc.Debug(fmt.Sprintf("Branch %s of repo %s is already up to date", localBranch, b.remoteUrl))
					break
				}
				log.Printf("Checkout failed. Cannot fetch remoteUrl references from localBranch %s of repo %s. Error: %s", localBranch, b.remoteUrl, err)
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
		co := &git.CheckoutOptions{Branch: localBranch, Force: true}
		misc.Debugf("trying to checkout %s at %s", localBranch.String(), b.repoPath)
		err = gwt.Checkout(co)
		if err != nil {
			if err.Error() == "reference not found" {
				misc.Debugf("trying to checkout %s at %s", remoteBranch.String(), b.repoPath)
				err = gwt.Checkout(&git.CheckoutOptions{Branch: remoteBranch})
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
					err = gwt.Checkout(&git.CheckoutOptions{Branch: branchRef.Name(), Create: true})
					if err != nil {
						log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", branch, err)
						return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", localBranch.String(), err))
					}
				}
			} else {
				log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", branch, err)
				return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", localBranch.String(), err))
			}
		}
		b.checkConfig()
		err := b.trackBranch(origin, branch)
		if err != nil {
			return fmt.Errorf("cannot set tracked branch. Error: %w", err)
		}
		b.checkConfig()
		po := getPullOptions(localBranch, origin)
		misc.Debug(fmt.Sprintf("Trying to pull localBranch %s form repo %s", localBranch, b.remote.String()))
		for i := 0; i < attempts; i++ {
			err = gwt.Pull(po)
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
	}
	//if b.repo == nil {
	//	return errors.New("the repository has been not cloned yet")
	//}
	//if b.repo.Storer == nil {
	//	log.Println("WARNING! Git repository Storer is nil value. This should never happen!")
	//}
	//remotes, err := b.repo.Remotes()
	//if err != nil {
	//	if viper.GetBool(misc.DebugKey) {
	//		log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
	//	}
	//	return err
	//}
	//gwt, err := b.repo.Worktree()
	//if err != nil {
	//	if viper.GetBool(misc.DebugKey) {
	//		log.Printf("Cannot get gwt of git repository %s. Error: %s", b.repoPath, err)
	//	}
	//	return err
	//}
	//if len(remotes) == 0 {
	//	//This should never happen after clone
	//	log.Printf("Cannot find git remotes")
	//	return errors.New("cannot find git remotes")
	//} else {
	//	origin := b.remote.Config().Name
	//	localBranch := plumbing.NewBranchReferenceName(branch)
	//	remoteBranch := plumbing.NewRemoteReferenceName(origin, branch)
	//
	//	po := getPullOptions(localBranch, origin)
	//	misc.Debug(fmt.Sprintf("Trying to pull localBranch %s form repo %s", localBranch, b.remote.String()))
	//	attempts := 5
	//	for i := 0; i < attempts; i++ {
	//		err = gwt.Pull(po)
	//		if err != nil {
	//			if err.Error() == "already up-to-date" {
	//				misc.Debug(fmt.Sprintf("Branch %s of repo %s is already up to date", localBranch, b.remoteUrl))
	//				break
	//			}
	//			log.Printf("Pull failed. Cannot fetch remoteUrl references from localBranch %s of repo %s. Error: %s", localBranch, b.remoteUrl, err)
	//			delay := 4 << i
	//			log.Printf("Attempt %d from %d failed. Cooldown %d seconds", i+1, attempts, delay)
	//			//Skip the timeout waiting after the last sleep
	//			if i < attempts-1 {
	//				time.Sleep(time.Duration(delay) * time.Second)
	//			}
	//		} else {
	//			break
	//		}
	//	}
	//	if err != nil && err.Error() != "already up-to-date" {
	//		return err
	//	}
	//	err = gwt.Checkout(&git.CheckoutOptions{Branch: localBranch, Create: true, Keep: false})
	//	if err != nil {
	//		if err.Error() == "reference not found" {
	//			err = gwt.Checkout(&git.CheckoutOptions{Branch: remoteBranch})
	//			if err != nil {
	//				log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", remoteBranch, err)
	//				return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", localBranch.String(), err))
	//			}
	//			currentRef, err := b.repo.Head()
	//			if err != nil {
	//				log.Printf("Cannot get HEAD. Error: %s", err)
	//				return err
	//			}
	//			branchRef := plumbing.NewHashReference(localBranch, currentRef.Hash())
	//
	//			if !branchRef.Name().IsRemote() {
	//				err = gwt.Checkout(&git.CheckoutOptions{Branch: branchRef.Name(), Create: true})
	//				if err != nil {
	//					log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", branch, err)
	//					return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", localBranch.String(), err))
	//				}
	//			} else {
	//				misc.Debugf("branch name %s should be local one", branchRef)
	//			}
	//		} else {
	//			log.Printf("Cannot checkout remoteUrl reference %s. Error: %s", branch, err)
	//			return errors.New(fmt.Sprintf("cannot checkout remoteUrl reference %s. Error: %s", localBranch.String(), err))
	//		}
	//	}
	//}
	return nil
}

func (b *BuiltInManager) trackBranch(remote, branch string) error {
	lb := plumbing.NewBranchReferenceName(branch)
	rb := plumbing.NewRemoteReferenceName(remote, branch)
	refSpec := config.RefSpec(fmt.Sprintf("+%s:%s", lb, rb))
	err := refSpec.Validate()
	if err != nil {
		return fmt.Errorf("refspec %q validation failed. Error %w", refSpec.String(), err)
	}
	conf, err := b.repo.Config()
	if err != nil {
		return fmt.Errorf("cannot get conf of git repo. Error: %w", err)
	}
	rawConfig := conf.Raw
	if !rawConfig.Section(misc.GitSectionRemote).HasSubsection(remote) {
		return fmt.Errorf("remote subsections does not contain given remote %q ", remote)
	}
	remoteSection := rawConfig.Section(misc.GitSectionRemote)
	remoteSubSection := remoteSection.Subsection(remote)
	branchSection := rawConfig.Section(misc.GitSectionBranch)
	remoteSubSection = remoteSubSection.AddOption(misc.GitSectionOptionFetch, refSpec.String())
	remoteSection.Subsections = []*fConfig.Subsection{remoteSubSection}
	rawConfig.Sections[1] = remoteSection
	remConf := conf.Remotes[remote]
	remConf.Fetch = append(remConf.Fetch, refSpec)
	branchSubSection := createSubsectionForBranch(branch, remote)
	branchSection.Subsections = append(branchSection.Subsections, branchSubSection)
	rawConfig.Sections[2] = branchSection
	conf.Raw = rawConfig
	branchObj := &config.Branch{Name: branch, Remote: remote, Merge: lb}
	conf.Branches[branch] = branchObj
	err = b.saveConfig(conf)
	if err != nil {
		return fmt.Errorf("failed to save git configuration for %s Error: %w", b.repoPath, err)
	}
	return nil
}

func (b *BuiltInManager) saveConfig(conf *config.Config) error {
	serializedConfig, err := conf.Marshal()
	if err != nil {
		return fmt.Errorf("cannot serialize git configuration. Error: %w", err)
	}
	configPath := path.Join(b.repoPath, ".git", "config")
	err = ioutil.WriteFile(configPath, serializedConfig, 0644)
	return err
}

func createSubsectionForBranch(branch, remote string) *fConfig.Subsection {
	ref := plumbing.NewBranchReferenceName(branch)
	var opts []*fConfig.Option
	opts = append(opts, &fConfig.Option{Key: misc.GitSectionRemote, Value: remote}, &fConfig.Option{Key: misc.GitSectionOptionMerge, Value: ref.String()})
	s := &fConfig.Subsection{Name: branch, Options: fConfig.Options(opts)}
	return s
}

func (b *BuiltInManager) checkConfig() {
	c, err := b.repo.Config()
	if err != nil {
		misc.Debugf("cannot get config. Error: %s", err.Error())
		return
	}
	for i, s := range c.Raw.Sections {
		misc.Debugf("configuration section %d %s", i, s.Name)
		misc.Debugf("options: ")
		for io, o := range s.Options {
			misc.Debugf("- option %d %s=%s", io, o.Key, o.Value)
		}
		for n, ss := range s.Subsections {
			misc.Debugf("\t%d. subsection %s of %s", n, ss.Name, s.Name)
			misc.Debugf("\t subsection options: ")
			for io, o := range ss.Options {
				misc.Debugf("\t- subsection option %d %s=%s", io, o.Key, o.Value)
			}
		}

	}
}

func getBranchRefSpecs(gitRef plumbing.ReferenceName, remote *git.Remote) (*[]config.RefSpec, error) {
	if !gitRef.IsBranch() {
		return nil, fmt.Errorf("%s should be branch", gitRef.String())
	}
	refName := gitRef.Short()
	fetchRefSpecs := remote.Config().Fetch
	branchRef := config.RefSpec(fmt.Sprintf("+%s:%s", plumbing.NewBranchReferenceName(refName), plumbing.NewRemoteReferenceName(remote.Config().Name, refName)))
	err := branchRef.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation of branch reference '%s' failed. Error: %w", branchRef.String(), err)
	}
	fetchRefSpecs = append(fetchRefSpecs, branchRef)
	return &fetchRefSpecs, nil
	//headRefSpec := config.RefSpec(fmt.Sprintf("+HEAD:refs/remotes/%s/HEAD", remote.Config().Name))
	//refSpecs := []config.RefSpec{headRefSpec}
	////refSpecs := []config.RefSpec{}
	//if gitRef.IsBranch() {
	//	branchSpec := config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", refName, remote.Config().Name, refName))
	//	err := branchSpec.Validate()
	//	if err != nil {
	//		log.Printf("RefSpec is not valid. Error: %s", err)
	//		return &refSpecs, err
	//	}
	//	refSpecs = append(refSpecs, branchSpec)
	//}
	//return &refSpecs, nil
}

func getFetchOptions(gitRef plumbing.ReferenceName, remote *git.Remote) (*git.FetchOptions, error) {
	refSpecs, err := getBranchRefSpecs(gitRef, remote)
	if err != nil {
		log.Printf("Could not get all ref specs. Error: %s", err)
	}

	fo := &git.FetchOptions{RemoteName: remote.Config().Name, RefSpecs: *refSpecs}
	return fo, nil
}

func getFetchAllOptions(remote *git.Remote) (*git.FetchOptions, error) {
	fo := &git.FetchOptions{RemoteName: remote.Config().Name, RefSpecs: []config.RefSpec{config.RefSpec("+refs/heads/*:refs/remotes/origin/*")}}
	return fo, nil
}

func getPullOptions(gitRef plumbing.ReferenceName, remoteName string) *git.PullOptions {
	po := &git.PullOptions{RemoteName: remoteName, SingleBranch: true, ReferenceName: gitRef, RecurseSubmodules: git.NoRecurseSubmodules, Force: true}
	//po := &git.PullOptions{SingleBranch: false, ReferenceName: gitRef, RecurseSubmodules: git.NoRecurseSubmodules, Force: true, Depth: 50}
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
