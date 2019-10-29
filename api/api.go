package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"tfChek/git"
	"tfChek/launcher"
	"tfChek/misc"
	"time"
)
import "gopkg.in/go-playground/webhooks.v5/github"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func RunShWebsocket(w http.ResponseWriter, r *http.Request) {
	tm := launcher.GetTaskManager()
	vars := mux.Vars(r)
	id := vars["Id"]
	if id == "" {
		log.Println("Cannot run with no id")
		w.WriteHeader(404)
		_, err := w.Write([]byte("Cannot run with no id"))
		if err != nil {
			log.Printf("Cannot run task id %s Error: %s", id, err)
		}
		return
	}
	taskId, err := strconv.Atoi(id)
	if err != nil {
		log.Println("Cannot convert parse task id")
		//TODO: add error handling
	}

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		//TODO: log error and handle it
	}
	log.Println("Client connected to run.sh Env websocket")
	bt := tm.Get(taskId)
	if bt == nil {
		log.Printf("Cannot find task by id: %d", taskId)
		w.WriteHeader(404)
		err := ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Cannot find task by id: %d", taskId)))
		if err != nil {
			log.Printf("Cannot find task by id: %d Error: %s", taskId, err)
		}
		return
	}
	errc := make(chan error)
	err = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Task (id: %d) Status: %s", bt.GetId(), launcher.GetStatusString(bt.GetStatus()))))
	if err != nil {
		log.Println(err)
	}
	lock := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go writeToWS(bt.GetStdOut(), ws, errc, lock, wg)
	go writeToWS(bt.GetStdErr(), ws, errc, lock, wg)
	go func(ws *websocket.Conn, errc <-chan error) {
		e := <-errc
		if e != nil {
			err = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Task (id: %d) Status: %s Error: %s", bt.GetId(), launcher.GetStatusString(bt.GetStatus()), e)))
			if err != nil {
				log.Println(err)
			}
		} else {
			err = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Task (id: %d) Status: %s", bt.GetId(), launcher.GetStatusString(bt.GetStatus()))))
			if err != nil {
				log.Println(err)
			}
		}

	}(ws, errc)
	wg.Wait()
	close(errc)
}

func writeToWS(in io.Reader, ws *websocket.Conn, errc chan<- error, lock *sync.Mutex, wg *sync.WaitGroup) {
	bufRdr := bufio.NewReader(in)
	for {
		line, _, err := bufRdr.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			errc <- err
			log.Printf("Cannot read stream. Error: %s", err)
			break
		}
		lock.Lock()
		err = ws.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			errc <- err
			log.Printf("Cannot write to websocket. Error: %s", err)
		}
		lock.Unlock()
	}
	wg.Done()
}

func RunShEnv(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	env := v["Env"]
	layer := ""
	envVars := make(map[string]string)
	envVars["TFRESDIF_NOPB"] = "true"

	runsh(w, r, env, layer, time.Duration(viper.GetInt(misc.TimeoutKey))*time.Second, &envVars)
}

func Cancel(w http.ResponseWriter, r *http.Request) {
	tm := launcher.GetTaskManager()
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		log.Println("Cannot cancel with no id")
		w.WriteHeader(404)
		_, err := w.Write([]byte("Cannot cancel with no id"))
		if err != nil {
			log.Printf("Cannot cancel Error: %s", err)
		}
		return
	}
	taskId, err := strconv.Atoi(id)
	if err != nil {
		log.Println("Cannot convert parse task id")
		w.WriteHeader(400)
		_, err := w.Write([]byte(fmt.Sprintf("Cannot convert parse task id %s", id)))
		if err != nil {
			log.Printf("Cannot cancel Error: %s", err)
		}

		return
	}
	bt := tm.Get(taskId)
	if bt == nil {
		log.Printf("Cannot find task by id: %d", taskId)
		w.WriteHeader(404)
		_, err := w.Write([]byte(fmt.Sprintf("Cannot find task by id: %d", taskId)))
		if err != nil {
			log.Printf("Cannot find task by id: %d Error: %s", taskId, err)
		}
		return
	}
	err = tm.Cancel(bt.GetId())
	if err != nil {
		log.Printf("Cannot cancel task by id: %d Error: %s", taskId, err)
		w.WriteHeader(404)
		_, err := w.Write([]byte(fmt.Sprintf("Cannot cancel task by id: %d", taskId)))
		if err != nil {
			log.Printf("Cannot find cancel by id: %d Error: %s", taskId, err)
		}
		return
	}
	w.WriteHeader(202)
}

func RunShEnvLayer(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	env := v["Env"]
	layer := v["Layer"]
	envVars := make(map[string]string)
	envVars["TFRESDIF_NOPB"] = "true"
	runsh(w, r, env, layer, time.Duration(viper.GetInt(misc.TimeoutKey))*time.Second, &envVars)
}

