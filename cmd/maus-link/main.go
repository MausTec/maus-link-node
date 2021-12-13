package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"flag"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var deviceMap = make(map[string]*websocket.Conn)
var clientMap = make(map[string][]*websocket.Conn)

type ConnectPayload struct {
	DeviceKey string `json:"deviceKey"`
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

func reader(conn *websocket.Conn, id string) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println(string(p))

		if clientMap[id] != nil {
			for _, client := range clientMap[id] {
				if err := client.WriteMessage(messageType, p); err != nil {
					log.Println(err)
					return
				}
			}
		}
	}
}

func writer(conn *websocket.Conn, id string) {
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println(string(p))

		if deviceMap[id] != nil {
			if err := deviceMap[id].WriteMessage(messageType, p); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func clientEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	vars := mux.Vars(r)
	id, ok := vars["id"]

	if !ok || deviceMap[id] == nil {
		fmt.Println("id is missing")
		w.WriteHeader(404)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Client connected: %s\n", id)
	clientMap[id] = append(clientMap[id], ws)
	writer(ws, id)
}

func deviceEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	idBytes := make([]byte, 3)
	if _, err := rand.Read(idBytes); err != nil {
		log.Println(err)
		return
	}

	id := hex.EncodeToString(idBytes)

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Device connected: %s\n", id)
	deviceMap[id] = ws

	connectAck := ConnectPayload {
		DeviceKey: id,
	}

	err = ws.WriteJSON(connectAck)
	if err != nil {
		log.Println(err)
		return
	}

	reader(ws, id)
}

func setupRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/remote/{id}", clientEndpoint)
	r.HandleFunc("/device", deviceEndpoint)

	return r
}

func main() {
	fmt.Println("Starting server...")
	use_tls := flag.Bool("tls", false, "Use TLS")

	router := setupRoutes()

	if *use_tls {
		err := http.ListenAndServeTLS(":8084", "assets/development.crt", "assets/development.key", router)
		log.Fatal(err)
	} else {
		err := http.ListenAndServe(":8080", router)
		log.Fatal(err)
	}
}