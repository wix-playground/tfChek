package launcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/acarl005/stripansi"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"github.com/wix-system/tfChek/git"
	"github.com/wix-system/tfChek/github"
	"github.com/wix-system/tfChek/misc"
	"github.com/wix-system/tfChek/storer"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

type RunShTask struct {
	Name      string
	Id        int
	Command   string
	Args      []string
	ExtraEnv  map[string]string
	StateLock string
	Context   context.Context
	Status    TaskStatus
	Socket    chan *websocket.Conn
	//These are not needed anymore here.
	//out, err    io.Reader
	//in          io.Writer
	//inR         io.ReadCloser
	//outW, errW  io.WriteCloser

	//This should be always on
	//Remove this field in the future
	save        bool
	GitOrigins  []string
	sink        bytes.Buffer
	authors     []string
	subscribers []chan TaskStatus
}

/**
This method has to return the git manager of the git repository, which contains executable script run.sh
*/
//Deprecated
func (rst *RunShTask) getFirstGitManager() (git.Manager, error) {
	managers, err := rst.getGitManagers()
	if err != nil {
		return nil, err
	}
	if len(managers) == 0 {
		misc.Debugf("No git managers were returned for task %d", rst.Id)
		return nil, fmt.Errorf("No git managers were returned for task %d", rst.Id)
	}
	var mgr git.Manager = nil
	for url, m := range managers {
		p := m.GetPath()
		runshPath := path.Join(p, misc.RunshExe)
		info, err := os.Stat(runshPath)
		if os.IsNotExist(err) {
			misc.Debugf("repository %s does not contain %s executable. Trying next one...", p, misc.RunshExe)
			continue
		}
		if info.IsDir() {
			misc.Debugf("%s cannot be directory, it should be an executable file. Trying next one...", runshPath)
			continue
		}
		if info.Mode()&0111 == 0 {
			misc.Debugf("file %s should be executable. Trying next one...")
			continue
		} else {
			mgr = m
			misc.Debugf("found %s executable in %s repository. Using it...", runshPath, url)
		}
	}
	if mgr == nil {
		for url, m := range managers {
			p := m.GetPath()
			wtfPath := path.Join(p, misc.WtfExe)
			info, err := os.Stat(wtfPath)
			if os.IsNotExist(err) {
				misc.Debugf("repository %s does not contain %s executable. Trying next one...", wtfPath, misc.WtfExe)
				continue
			}
			if info.IsDir() {
				misc.Debugf("%s cannot be directory, it should be an executable file. Trying next one...", wtfPath)
				continue
			}
			if info.Mode()&0111 == 0 {
				misc.Debugf("file %s should be executable. Trying next one...")
				continue
			} else {
				mgr = m
				misc.Debugf("found %s executable in %s repository. Using it...", wtfPath, url)
			}
		}
	}
	if mgr == nil {
		return nil, fmt.Errorf("failed to find repository with %s executable", misc.RunshExe)
	}
	return mgr, nil
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
		rst.notifySubscribers()
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled registered, beacuse it is not open. Please make get request. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) Schedule() error {
	//TODO: make scheduling locked by webhooks using waiting group
	if rst.Status == misc.REGISTERED {
		rst.Status = misc.SCHEDULED
		rst.notifySubscribers()
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be scheduled because it has been not registered. Please wait for a webhook. Current state number is %d", rst.Status)}
	}
}

func (rst *RunShTask) Start() error {
	if rst.Status < misc.STARTED {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Start of task %s", rst.Name)
		}
		rst.Status = misc.STARTED
		rst.notifySubscribers()
		return nil
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be Started because it is not in scheduled state. Current state number is %d", rst.Status)}
	}
}
func (rst *RunShTask) Done() error {
	if rst.Status == misc.STARTED {
		rst.Status = misc.DONE
		rst.notifySubscribers()
		gitManagers, err := rst.getGitManagers()
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
				o := rst.GetCleanOut()
				if o == "" {
					o = misc.NOOUTPUT
				}
				data := github.NewTaskResult(rst.Id, true, &o, rst.GetAuthors())
				c <- data
			}
		}
	} else {
		return &StateError{msg: fmt.Sprintf("Task cannot be done, because it has been not Started. Current state number is %d", rst.Status)}
	}
	return nil
}