func RunShWebHook(w http.ResponseWriter, r *http.Request) {
	tm := launcher.GetTaskManager()
	hook, _ := github.New(github.Options.Secret(viper.GetString(misc.WebHookSecretKey)))
	payload, err := hook.Parse(r, github.PushEvent)
	if err != nil {
		if err == github.ErrEventNotFound {
			// ok event wasn't one of the ones asked to be parsed
			errmsg := fmt.Sprintf("Unknown event. Error: %s", err)
			log.Println(errmsg)
			w.WriteHeader(404)
			_, err := w.Write([]byte(errmsg))
			if err != nil {
				log.Printf("Cannot post message '%s' Error: %s", errmsg, err)
			}
			return
		} else {
			if e, ok := err.(*json.SyntaxError); ok {
				log.Printf("syntax error at byte offset %d", e.Offset)
			}
			errmsg := fmt.Sprintf("Got error %s", err)
			log.Println(errmsg)
			w.WriteHeader(400)
			_, err := w.Write([]byte(errmsg))
			if err != nil {
				log.Printf("Cannot post message '%s' Error: %s", errmsg, err)
			}
			return
		}
	}
	switch payload.(type) {
	case github.PushPayload:
		pushPayload := payload.(github.PushPayload)
		if pushPayload.Created {
			//branchName := plumbing.NewBranchReferenceName(pushPayload.Ref).Short()
			branchName := strings.ReplaceAll(pushPayload.Ref, "refs/heads/", "")
			matched, err := regexp.Match("^"+misc.TaskPrefix+"[0-9]+", []byte(branchName))
			if err != nil {
				log.Printf("Cannot match against regex")
			}
			if matched {
				log.Printf("I have to process this event")
				chunks := strings.Split(branchName, "-")
				taskId, err := strconv.Atoi(chunks[1])
				if err != nil {
					errmsg := fmt.Sprintf("Cannot parse task id %s", chunks[1])
					log.Println(errmsg)
					w.WriteHeader(400)
					_, err := w.Write([]byte(errmsg))
					if err != nil {
						log.Printf("Cannot post message '%s' Error: %s", errmsg, err)
					}
					return
				} else {

					//Prepare git directory
					task := tm.Get(taskId)
					if task == nil {
						log.Printf("Cannot find task by id: %d", taskId)
						w.WriteHeader(404)
						errmsg := fmt.Sprintf("Cannot find task by id: %d", taskId)
						_, err := w.Write([]byte(errmsg))
						if err != nil {
							log.Printf("Cannot post message '%s' Error: %s", errmsg, err)
						}
						return
					}
					gitMan, err := git.GetManager(pushPayload.Repository.GitURL, task.SyncName())
					if err != nil {
						log.Printf("Cannot get Git manager")
					}
					if gaTask, ok := task.(launcher.GitHubAwareTask); ok {
						gaTask.SetGitManager(gitMan)
						authors := fetch_authors(&pushPayload)
						gaTask.SetAuthors(*authors)
					}
					err = tm.LaunchById(taskId)
					if err != nil {
						errmsg := fmt.Sprintf("Cannot launch task id %d. Error: %s", taskId, err)
						log.Println(errmsg)
						w.WriteHeader(400)
						_, err = w.Write([]byte(errmsg))
						if err != nil {
							log.Printf("Cannot post message '%s' Error: %s", errmsg, err)
						}
					} else {
						w.WriteHeader(202)
					}
				}
			} else {
				w.WriteHeader(200)
			}
		}
		if pushPayload.Deleted {
			log.Printf("Branch %s has been deleted", pushPayload.Ref)
		}
		log.Printf("Processed webhook of branch %s", pushPayload.Ref)
	}
}

func fetch_authors(payload *github.PushPayload) *[]string {
	author_usernames := make(map[string]struct{}, 0)
	if payload != nil {
		for _, commit := range payload.Commits {
			author_usernames[commit.Author.Username] = struct{}{}
		}
	}
	res := make([]string, len(author_usernames))
	n := 0
	for i := range author_usernames {
		res[n] = i
		n++
	}
	return &res
}

func runsh(w http.ResponseWriter, r *http.Request, env, layer string, timeout time.Duration, envVars *map[string]string) {
	tm := launcher.GetTaskManager()
	w.Header().Set("Content-Type", "application/json")
	var cmd launcher.RunShCmd
	err := r.ParseForm()
	if err != nil {
		em := fmt.Sprintf("Cannot parse request. Error: %s", err.Error())
		_, e := w.Write([]byte(em))
		if e != nil {
			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
		}
	}
	targets := r.Form["target"]
	omit := r.FormValue("omit")
	all := r.Form.Get("all")
	no := r.Form.Get("no")
	yes := r.Form.Get("yes")
	cmd = launcher.RunShCmd{Layer: layer, Env: env, All: all == "true", Omit: omit == "true", Targets: targets, No: no == "true", Yes: yes == "true"}
	ctx, cancel := context.WithTimeout(
		context.WithValue(
			context.Background(),
			misc.EnvVarsKey, envVars),
		timeout)
	bt, err := tm.AddRunSh(cmd, ctx)
	if err != nil {
		em := fmt.Sprintf("Cannot create background task. Error: %s", err.Error())
		_, e := w.Write([]byte(em))
		if e != nil {
			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
			return
		}
	} else {
		if viper.GetBool("debug") {
			log.Printf("Task %d has been added", bt.GetId())
		}
	}
	if bt != nil {
		err = tm.RegisterCancel(bt.GetId(), cancel)
		if err != nil {
			log.Printf("Cannot register cancel function. Task (id: %d) will be impossible to cancel. Error: %s", bt.GetId(), err)
		}
		w.WriteHeader(201)
		_, err = w.Write([]byte(strconv.Itoa(bt.GetId())))
		if err != nil {
			log.Printf("Cannot write response. Error: %s", err)
		}
	} else {
		log.Print("Cannot register cancel function of nil task. This should never happen!")
		w.WriteHeader(404)
		errmsg := "Cannot process nil task"
		_, err = w.Write([]byte(errmsg))
		if err != nil {
			log.Printf("Cannot post message '%s'. Error: %s", errmsg, err)
		}
	}

}
