package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"tfChek/launcher"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var disp launcher.Dispatcher

//Deprecated
func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "Hello world!")
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client connected")
	err = ws.WriteMessage(1, []byte("Hi, client!"))
	if err != nil {
		log.Println(err)
	}
	//reader(ws)
	processAdapter(ws)
}

func runShEnvWs(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client connected to run.sh Env websocket")
	err = ws.WriteMessage(1, []byte("Ready to run run.sh"))
	if err != nil {
		log.Println(err)
	}
	uri := r.RequestURI
	log.Printf("URI: %s", uri)
}

//Deprecated
func reader(conn *websocket.Conn) {
	messageType, p, err := conn.ReadMessage()
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(p))
	if err = conn.WriteMessage(messageType, p); err != nil {
		log.Println(err)
		return
	}
}
func processAdapter(conn *websocket.Conn) {
	disp.Launch(conn, strconv.Itoa(100), "logs", []string{"./run.sh", "-n", "100/logs"})
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
	log.Println(cmd)
	launcher.LaunchRunSh()
}
func setupRoutes() {
	router := mux.NewRouter()
	router.Handle("/", http.FileServer(http.Dir("static")))
	router.HandleFunc("/ws", wsEndpoint).Methods("GET")
	router.HandleFunc("/ws/runsh/{Env}", runShEnvWs).Methods("GET")
	//router.Path("/api/v1/runsh/{Env}").HandlerFunc(apiRSE).Methods("GET").Name("Env")
	router.Path("/api/v1/runsh/{Env}/{Layer}").Methods("GET").Name("Env/Layer").HandlerFunc(apiRSEL)
	http.Handle("/", router)
}

func main() {
	log.Println("Starting launcher...")
	disp = launcher.NewDispatcher()
	go disp.Start()
	defer disp.Close()
	fmt.Println("Starting server")
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8085", nil))
}
