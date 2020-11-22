package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	messageTypeData   = "DATA"
	messageTypeCursor = "CURSOR"
	messageTypeURL    = "URL"
)

type message struct {
	// the json tag means this will serialize as a lowercased field
	Type    string `json:"type"`
	Content string `json:"content"`
}

var sessions map[string]chan chan message

func sendMessage(w *wsutil.Writer, encoder *json.Encoder, msg message) bool {
	if err := encoder.Encode(&msg); err != nil {
		fmt.Println("Encode err:", err)
		return false
	}

	if err := w.Flush(); err != nil {
		fmt.Println("Flush err:", err)
		return false
	}

	return true
}

func sessionLog(uuid string, args ...string) {
	fmt.Println(fmt.Sprintf("[%s]", uuid), strings.Join(args, " "))
}

func joinURL(uuid string, ws bool) string {
	var (
		protocol = "http"
		route    = "join"
	)
	if ws {
		protocol = "ws"
		route = "ws"
	}

	return fmt.Sprintf("%s://localhost:8080/%s/%s", protocol, route, uuid)
}

func create(w http.ResponseWriter, r *http.Request) {
	serverChan := make(chan chan message, 4)

	uuid := uuid.New().String()
	sessions[uuid] = serverChan

	sessionLog(uuid, "Creating")

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}
	go func() {
		defer conn.Close()

		var (
			state = ws.StateServerSide
			r     = wsutil.NewReader(conn, state)
			w     = wsutil.NewWriter(conn, state, ws.OpText)

			decoder = json.NewDecoder(r)
			encoder = json.NewEncoder(w)

			clients []chan message
		)

		// Send UUID join URL
		sendMessage(w, encoder, message{messageTypeURL, joinURL(uuid, false)})

		go func() {
			for {
				// Listen for new clients
				client := <-serverChan
				clients = append(clients, client)
				sessionLog(uuid, "Detected new client")
			}
		}()

		for {
			// Await some data from main code server
			hdr, err := r.NextFrame()
			if err != nil {
				// no data, skip
				continue
			}

			if hdr.OpCode == ws.OpClose {
				// Close listening clients somehow?
				return
			}

			var msg message
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF {
					return
				}
				fmt.Println("Decode err:", err)
				continue
			}

			fmt.Println("Received:", msg)
			for _, c := range clients {
				c <- msg
			}
		}
	}()
}

func join(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	_, ok := sessions[uuid]
	if !ok {
		sessionLog(uuid, "No session found")
		w.Write([]byte(fmt.Sprintf("No session found for ID %s", uuid)))
		return
	}

	t, err := template.ParseFiles("./templates/join.html")
	if err != nil {
		fmt.Println("Template parse err:", err)
		w.Write([]byte("An unexpected error occurred"))
	}

	t.Execute(w, joinURL(uuid, true))
}

func joinWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	serverChan, ok := sessions[uuid]
	if !ok {
		sessionLog(uuid, "No session found")
		w.Write([]byte(fmt.Sprintf("[%s] No session found", uuid)))
		return
	}

	sessionLog(uuid, "Joining")

	client := make(chan message, 1)
	serverChan <- client

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		// handle error
		fmt.Println(err)
		return
	}

	go func() {
		defer conn.Close()

		var (
			state = ws.StateServerSide
			// r     = wsutil.NewReader(conn, state)
			w = wsutil.NewWriter(conn, state, ws.OpText)

			// decoder = json.NewDecoder(r)
			encoder = json.NewEncoder(w)
		)

		// Wait for new data to send indefinitely
		for {
			msg := <-client
			sendMessage(w, encoder, msg)
		}
	}()
}

func main() {
	sessions = make(map[string]chan chan message)

	router := mux.NewRouter()
	router.HandleFunc("/create", create)
	router.HandleFunc("/join/{uuid}", join)
	router.HandleFunc("/ws/{uuid}", joinWS)

	// Setup file server
	fs := http.FileServer(http.Dir("public"))
	router.PathPrefix("/public").Handler(http.StripPrefix("/public/", fs))

	http.Handle("/", router)
	http.ListenAndServe(":8080", nil)
}
