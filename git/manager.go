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
)

var Debug bool = viper.GetBool(misc.DebugKey)
var lock sync.Mutex

//Key split by ';' on url and state
var repomngrs map[string]Manager

type Manager interface {
	Checkout(ref string) error
	Pull() error
	Clone() error
	Open() error
	GetPath() string
	IsCloned() bool
}

type BuiltInManager struct {
	remoteUrl string
	repoPath  string
	remote    *git.Remote
	repo      *git.Repository
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

func (b *BuiltInManager) GetPath() string {
	return b.repoPath
}

//In case the repo is used for generating .tf files for different terraform states
//Or it can be empty if terraform uses only 1 state
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
			path := fmt.Sprintf("%s/%s/%s", viper.GetString(misc.RepoDirKey), repoName, state)
			repomngrs[key] = &BuiltInManager{remoteUrl: url, repoPath: path}
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
		if Debug {
			log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
		}
		return err
	}
	worktree, err := b.repo.Worktree()
	if err != nil {
		if Debug {
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
		err = b.repo.Fetch(fo)
		if err != nil {
			log.Printf("Cannot fetch remoteUrl references. Error: %s", err)
			if err.Error() != "already up-to-date" {
				return err
			}
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
		if Debug && err.Error() == "reference not found" {
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
		if Debug {
			log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
		}
		return err
	}
	if len(remotes) == 0 {
		if Debug {
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
		log.Printf("Cannot fetch remoteUrl references. Error: %s", err)
		if err.Error() != "already up-to-date" {
			return err
		}

	}
	err = gwt.Pull(&git.PullOptions{RemoteName: b.remote.Config().Name, ReferenceName: gitRef.Name(), SingleBranch: true})
	if err != nil && err.Error() != "already up-to-date" {
		log.Printf("Cannot fetch remoteUrl references. Error: %s", err)
		if err.Error() != "already up-to-date" {
			return err
		}
	}
	return nil
}

func getRefSpecs(gitRef plumbing.ReferenceName, remote *git.Remote) (*[]config.RefSpec, error) {
	refName := gitRef.Short()
	headRefSpec := config.RefSpec(fmt.Sprintf("+HEAD:refs/remotes/%s/HEAD", remote.Config().Name))
	refSpecs := []config.RefSpec{headRefSpec}
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

	fo := &git.FetchOptions{RemoteName: remote.Config().Name, Depth: 10, RefSpecs: *refSpecs}
	return fo, nil
}

//func getFetchAllOptions(remote *git.Remote) *git.FetchOptions {
//	headRefSpec := config.RefSpec(fmt.Sprintf("+HEAD:refs/remotes/%s/HEAD", remote.Config().Name))
//	branchesSpec := config.RefSpec(fmt.Sprintf("+refs/heads/*:refs/remotes/%s/*", remote.Config().Name))
//	return &git.FetchOptions{RemoteName: remote.Config().Name, Depth: 10, RefSpecs: []config.RefSpec{headRefSpec, branchesSpec}}
//}

func (b *BuiltInManager) Clone() error {
	var prog io.Writer = nil
	if Debug {
		prog = os.Stderr
	}
	var err error
	b.repo, err = git.PlainClone(b.repoPath, false, &git.CloneOptions{URL: b.remoteUrl, Progress: prog, SingleBranch: true})
	if err != nil {
		b.repo = nil
		log.Printf("Cannot clone repository. Error: %s", err)
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
		if Debug {
			log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
		}
		//Not critical
		return err
	}
	//This should never happen
	if len(remotes) == 0 {
		if Debug {
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
