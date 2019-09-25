package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"tfChek/launcher"
	"time"
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
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), launcher.WD, "/Users/maksymsh/terraform/production_42/generator/output/100/logs"), 60*time.Second)
	defer cancel()
	commands := make(map[string][]string)
	commands["init"] = []string{"../../../../bin/terraform", "init", "-force-copy"}
	commands["plan"] = []string{"../../../../bin/terraform", "plan", "-lock-timeout=1200s", "-out=terraform.plan"}
	//go launcher.RunCommand(w,ctx,"../../../../bin/terraform", "init", "-force-copy")
	go launcher.RunCommands(w, ctx, &commands)
	defer w.Close()
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
