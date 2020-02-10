package launcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/acarl005/stripansi"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"tfChek/git"
	"tfChek/github"
	"tfChek/misc"
	"tfChek/storer"
)

type RunShTask struct {
	Name       string
	Id         int
	Command    string
	Args       []string
	ExtraEnv   map[string]string
	StateLock  string
	Context    context.Context
	Status     TaskStatus
	Socket     chan *websocket.Conn
	out, err   io.Reader
	in         io.Writer
	inR        io.ReadCloser
	outW, errW io.WriteCloser
	save       bool
	GitOrigins []string
	sink       bytes.Buffer
	authors    []string
}

/**
This method has to return the git manager of the git repository, which contains executable script run.sh
*/
func (rst *RunShTask) getFirstGitManager() (git.Manager, error) {
	managers, err := rst.getGitManagers()
	if err != nil {
		return nil, err
	}
	if len(*managers) == 0 {
		msg := fmt.Sprintf("No git managers vere returned for task %d", rst.Id)
		if Debug {
			log.Print(msg)
		}
		return nil, errors.New(msg)
	}
	return (*managers)[0], nil
}

func (rst *RunShTask) GetOrigins() *[]string {
	return &rst.GitOrigins
}

func (rst *RunShTask) SetAuthors(authors []string) {
	rst.authors = authors
}

func (rst *RunShTask) GetAuthors() *[]string {
	return &rst.authors
}

func (rst *RunShTask) GetExtraEnv() *map[string]string {
	return &rst.ExtraEnv
}

