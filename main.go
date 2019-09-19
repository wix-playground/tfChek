package main

import (
	"bufio"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"tfChek/launcher"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Home page!")
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

	r, w := io.Pipe()
	go launcher.Exmpl(w)
	reader := bufio.NewReader(r)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			err = conn.WriteMessage(websocket.TextMessage, line)
			if err != nil {
				log.Println(err)
				return
			}
			break
		}
		err = conn.WriteMessage(websocket.TextMessage, line)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

func main() {
	fmt.Println("Starting server")
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8085", nil))
}
