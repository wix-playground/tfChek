package api

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"tfChek/launcher"
	"time"
)
import "gopkg.in/go-playground/webhooks.v5/github"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func RunShWebsocket(w http.ResponseWriter, r *http.Request) {
	tm := launcher.GetTaskManager()
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		log.Println("Cannot run with no id")
	}
	taskId, err := strconv.Atoi(id)
	if err != nil {
		log.Println("Cannot convert parse task id")
	}
	bt := tm.Get(taskId)
	if bt == nil {
		log.Printf("Cannot find task by id: %d", taskId)
		err := ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Cannot find task by id: %d", taskId)))
		if err != nil {
			log.Printf("Cannot find task by id: %d Error: %s", taskId, err)
		}
		w.WriteHeader(404)
		return
	}

	log.Println("Client connected to run.sh Env websocket")
	errc := make(chan error)
	err = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Task (id: %d) status is %d", bt.GetId(), bt.GetStatus())))
	if err != nil {
		log.Println(err)
	}
	lock := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go writeToWS(bt.GetStdOut(), ws, errc, lock, wg)
	go writeToWS(bt.GetStdErr(), ws, errc, lock, wg)
	go func(ws *websocket.Conn, errc <-chan error) {
		err = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Task (id: %d) status is %d", bt.GetId(), bt.GetStatus())))
		if err != nil {
			log.Println(err)
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
	runsh(w, r, env, layer, "/tmp/production_42", 60*time.Second, &envVars)
}

func RunShEnvLayer(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	env := v["Env"]
	layer := v["Layer"]
	envVars := make(map[string]string)
	envVars["TFRESDIF_NOPB"] = "true"
	runsh(w, r, env, layer, "/tmp/production_42", 60*time.Second, &envVars)
}

func RunShWebHook(w http.ResponseWriter, r *http.Request) {
	tm := launcher.GetTaskManager()
	//hook, _ := github.New(github.Options.Secret("MyGitHubSuperSecretSecrect...?"))
	hook, _ := github.New()
	payload, err := hook.Parse(r, github.ReleaseEvent, github.PullRequestEvent, github.CreateEvent)
	if err != nil {
		if err == github.ErrEventNotFound {
			// ok event wasn;t one of the ones asked to be parsed
			errmsg := fmt.Sprintf("Unknown event. Error: %s", err)
			log.Println(errmsg)
			w.WriteHeader(404)
			w.Write([]byte(errmsg))
		} else {
			errmsg := fmt.Sprintf("Got error %s", err)
			log.Println(errmsg)
			w.WriteHeader(400)
			w.Write([]byte(errmsg))
		}
	}

	switch payload.(type) {
	case github.ReleasePayload:
		release := payload.(github.ReleasePayload)
		// Do whatever you want from here...
		fmt.Printf("%+v", release)

	case github.PullRequestPayload:
		pullRequest := payload.(github.PullRequestPayload)
		// Do whatever you want from here...
		fmt.Printf("%+v", pullRequest)
	case github.CreatePayload:
		createRequest := payload.(github.CreatePayload)
		//fmt.Printf("%+v", createRequest)
		fmt.Println(createRequest.Repository.Name)
		fmt.Println(createRequest.Ref)
		fmt.Println(createRequest.RefType)
		fmt.Println(createRequest.Sender.Login)
		fmt.Println(createRequest.Sender.Type)
		fmt.Println(createRequest.PusherType)
		//TODO: perform regexp validation
		switch createRequest.RefType {
		case "tag":
			tagChunks := strings.Split(createRequest.Ref, "-")
			if len(tagChunks) != 2 {
				errmsg := fmt.Sprintf("Cannot parse tag name %s", createRequest.Ref)
				log.Println(errmsg)
				w.WriteHeader(400)
				w.Write([]byte(errmsg))
			} else {
				if tagChunks[0] != "tfci" {
					errmsg := fmt.Sprintf("Unsupported tag format %s. Shold be in form of /tfci-[0-9]+/", createRequest.Ref)
					log.Println(errmsg)
					w.WriteHeader(400)
					w.Write([]byte(errmsg))
				} else {
					id, err := strconv.Atoi(tagChunks[1])
					if err != nil {
						errmsg := fmt.Sprintf("Cannot parse task id %s", tagChunks[1])
						log.Println(errmsg)
						w.WriteHeader(400)
						w.Write([]byte(errmsg))
					} else {
						err := tm.LaunchById(id)
						if err != nil {
							errmsg := fmt.Sprintf("Cannot launch task id %d. Error: %s", id, err)
							log.Println(errmsg)
							w.WriteHeader(400)
							w.Write([]byte(errmsg))
						} else {
							w.WriteHeader(202)
						}
					}
				}
			}
		}
	}
}

func runsh(w http.ResponseWriter, r *http.Request, env, layer, workDir string, timeout time.Duration, envVars *map[string]string) {
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
			context.WithValue(
				context.Background(),
				launcher.WD, workDir),
			launcher.ENVVARS, envVars),
		timeout)
	bt, err := tm.AddRunSh(cmd, ctx)
	if err != nil {
		em := fmt.Sprintf("Cannot create background task. Error: %s", err.Error())
		_, e := w.Write([]byte(em))
		if e != nil {
			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
		}
	} else {
		if viper.GetBool("debug") {
			log.Printf("Task %d has been added", bt.GetId())
		}
	}
	tm.RegisterCancel(bt.GetId(), cancel)
	//err = tm.Launch(bt)
	//if err != nil {
	//	em := fmt.Sprintf("Cannot launch background task. Error: %s", err.Error())
	//	w.WriteHeader(505)
	//	_, e := w.Write([]byte(em))
	//	if e != nil {
	//		log.Printf("Cannot respond with message '%s' Error: %s", err, e)
	//	}
	//}
	w.WriteHeader(201)
	_, err = w.Write([]byte(strconv.Itoa(bt.GetId())))
	if err != nil {
		log.Printf("Cannot write response. Error: %s", err)
	}
}