func (rst *RunShTask) Register() error {
	if rst.Status == misc.OPEN {
		rst.Status = misc.REGISTERED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled registered, beacuse it is not open. Please make get request. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) Schedule() error {
	if rst.Status == misc.REGISTERED {
		rst.Status = misc.SCHEDULED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled because it has been not registered. Please wait for a webhook. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) Start() error {
	if rst.Status < misc.STARTED {
		if Debug {
			log.Printf("Start of task %s", rst.Name)
		}
		rst.Status = misc.STARTED
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be Started because it is not in scheduled state. Current state number is %d", rst.Status)}
	}
}
func (rst *RunShTask) Done() error {
	if rst.Status == misc.STARTED {
		rst.Status = misc.DONE
		fgm, err := rst.getFirstGitManager()
		if err != nil {
			if Debug {
				log.Printf("Cannot get first Git manager. Error: %s", err)
			}
			return err
		}
		manager := github.GetManager(fgm.GetRemote())
		if manager != nil {
			c := manager.GetChannel()
			o := rst.GetOut()
			if o == "" {
				o = misc.NOOUTPUT
			}
			data := github.NewTaskResult(rst.Id, true, &o, rst.GetAuthors())
			c <- data
		}
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be done, because it has been not Started. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) Fail() error {
	if rst.Status == misc.STARTED {
		rst.Status = misc.FAILED
		fgm, err := rst.getFirstGitManager()
		if err != nil {
			if Debug {
				log.Printf("Cannot get first Git manager. Error: %s", err)
			}
			return err
		}
		manager := github.GetManager(fgm.GetRemote())
		if manager != nil {
			c := manager.GetChannel()
			o := rst.GetOut()
			if o == "" {
				o = misc.NOOUTPUT
			}
			data := github.NewTaskResult(rst.Id, false, &o, rst.GetAuthors())
			c <- data
		}
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be failed, because it has been not Started. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) TimeoutFail() error {
	if rst.Status == misc.STARTED {
		rst.Status = misc.TIMEOUT
		fgm, err := rst.getFirstGitManager()
		if err != nil {
			if Debug {
				log.Printf("Cannot get first Git manager. Error: %s", err)
			}
			return err
		}
		manager := github.GetManager(fgm.GetRemote())
		if manager != nil {
			c := manager.GetChannel()
			o := rst.GetOut()
			if o == "" {
				o = misc.NOOUTPUT
			}
			data := github.NewTaskResult(rst.Id, false, &o, rst.GetAuthors())
			c <- data
		}
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be timed out, because it has been not Started. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) GetOut() string {
	cleanOut := stripansi.Strip(rst.sink.String())
	return cleanOut
}

func (rst *RunShTask) ForceFail() {
	rst.Status = misc.FAILED
}

func (rst *RunShTask) GetStatus() TaskStatus {
	return rst.Status
}

func (rst *RunShTask) SetStatus(status TaskStatus) {
	rst.Status = status
}

func (rst *RunShTask) GetId() int {
	return rst.Id
}

func (rst *RunShTask) setId(id int) {
	rst.Id = id
}

func (rst *RunShTask) SyncName() string {
	if rst.StateLock != "" {
		return rst.StateLock
	}
	return rst.Command
}

func (rst *RunShTask) GetStdOut() io.Reader {
	return rst.out
}

func (rst *RunShTask) GetStdErr() io.Reader {
	return rst.err
}

func (rst *RunShTask) GetStdIn() io.Writer {
	return rst.in
}

func convertToSSHGitUrl(url string) string {
	//TODO: implement this
	//TODO: write test here
	return url
}

func (rst *RunShTask) prepareGit() error {
	//create RUNSH_APTH here for launching run.sh

	//Prehaps here I have to convert git url to ssh url (in form of "git@github.com:...")
	gms, err := rst.getGitManagers()
	if err != nil {
		return err
	}

	for gi, gitman := range *gms {
		if Debug {
			log.Printf("Preparing Git repo %d (%s) of %d", gi+1, gitman.GetRemote(), len(rst.GitOrigins))
		}
		if gitman.IsCloned() {
			err := gitman.Open()
			if err != nil {
				log.Printf("Cannot open git repository. Error: %s", err)
				return err
			}
		} else {
			path := gitman.GetPath()
			_, err := os.Stat(path)
			if os.IsNotExist(err) {
				err := os.MkdirAll(path, 0755)
				if err != nil {
					log.Printf("Cannot create directory for git repository. Error: %s", err)
					return err
				}
			}
			err = gitman.Clone()
			if err != nil {
				log.Printf("Cannot clone repository. Error: %s", err)
				return err
			}
		}
		branch := fmt.Sprintf("%s%d", misc.TaskPrefix, rst.Id)
		err = gitman.Checkout(branch)
		if err != nil {
			log.Printf("Cannot checkout branch ")
			return err
		}
		err = gitman.Pull()
		if err != nil {
			log.Printf("Cannot pull changes. Error: %s", err)
			return err
		}

	}
	return nil
}

func (rst *RunShTask) getGitManagers() (*[]git.Manager, error) {
	if len(rst.GitOrigins) == 0 {
		return nil, errors.New(fmt.Sprintf("Cannot obtain a git manager. Task id %d contains no git remotes"))
	} else {
		var managers []git.Manager
		for _, gurl := range rst.GitOrigins {
			sshurl := convertToSSHGitUrl(gurl)
			gitman, err := git.GetManager(sshurl, rst.StateLock)
			if err != nil {
				return nil, err
			}
			managers = append(managers, gitman)
		}
		return &managers, nil
	}
}

func (rst *RunShTask) generateRunshPath() (string, error) {
	gms, err := rst.getGitManagers()
	if err != nil {
		if Debug {
			log.Printf("Generation of RUNSH_PATH failed. Error: %s", err)
			return "", err
		}
	}
	var paths []string
	for _, gitman := range *gms {
		paths = append(paths, gitman.GetPath())
	}
	return strings.Join(paths, ":"), nil
}

func (rst *RunShTask) prepareGitHub() error {
	gitManagers, err := rst.getGitManagers()
	if err != nil {
		//Add Debug output here
		if Debug {
			log.Printf("Cannot prepare GitHub, because Git manager are not available. Error: %s", err)
		}
		return err
	}
	repoOwner := viper.GetString(misc.RepoOwnerKey)
	token := viper.GetString(misc.TokenKey)
	for _, gm := range *gitManagers {
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

func (rst *RunShTask) Run() error {
	if rst.Status != misc.SCHEDULED {
		return errors.New("cannot run unscheduled task")
	}
	//Perform git routines
	err := rst.prepareGit()
	if err != nil {
		log.Printf("Cannot prepare git repositories. Error: %s", err)
		rst.ForceFail()
		return err
	}
	err = rst.prepareGitHub()
	if err != nil {
		log.Printf("Cannot prepare GitHub repositories. Error: %s", err)
		rst.ForceFail()
		return err
	}
	defer rst.outW.Close()
	defer rst.errW.Close()
	defer rst.inR.Close()
	//Get working directory
	gitman, err := rst.getFirstGitManager()
	if err != nil {
		if Debug {
			log.Printf("Failed to obtain git manager. Error: %s", err)
		}
		return err
	}
	cwd := gitman.GetPath()
	//Copy certificates to the landscape directory of the git repository which contains run.sh. Usually it is the very first one
	err = deliverCerts(cwd)
	if err != nil {
		log.Printf("Warning! Task id %d can fail, because certificates delivery failed. Error: %s", rst.Id, err)
	}
	err = deliverLambdas(cwd)
	if err != nil {
		log.Printf("Warning! Task id %d can fail, because lambdas delivery failed. Error: %s", rst.Id, err)
	}
	log.Printf("Task id: %d working directory: %s", rst.Id, cwd)
	//Get environment
	sysenv := os.Environ()
	//Inject extra vars
	if d, ok := rst.Context.Value(misc.EnvVarsKey).(map[string]string); ok {
		for k, v := range d {
			sysenv = append(sysenv, fmt.Sprintf("%s=%s", k, v))
		}
	}

	//Inject RUNSH_PATH (important!)
	pshp, err := rst.generateRunshPath()
	if err != nil {
		log.Printf("Warning! Failed to generate RUNSH_PATH. Error: %s", err)
	} else {
		sysenv = append(sysenv, fmt.Sprintf("%s=%s", misc.RunShPathEnvVar, pshp))
	}

	//Disable tfChek notification to avoid recursion
	sysenv = append(sysenv, fmt.Sprintf("%s=%s", misc.NotifyTfChekEnvVar, "false"))

	log.Printf("Task id: %d environment: %s", rst.Id, sysenv)

	//Save command execution output

	mw := io.MultiWriter(rst.outW, &rst.sink)

	log.Printf("Running command '%s %s' and waiting for it to finish...", rst.Command, strings.Join(rst.Args, " "))
	command := exec.CommandContext(rst.Context, rst.Command, rst.Args...)
	command.Dir = cwd
	command.Env = sysenv
	command.Stdout = mw
	command.Stderr = mw
	//command.Stdin = rst.inR
	command.Stdin = nil
	//Ugly but I did not found a better place
	if rst.save {
		out, err := storer.GetTaskFileWriteCloser(rst.Id)
		if err != nil {
			log.Printf("Save to file for task %d is disabled. Error: %s", rst.Id, err)
		} else {
			ow := io.MultiWriter(mw, out)
			eow := io.MultiWriter(mw, out)
			command.Stdout = ow
			command.Stderr = eow
		}
	}

	//I will write nothing to the command
	//So closing stdin immediately
	err = rst.inR.Close()
	if err != nil {
		log.Printf("Cannot close stdin for task id: %d", rst.Id)
	}

	err = rst.Start()
	if err != nil {
		log.Printf("Cannot change task state. Error: %s", err)
	}
	err = command.Run()
	if err != nil {
		if err.Error() == "context deadline exceeded" {
			log.Printf("Command timed out error: %v", err)
			err = rst.TimeoutFail()
			if err != nil {
				log.Printf("Cannot change task state. Error: %s", err)
			}
		} else {
			log.Printf("Command finished with error: %v", err)
			err = rst.Fail()
			if err != nil {
				log.Printf("Cannot change task state. Error: %s", err)
			}
		}
	} else {
		err = rst.Done()
		if err != nil {
			log.Printf("Cannot change task state. Error: %s", err)
		}
		log.Println("Command completed successfully")
	}
	return err
}
