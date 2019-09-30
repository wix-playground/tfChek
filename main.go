package main

import (
	"fmt"
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

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Home page!")
}

func runEnvironment(w http.ResponseWriter, r *http.Request) {

}

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

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
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