func (rst *RunShTask) Fail() error {
	if rst.Status == misc.STARTED {
		rst.Status = misc.FAILED
		rst.notifySubscribers()
		fgm, err := rst.getFirstGitManager()
		if err != nil {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Cannot get first Git manager. Error: %s", err)
			}
			return err
		}
		manager := github.GetManager(fgm.GetRemote())
		if manager != nil {
			c := manager.GetChannel()
			o := rst.GetCleanOut()
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
		rst.notifySubscribers()
		fgm, err := rst.getFirstGitManager()
		if err != nil {
			if viper.GetBool(misc.DebugKey) {
				log.Printf("Cannot get first Git manager. Error: %s", err)
			}
			return err
		}
		manager := github.GetManager(fgm.GetRemote())
		if manager != nil {
			c := manager.GetChannel()
			o := rst.GetCleanOut()
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

//GetCleanOut function returns output without ANSI characters (Non colored out)
func (rst *RunShTask) GetCleanOut() string {
	cleanOut := stripansi.Strip(rst.sink.String())
	return cleanOut
}

func (rst *RunShTask) ForceFail() {
	rst.Status = misc.FAILED
	rst.notifySubscribers()
}

func (rst *RunShTask) GetStatus() TaskStatus {
	return rst.Status
}

func (rst *RunShTask) SetStatus(status TaskStatus) {
	rst.Status = status
}

func (rst *RunShTask) Subscribe() chan TaskStatus {
	sts := make(chan TaskStatus, 2)
	sts <- rst.Status
	//Add channel to subscribers if the task is active
	if !IsCompleted(rst) {
		rst.subscribers = append(rst.subscribers, sts)
	}
	return sts
}

func IsCompleted(t Task) bool {

	if t.GetStatus() == misc.DONE || t.GetStatus() == misc.FAILED || t.GetStatus() == misc.TIMEOUT {
		return true
	} else {
		return false
	}
}

func (rst *RunShTask) notifySubscribers() {
	for _, sc := range rst.subscribers {
		sc <- rst.Status
		//Let the reader do this
		//if rst.Status == misc.DONE || rst.Status == misc.FAILED || rst.Status == misc.TIMEOUT {
		//	close(sc)
		//}
	}
	//Remove all subscribers after notification that task is completed
	if IsCompleted(rst) {
		rst.subscribers = nil
	}
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

//Deprecated
func (rst *RunShTask) GetStdOut() io.Reader {
	return &rst.sink
}

//Deprecated
func (rst *RunShTask) GetStdErr() io.Reader {
	return &rst.sink
}

//Deprecated
func (rst *RunShTask) GetStdIn() io.Writer {
	return nil
}

func (rst *RunShTask) prepareGit() error {
	//create RUNSH_APTH here for launching run.sh
	branch := fmt.Sprintf("%s%d", misc.TaskPrefix, rst.Id)
	//Prehaps here I have to convert git url to ssh url (in form of "git@github.com:...")
	gms, err := rst.getGitManagers()
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
			misc.Debugf("Waiting for a webhook to come from %s repo for a task %d", manager.GetRemote(), rst.Id)
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
	misc.Debugf("preparation of git repositories succesfully finished for task %d", rst.Id)
	return nil
}

func (rst *RunShTask) AddWebhookLocks() error {
	managers, err := rst.getGitManagers()
	if err != nil {
		return fmt.Errorf("cannot get git managers for task %d %w", rst.Id, err)
	}
	branch := fmt.Sprintf("%s%d", misc.TaskPrefix, rst.Id)
	for _, m := range managers {
		//frn, err := git.GetFullRepoName(m.GetRemote())
		//if err != nil {
		//	return fmt.Errorf("cannot get full repository name of %s %w",m.GetRemote(), err)
		//}
		//if frn == repoFullName {
		err := m.RegisterWebhookLock(branch)
		if err != nil {
			return fmt.Errorf("cannot add webhook lock for task %d at %s, %w", rst.Id, m.GetPath(), err)
		}
		//}
	}
	return nil
}

func (rst *RunShTask) UnlockWebhookRepoLock(fullName string) error {
	//TODO: It is better to return a map here
	managers, err := rst.getGitManagers()
	if err != nil {
		return fmt.Errorf("cannot get git managers for task %d %w", rst.Id, err)
	}
	branch := fmt.Sprintf("%s%d", misc.TaskPrefix, rst.Id)
	for _, m := range managers {
		frn, err := git.GetFullRepoName(m.GetRemote())
		if err != nil {
			return fmt.Errorf("cannot get full repository name of %s %w", m.GetRemote(), err)
		}
		if frn == fullName {
			err := m.UnlockWebhookLock(branch)
			if err != nil {
				return fmt.Errorf("cannot add webhook lock for task %d at %s, %w", rst.Id, m.GetPath(), err)
			}
			misc.Debugf("successfully unlocked task %d at %s", rst.Id, m.GetPath())
		}
	}
	return nil
}

// Returns mapping of git manager to its URL
func (rst *RunShTask) getGitManagers() (map[string]git.Manager, error) {
	if len(rst.GitOrigins) == 0 {
		return nil, errors.New(fmt.Sprintf("Cannot obtain a git manager. Task id %d contains no git remotes", rst.Id))
	} else {
		managers := make(map[string]git.Manager)
		for _, gurl := range rst.GitOrigins {
			gitman, err := git.GetManager(gurl, rst.StateLock)
			if err != nil {
				return nil, err
			}
			managers[gurl] = gitman
		}
		return managers, nil
	}
}

func (rst *RunShTask) generateRunshPath() (string, error) {
	gms, err := rst.getGitManagers()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Generation of RUNSH_PATH failed. Error: %s", err)
			return "", err
		}
	}
	var paths []string
	for _, gitman := range gms {
		paths = append(paths, gitman.GetPath())
	}
	return strings.Join(paths, ":"), nil
}

func (rst *RunShTask) prepareGitHub() error {
	gitManagers, err := rst.getGitManagers()
	if err != nil {
		//Add Debug output here
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Cannot prepare GitHub, because Git manager are not available. Error: %s", err)
		}
		return err
	}
	repoOwner := viper.GetString(misc.RepoOwnerKey)
	token := viper.GetString(misc.TokenKey)
	for _, gm := range gitManagers {
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

func logTaskEnv(tid int, env *[]string) {
	if viper.GetBool(misc.DebugKey) {
		var builder strings.Builder
		builder.Grow(50)
		fmt.Fprintf(&builder, "Task id: %d enjvironment:\n", tid)
		for i, s := range *env {
			kv := strings.SplitN(s, "=", 2)
			ms := misc.MaskEnvValue(kv[0], kv[1])
			fmt.Fprintf(&builder, "\t#%d\t%s = %s\n", i, kv[0], ms)
		}
		misc.Debug(builder.String())
	}
}

func (rst *RunShTask) Run() error {
	if rst.Status != misc.SCHEDULED {
		return errors.New("cannot run unscheduled task")
	}
	//Prepare github first
	err := rst.prepareGitHub()
	if err != nil {
		log.Printf("Cannot prepare GitHub repositories. Error: %s", err)
		rst.ForceFail()
		return err
	}
	//Perform git routines
	err = rst.prepareGit()
	if err != nil {
		log.Printf("Cannot prepare git repositories. Error: %s", err)
		rst.ForceFail()
		return err
	}

	//defer rst.outW.Close()
	//defer rst.errW.Close()
	//defer rst.inR.Close()
	//Get working directory
	gitman, err := rst.getFirstGitManager()
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Failed to obtain git manager. Error: %s", err)
		}
		return err
	}
	cwd := gitman.GetPath()
	//Copy certificates to the landscape directory of the git repository which contains run.sh. Usually it is the very first one
	//TODO: Remove it after WTF integration
	err = deliverCerts(cwd)
	if err != nil {
		log.Printf("Warning! Task id %d can fail, because certificates delivery failed. Error: %s", rst.Id, err)
	}
	//TODO: Remove it after WTF integration
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

	//This is disabled by now, because there are multiple credentials for different AWS resources
	//Add AWS credentials for terraform
	//if viper.GetString(misc.AWSAccessKey) != "" && viper.GetString(misc.AWSSecretKey) != "" {
	//	sysenv = append(sysenv, fmt.Sprintf("%s=%s", misc.AwsAccessKeyVar, viper.GetString(misc.AWSAccessKey)))
	//	sysenv = append(sysenv, fmt.Sprintf("%s=%s", misc.AwsSecretKeyVar, viper.GetString(misc.AWSSecretKey)))
	//}

	logTaskEnv(rst.Id, &sysenv)

	//Save command execution output

	//mw := io.MultiWriter(rst.outW, &rst.sink)

	log.Printf("Running command '%s %s' and waiting for it to finish...", rst.Command, strings.Join(rst.Args, " "))
	command := exec.CommandContext(rst.Context, rst.Command, rst.Args...)
	command.Dir = cwd
	command.Env = sysenv
	command.Stdout = &rst.sink
	command.Stderr = &rst.sink
	//command.Stdin = rst.inR
	command.Stdin = nil
	//Ugly but I did not found a better place
	if rst.save {
		out, err := storer.GetTaskFileWriteCloser(rst.Id)
		if err != nil {
			log.Printf("Save to file for task %d is disabled. Error: %s", rst.Id, err)
		} else {
			ow := io.MultiWriter(command.Stdout, out)
			eow := io.MultiWriter(command.Stderr, out)
			command.Stdout = ow
			command.Stderr = eow
		}
	}

	//I will write nothing to the command
	//So closing stdin immediately
	//err = rst.inR.Close()
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
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Command completed successfully for task %d", rst.Id)
		}
	}
	upload2s3(rst.Id, rst.Status)
	return err
}

func upload2s3(id int, status TaskStatus) {
	bucketName := viper.GetString(misc.S3BucketName)
	suffix := GetStatusString(status)
	err := storer.S3UploadTaskWithSuffix(bucketName, id, &suffix)
	if err != nil {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Failed to upload output of the task %d Error: %s", id, err)
		}
	} else {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Output of the task %d has been successfully stored at S3 bucket", id)
		}
	}
}
