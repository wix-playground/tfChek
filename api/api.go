package api

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"tfChek/launcher"
	"time"
)

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
	bt := tm.GetTask(taskId)
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
	bt, err := tm.TaskOfRunSh(cmd, ctx)
	tm.RegisterCancel(bt, cancel)
	if err != nil {
		em := fmt.Sprintf("Cannot create background task. Error: %s", err.Error())
		_, e := w.Write([]byte(em))
		if e != nil {
			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
		}
	}
	err = tm.Launch(bt)
	if err != nil {
		em := fmt.Sprintf("Cannot launch background task. Error: %s", err.Error())
		w.WriteHeader(505)
		_, e := w.Write([]byte(em))
		if e != nil {
			log.Printf("Cannot respond with message '%s' Error: %s", err, e)
		}
	}
	w.WriteHeader(202)
	_, err = w.Write([]byte(strconv.Itoa(bt.GetId())))
	if err != nil {
		log.Printf("Cannot write response. Error: %s", err)
	}
}
