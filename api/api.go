package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
	//Debug websocket
	//log.Printf("Request %s headers:", r.URL.String())
	//for k, v := range r.Header {
	//	log.Printf("\tHeader field %q, Value %q\n", k, v)
	//}
	//End Debug

	tm := launcher.GetTaskManager()
	vars := mux.Vars(r)
	id := vars["Id"]
	if id == "" {
		erm := "Cannot run with no id"
		log.Println(erm)
		w.Header().Add("Reason", erm)
		w.WriteHeader(404)
		_, err := w.Write([]byte(erm))
		if err != nil {
			log.Printf(erm+" Error: %s", err)
		}
		return
	}
	taskId, err := strconv.Atoi(id)
	if err != nil {
		erm := fmt.Sprintf("Cannot convert parse task id %s Error: %s", id, err)
		log.Println(erm)
		w.Header().Add("Reason", erm)
		w.WriteHeader(500)
		_, err := w.Write([]byte(erm))
		if err != nil {
			log.Printf("Cannot post error message '%s' Error: %s", erm, err)
		}
		return
	}
	bt := tm.Get(taskId)
	if bt == nil {
		erm := fmt.Sprintf("Cannot find task by id: %d", taskId)
		log.Println(erm)
		w.Header().Add("Reason", erm)
		w.WriteHeader(404)
		_, err := w.Write([]byte(erm))
		if err != nil {
			log.Printf("Cannot post error message '%s' Error: %s", erm, err)
		}
		//err := ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Cannot find task by id: %d", taskId)))
		//if err != nil {
		//	log.Printf("Cannot find task by id: %d Error: %s", taskId, err)
		//}
		return
	}
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		erm := fmt.Sprintf("Cannot upgrade connection to use websocket. Error: %s", err)
		log.Println(erm)
		w.WriteHeader(500)
		w.Header().Add("Reason", erm)
		_, err := w.Write([]byte(erm))
		if err != nil {
			log.Printf("Cannot post error message '%s' Error: %s", erm, err)
		}
		return
	}
	log.Println("Client connected to run.sh Env websocket")

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

func RunShPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	msg, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Cannot read body message")
		handleReqErr(err, w)
		return
	}
	hash, err := misc.GetPayloadHash(msg, misc.PAYLOADHASH_SHA512)
	if err != nil {
		w.WriteHeader(http.StatusNotImplemented)
		em := fmt.Sprintf("Cannot compute hash of the message. Error: %s", err.Error())
		_, e := w.Write([]byte(em))
		if e != nil {
			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
		}
		return
	}
	smsg := string(msg)
	dec := json.NewDecoder(strings.NewReader(smsg))
	dec.DisallowUnknownFields()
	var rgp launcher.RunSHLaunchConfig
	//for dec.More() {
	err = dec.Decode(&rgp)
	if err != nil {
		handleReqErr(err, w)
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Could not parse json. Original message was: %s", smsg)
		}
		return
	}
	if viper.GetBool(misc.DebugKey) {
		log.Printf("The posted command is './run.sh %s'", rgp.FullCommand)
		log.Printf("Parsed command struct %v", rgp)
		log.Printf("Command computed hash %s", hash)
	}
	envVars := make(map[string]string)
	//Explicitly Disable progress bar fo tfResDif
	envVars["TFRESDIF_NOPB"] = "true"
	//Explicitly disable notification of tfChek to avoid endless loop
	envVars["NOTIFY_TFCHEK"] = "false"
	cmd, err := rgp.GetHashedCommand(hash)
	if err != nil {
		em := fmt.Sprintf("Cannot create background task. Error: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, e := w.Write([]byte(em))
		if e != nil {
			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
		}
		return
	}
	bt, err := submitCommand(cmd, &envVars, rgp.GetTimeout())
	if err != nil {
		em := fmt.Sprintf("Cannot create background task. Error: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, e := w.Write([]byte(em))
		if e != nil {
			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
		}
	} else {
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(strconv.Itoa(bt.GetId())))
		if err != nil {
			log.Printf("Cannot write response. Error: %s", err)
		}
	}

}

////Deprecated
//func RunShEnv(w http.ResponseWriter, r *http.Request) {
//	v := mux.Vars(r)
//	env := v["Env"]
//	layer := ""
//	envVars := make(map[string]string)
//	envVars["TFRESDIF_NOPB"] = "true"
//	getRunsh(w, r, env, layer, time.Duration(viper.GetInt(misc.TimeoutKey))*time.Second, &envVars)
//}

