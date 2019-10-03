package main

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

const (
	STATICDIR = "/static/"
	PORT      = "8085"
)

var tm launcher.TaskManager

func runShWs(w http.ResponseWriter, r *http.Request) {
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
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Cannot find task by id: %d", taskId)))
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

func apiRSEL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var cmd launcher.RunShCmd
	v := mux.Vars(r)
	env := v["Env"]
	layer := v["Layer"]
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
	envVars := make(map[string]string)
	envVars["TFRESDIF_NOPB"] = "true"
	ctx, cancel := context.WithTimeout(
		context.WithValue(
			context.WithValue(
				context.Background(),
				launcher.WD, "/tmp/production_42"),
			launcher.ENVVARS, envVars),
		60*time.Second)
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
	w.Write([]byte(strconv.Itoa(bt.GetId())))
}
func setupRoutes() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/ws/runsh/{id}", runShWs).Methods("GET")
	router.Path("/api/v1/runsh/{Env}/{Layer}").Methods("GET").Name("Env/Layer").HandlerFunc(apiRSEL)
	router.PathPrefix(STATICDIR).Handler(http.StripPrefix(STATICDIR, http.FileServer(http.Dir("."+STATICDIR))))
	router.PathPrefix("/").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "./static/index.html")
	})
	return router

}

func main() {
	tm = launcher.NewTaskManager()
	fmt.Println("Starting task manager")
	go tm.Start()
	defer tm.Close()
	fmt.Println("Starting server")
	router := setupRoutes()
	log.Fatal(http.ListenAndServe(":"+PORT, router))
}
