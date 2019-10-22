package git

import (
	"errors"
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"io"
	"log"
	"os"
)

var Debug bool

type Manager interface {
	Checkout(ref string) error
	Pull() error
	Clone() error
}

type BuiltInManager struct {
	remoteUrl string
	repoPath  string
	remote    *git.Remote
	repo      *git.Repository
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
func getFetchAllOptions(remote *git.Remote) *git.FetchOptions {
	headRefSpec := config.RefSpec(fmt.Sprintf("+HEAD:refs/remotes/%s/HEAD", remote.Config().Name))
	branchesSpec := config.RefSpec(fmt.Sprintf("+refs/heads/*:refs/remotes/%s/*", remote.Config().Name))
	return &git.FetchOptions{RemoteName: remote.Config().Name, Depth: 10, RefSpecs: []config.RefSpec{headRefSpec, branchesSpec}}
}

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
	remotes, err := b.repo.Remotes()
	if err != nil {
		if Debug {
			log.Printf("Cannot get remotes of git repository %s. Error: %s", b.repoPath, err)
		}
		//Not critical
		return nil
	}
	//This should never happen
	if len(remotes) == 0 {
		if Debug {
			log.Printf("Got no remotes of git repository %s", b.repoPath)
		}
		//Not critical
		return nil
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