func Cancel(w http.ResponseWriter, r *http.Request) {
	tm := launcher.GetTaskManager()
	vars := mux.Vars(r)
	id := vars["Id"]
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

//Deprecated
//func RunShEnvLayer(w http.ResponseWriter, r *http.Request) {
//	v := mux.Vars(r)
//	env := v["Env"]
//	layer := v["Layer"]
//	envVars := make(map[string]string)
//	envVars["TFRESDIF_NOPB"] = "true"
//	getRunsh(w, r, env, layer, time.Duration(viper.GetInt(misc.TimeoutKey))*time.Second, &envVars)
//}

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
				log.Printf("Cannot match branch name %s against regex", branchName)
			}
			if matched {
				misc.Debug("This event is eligible for further processing")
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

					if gaTask, ok := task.(launcher.GitHubAwareTask); ok {
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

func handleReqErr(err error, w http.ResponseWriter) {
	if err != nil {
		errmsg := fmt.Sprintf("Cannot read request body. Error: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(errmsg))
		if err != nil {
			log.Printf("Cannot write a server response \"%s\". Error %s", errmsg, err)
		}
	}
}

func GetTaskIdByHash(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	hash := v["Hash"]
	w.Header().Set("Content-Type", "application/json")
	tm := launcher.GetTaskManager()
	tid, err := tm.GetId(hash)
	if err != nil {
		em := fmt.Sprintf("Cannot find task id by hash %s. Error %s", hash, err.Error())
		log.Print(em)
		w.WriteHeader(http.StatusNotFound)
		_, e := w.Write([]byte(em))
		if e != nil {
			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(strconv.Itoa(tid)))
		if err != nil {
			log.Printf("Cannot write response. Error: %s", err)
		}
	}
}

//Deprecated
//func getRunsh(w http.ResponseWriter, r *http.Request, env, layer string, timeout time.Duration, envVars *map[string]string) {
//	w.Header().Set("Content-Type", "application/json")
//	var cmd launcher.RunShCmd
//	err := r.ParseForm()
//	if err != nil {
//		em := fmt.Sprintf("Cannot parse request. Error: %s", err.Error())
//		_, e := w.Write([]byte(em))
//		if e != nil {
//			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
//		}
//	}
//	targets := r.Form["target"]
//	omit := r.FormValue("omit")
//	all := r.Form.Get("all")
//	no := r.Form.Get("no")
//	yes := r.Form.Get("yes")
//	startTime := time.Now()
//	cmd = launcher.RunShCmd{Layer: layer, Env: env, All: all == "true", Omit: omit == "true", Targets: targets, No: no == "true", Yes: yes == "true", Started: &startTime}
//
//	bt, err := submitCommand(&cmd, envVars, timeout)
//
//	if err != nil {
//		em := fmt.Sprintf("Cannot create background task. Error: %s", err.Error())
//		w.WriteHeader(http.StatusInternalServerError)
//		_, e := w.Write([]byte(em))
//		if e != nil {
//			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
//		}
//	} else {
//		w.WriteHeader(http.StatusCreated)
//		_, err = w.Write([]byte(strconv.Itoa(bt.GetId())))
//		if err != nil {
//			log.Printf("Cannot write response. Error: %s", err)
//		}
//	}
//}

func submitCommand(cmd *launcher.RunShCmd, envVars *map[string]string, timeout time.Duration) (launcher.Task, error) {
	tm := launcher.GetTaskManager()
	ctx, cancel := context.WithTimeout(
		context.WithValue(
			context.Background(),
			misc.EnvVarsKey, envVars),
		timeout)
	bt, err := tm.AddRunSh(cmd, ctx)
	if err != nil {
		return bt, err
	} else {
		if viper.GetBool("debug") {
			log.Printf("Task %d has been added", bt.GetId())
		}
	}
	if bt != nil {
		err = tm.RegisterCancel(bt.GetId(), cancel)
		if err != nil {
			log.Printf("ERROR! Cannot register cancel function. Task (id: %d) will be impossible to cancel. Error: %s", bt.GetId(), err)
			return nil, err
		}
		return bt, nil
	} else {
		log.Print("ERROR! Cannot register cancel function of nil task. This should never happen!")
		return nil, errors.New("Cannot register cancel function of nil task. This should never happen!")
	}
}
